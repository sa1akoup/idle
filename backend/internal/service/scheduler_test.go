// 调度器生命周期测试：确认后台 worker 可以停止、等待并在同一进程内重新启动。
package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"idle/internal/repository/database"

	"gorm.io/gorm"
)

func TestSessionSchedulerCanRestartAndStop(t *testing.T) {
	dsn := fmt.Sprintf("file:scheduler-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := database.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := database.Migrate(db, "sqlite"); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}

	scheduler := NewSessionScheduler(db)
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("首次启动 scheduler: %v", err)
	}
	if err := scheduler.Start(context.Background()); !errors.Is(err, ErrSessionSchedulerStarted) {
		t.Fatalf("重复启动 scheduler 错误 = %v，期望 ErrSessionSchedulerStarted", err)
	}
	scheduler.Stop()
	scheduler.Wait()
	scheduler.Stop()
	scheduler.Wait()

	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("同一实例重新启动 scheduler: %v", err)
	}
	scheduler.Stop()
	scheduler.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("使用 context 启动 scheduler: %v", err)
	}
	cancel()
	scheduler.Wait()
	scheduler.Stop()
	scheduler.Wait()
}

func newStartedTestScheduler(t *testing.T, db *gorm.DB) *SessionScheduler {
	t.Helper()
	scheduler := NewSessionScheduler(db)
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("启动测试 scheduler: %v", err)
	}
	t.Cleanup(func() {
		scheduler.Stop()
		scheduler.Wait()
	})
	return scheduler
}
