// 行动调度器：负责启动新行动，并在服务重启后恢复数据库中未完成的行动。
package service

import (
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

var (
	activeSessionWorkers sync.Map
	sessionWorkerQueue   = make(chan sessionWorkerTask, sessionWorkerQueueSize)
	sessionWorkerPool    sync.Once
)

var ErrSessionLeaseLost = errors.New("行动调度 lease 已失效")

type sessionWorkerTask struct {
	db        *gorm.DB
	userID    uint
	sessionID uint
}

func sessionWorkerKey(userID, sessionID uint) uint64 {
	return uint64(userID)<<32 | uint64(sessionID)
}

func dispatchSessionWorker(db *gorm.DB, userID, sessionID uint) {
	startSessionWorkerPool()
	key := sessionWorkerKey(userID, sessionID)
	if _, loaded := activeSessionWorkers.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	select {
	case sessionWorkerQueue <- sessionWorkerTask{db: db, userID: userID, sessionID: sessionID}:
	default:
		activeSessionWorkers.Delete(key)
		log.Printf("行动 worker 队列已满，稍后重试 session=%d user=%d", sessionID, userID)
	}
}

func startSessionWorkerPool() {
	sessionWorkerPool.Do(func() {
		for i := 0; i < sessionWorkerPoolSize; i++ {
			go func() {
				for task := range sessionWorkerQueue {
					runSessionWorker(task)
				}
			}()
		}
	})
}

func runSessionWorker(task sessionWorkerTask) {
	key := sessionWorkerKey(task.userID, task.sessionID)
	defer activeSessionWorkers.Delete(key)
	workerID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	if !claimSessionLease(task.db, task.userID, task.sessionID, workerID, time.Now()) {
		return
	}
	service := NewSessionServiceWithLease(task.db, task.userID, workerID)
	defer service.releaseSessionLease(task.sessionID)
	service.runSession(task.sessionID)
}

// StartSessionScheduler 启动后台调度器，并恢复数据库中的未完成行动。
func StartSessionScheduler(db *gorm.DB) {
	startSessionWorkerPool()
	dispatchPendingSessions(db)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			dispatchPendingSessions(db)
		}
	}()
}

func dispatchPendingSessions(db *gorm.DB) {
	var sessions []struct {
		ID     uint
		UserID uint
	}
	now := time.Now()
	if err := db.Table("sessions").
		Select("id, user_id").
		Where("status IN ? AND next_run_at <= ? AND (lease_until IS NULL OR lease_until <= ?)", []string{"running", "waiting_injury"}, now, now).
		Order("next_run_at ASC, id ASC").
		Limit(sessionDispatchBatchSize).
		Find(&sessions).Error; err != nil {
		log.Printf("查询到期行动失败: %v", err)
		return
	}
	for _, session := range sessions {
		dispatchSessionWorker(db, session.UserID, session.ID)
	}
}

func claimSessionLease(db *gorm.DB, userID, sessionID uint, owner string, now time.Time) bool {
	leaseUntil := now.Add(30 * time.Second)
	result := db.Model(&models.Session{}).
		Where("user_id = ? AND id = ? AND status IN ? AND next_run_at <= ? AND (lease_until IS NULL OR lease_until <= ?)", userID, sessionID, []string{"running", "waiting_injury"}, now, now).
		Updates(map[string]interface{}{"lease_owner": owner, "lease_until": leaseUntil, "heartbeat_at": now})
	return result.Error == nil && result.RowsAffected == 1
}
