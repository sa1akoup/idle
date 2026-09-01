// 行动调度器：负责启动新行动，并在服务重启后恢复数据库中未完成的行动。
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"idle/internal/models"

	"gorm.io/gorm"
)

const (
	sessionDispatchBatchSize = 32
	sessionWorkerPoolSize    = 4
	sessionWorkerQueueSize   = sessionDispatchBatchSize * 2
)

var ErrSessionLeaseLost = errors.New("行动调度 lease 已失效")

var ErrSessionSchedulerStarted = errors.New("行动调度器已启动")

var ErrSessionSchedulerNotConfigured = errors.New("行动调度器未配置")

type sessionWorkerTask struct {
	db        *gorm.DB
	userID    uint
	sessionID uint
}

// SessionScheduler 持有单个进程内的行动调度状态，生命周期由应用入口显式管理。
type SessionScheduler struct {
	db      *gorm.DB
	mu      sync.Mutex
	runtime *schedulerRuntime
}

type schedulerRuntime struct {
	queue        chan sessionWorkerTask
	stop         chan struct{}
	loopDone     chan struct{}
	waitComplete chan struct{}
	workers      sync.WaitGroup
	stopOnce     sync.Once
	waitOnce     sync.Once
	activeWorker sync.Map
	stopping     bool
}

// NewSessionScheduler 创建调度器实例，仅持有数据库连接，运行状态由 Start 初始化。
func NewSessionScheduler(db *gorm.DB) *SessionScheduler {
	return &SessionScheduler{db: db}
}

// sessionWorkerKey 将用户 ID 与会话 ID 编码为 64 位键，用于去重与活跃标记。
func sessionWorkerKey(userID, sessionID uint) uint64 {
	return uint64(userID)<<32 | uint64(sessionID)
}

// Start 启动调度循环并立即派发当前已到期的行动。
func (s *SessionScheduler) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("行动调度器启动上下文不能为空")
	}
	if s.db == nil {
		return errors.New("行动调度器数据库连接不能为空")
	}

	s.mu.Lock()
	if s.runtime != nil {
		s.mu.Unlock()
		return ErrSessionSchedulerStarted
	}
	runtime := newSchedulerRuntime()
	s.runtime = runtime
	s.mu.Unlock()

	go s.run(ctx, runtime)
	s.dispatchPendingSessions()
	return nil
}

// run 调度主循环：每秒扫描一次到期会话并派发，收到停止信号或上下文取消时退出。
func (s *SessionScheduler) run(ctx context.Context, runtime *schedulerRuntime) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer close(runtime.loopDone)

	for {
		select {
		case <-ctx.Done():
			s.requestStop(runtime)
			return
		case <-runtime.stop:
			return
		case <-ticker.C:
			s.dispatchPendingSessions()
		}
	}
}

// requestStop 标记停止状态，并由 stopOnce 保证停止通道只关闭一次。
func (s *SessionScheduler) requestStop(runtime *schedulerRuntime) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtime != runtime {
		return
	}
	runtime.stopping = true
	runtime.stopOnce.Do(func() { close(runtime.stop) })
}

// Stop 仅发出停止信号；调用方通过 Wait 等待调度循环和 worker 完整退出。
func (s *SessionScheduler) Stop() {
	s.mu.Lock()
	runtime := s.runtime
	if runtime != nil {
		runtime.stopping = true
		runtime.stopOnce.Do(func() { close(runtime.stop) })
	}
	s.mu.Unlock()
}

// Wait 等待调度循环退出、队列排空和全部 worker 完成。
func (s *SessionScheduler) Wait() {
	s.mu.Lock()
	runtime := s.runtime
	s.mu.Unlock()
	if runtime == nil {
		return
	}

	runtime.waitOnce.Do(func() {
		<-runtime.loopDone

		s.mu.Lock()
		close(runtime.queue)
		s.mu.Unlock()
		runtime.workers.Wait()

		s.mu.Lock()
		if s.runtime == runtime {
			s.runtime = nil
		}
		s.mu.Unlock()
		close(runtime.waitComplete)
	})
	<-runtime.waitComplete
}

