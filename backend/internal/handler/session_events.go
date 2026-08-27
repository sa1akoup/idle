// Session 事件接口：提供历史事件读取和可断线重连的 SSE 实时推送。
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"idle/internal/models"
	"idle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// sseWriteTimeout 是单次响应写的最大阻塞时长；客户端半开连接时，写操作在截止时间后必然失败并释放 goroutine。
const sseWriteTimeout = 10 * time.Second

func isTerminalSessionStatus(status string) bool {
	switch status {
	case "success", "incapacitated", "failed":
		return true
	}
	return false
}

func parseSessionEventCursor(c *gin.Context) (uint, error) {
	value := c.Query("afterId")
	if value == "" {
		value = c.GetHeader("Last-Event-ID")
	}
	if value == "" {
		return 0, nil
	}
	cursor, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("事件游标无效")
	}
	return uint(cursor), nil
}

func parseSessionID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("行动编号无效")
	}
	return uint(id), nil
}

func (h *Handler) ListSessionEvents(c *gin.Context) {
	sessionID, err := parseSessionID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	afterID, err := parseSessionEventCursor(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	events, err := service.ListSessionEvents(h.db, userID(c), sessionID, afterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "行动不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

func writeSessionEvent(c *gin.Context, event service.SessionEventView) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "id: %d\nevent: session_event\ndata: %s\n\n", event.ID, payload); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func (h *Handler) StreamSessionEvents(c *gin.Context) {
	sessionID, err := parseSessionID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cursor, err := parseSessionEventCursor(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := service.ListSessionEvents(h.db, userID(c), sessionID, cursor); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "行动不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	rw := http.NewResponseController(c.Writer)
	pollTicker := time.NewTicker(time.Second)
	heartbeatTicker := time.NewTicker(20 * time.Second)
	defer pollTicker.Stop()
	defer heartbeatTicker.Stop()

	var state struct {
		Status  string     `json:"status"`
		EndTime *time.Time `json:"endTime"`
	}
	pollCount := 0
	for {
		events, err := service.ListSessionEvents(h.db, userID(c), sessionID, cursor)
		if err != nil {
			return
		}
		_ = rw.SetWriteDeadline(time.Now().Add(sseWriteTimeout))
		for _, event := range events {
			if err := writeSessionEvent(c, event); err != nil {
				return
			}
			cursor = event.ID
		}

		// 首轮和有新事件后立即读取会话状态；空闲等待时降频为每 5 个轮询周期一次，
		// 避免每个连接每秒固定产生两条数据库查询。
		if pollCount == 0 || len(events) > 0 || pollCount%5 == 4 {
			state.Status = ""
			if err := h.db.Model(&models.Session{}).Select("status, end_time").Where("user_id = ? AND id = ?", userID(c), sessionID).First(&state).Error; err != nil {
				return
			}
		}
		pollCount++
		if len(events) == 0 && isTerminalSessionStatus(state.Status) {
			endPayload, marshalErr := json.Marshal(map[string]string{"status": state.Status})
			if marshalErr != nil {
				return
			}
			if _, err := fmt.Fprintf(c.Writer, "event: stream_end\ndata: %s\n\n", endPayload); err != nil {
				return
			}
			c.Writer.Flush()
			return
		}

		select {
		case <-c.Request.Context().Done():
			return
		case <-pollTicker.C:
		case <-heartbeatTicker.C:
			if _, err := fmt.Fprint(c.Writer, ": heartbeat\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}