// dispatchSessionWorker 将一个行动加入当前实例的 worker 队列。
func (s *SessionScheduler) dispatchSessionWorker(userID, sessionID uint) {
	s.mu.Lock()
	runtime := s.runtime
	if runtime == nil || runtime.stopping {
		s.mu.Unlock()
		return
	}
	key := sessionWorkerKey(userID, sessionID)
	// 防重入：该会话已有 worker 在跑则直接跳过，避免同一会话被并发推进。
	if _, loaded := runtime.activeWorker.LoadOrStore(key, struct{}{}); loaded {
		s.mu.Unlock()
		return
	}
	select {
	case runtime.queue <- sessionWorkerTask{db: s.db, userID: userID, sessionID: sessionID}:
		s.mu.Unlock()
	default:
		// 队列已满：撤回去重标记，交由后续调度扫描重新派发。
		runtime.activeWorker.Delete(key)
		s.mu.Unlock()
		log.Printf("行动 worker 队列已满，稍后重试 session=%d user=%d", sessionID, userID)
	}
}

// isSessionWorkerActive 仅供同包测试确认启动后的异步派发已完成。
func (s *SessionScheduler) isSessionWorkerActive(userID, sessionID uint) bool {
	s.mu.Lock()
	runtime := s.runtime
	s.mu.Unlock()
	if runtime == nil {
		return false
	}
	_, active := runtime.activeWorker.Load(sessionWorkerKey(userID, sessionID))
	return active
}

// newSchedulerRuntime 初始化队列与停止通道，并启动固定数量的 worker 协程消费任务队列。
func newSchedulerRuntime() *schedulerRuntime {
	runtime := &schedulerRuntime{
		queue:        make(chan sessionWorkerTask, sessionWorkerQueueSize),
		stop:         make(chan struct{}),
		loopDone:     make(chan struct{}),
		waitComplete: make(chan struct{}),
	}
	for i := 0; i < sessionWorkerPoolSize; i++ {
		runtime.workers.Add(1)
		go func() {
			defer runtime.workers.Done()
			for task := range runtime.queue {
				runSessionWorker(runtime, task)
			}
		}()
	}
	return runtime
}

// runSessionWorker 单任务执行入口：先原子抢占租约，失败则直接放弃；成功则带租约推进会话并在结束或失败时释放。
func runSessionWorker(runtime *schedulerRuntime, task sessionWorkerTask) {
	key := sessionWorkerKey(task.userID, task.sessionID)
	defer runtime.activeWorker.Delete(key)
	workerID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	if !claimSessionLease(task.db, task.userID, task.sessionID, workerID, time.Now()) {
		return
	}
	service := NewSessionServiceWithLease(task.db, task.userID, workerID)
	defer service.releaseSessionLease(task.sessionID)
	service.runSession(task.sessionID)
}

// dispatchPendingSessions 扫描到期（next_run_at 已到且租约为空或已过期）的会话，按时间顺序批量派发，单次上限 sessionDispatchBatchSize 条。
func (s *SessionScheduler) dispatchPendingSessions() {
	var sessions []struct {
		ID     uint
		UserID uint
	}
	now := time.Now()
	if err := s.db.Table("sessions").
		Select("id, user_id").
		Where("status = ? AND next_run_at <= ? AND (lease_until IS NULL OR lease_until <= ?)", "running", now, now).
		Order("next_run_at ASC, id ASC").
		Limit(sessionDispatchBatchSize).
		Find(&sessions).Error; err != nil {
		log.Printf("查询到期行动失败: %v", err)
		return
	}
	for _, session := range sessions {
		s.dispatchSessionWorker(session.UserID, session.ID)
	}
}

// claimSessionLease 原子抢占会话租约：仅当会话仍运行、已到期且租约可用时写入归属并延到 30 秒后，成功才允许该 worker 推进。
func claimSessionLease(db *gorm.DB, userID, sessionID uint, owner string, now time.Time) bool {
	leaseUntil := now.Add(30 * time.Second)
	result := db.Model(&models.Session{}).
		Where("user_id = ? AND id = ? AND status = ? AND next_run_at <= ? AND (lease_until IS NULL OR lease_until <= ?)", userID, sessionID, "running", now, now).
		Updates(map[string]interface{}{"lease_owner": owner, "lease_until": leaseUntil, "heartbeat_at": now})
	return result.Error == nil && result.RowsAffected == 1
}
