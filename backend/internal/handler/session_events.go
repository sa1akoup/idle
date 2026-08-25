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

	pollTicker := time.NewTicker(time.Second)
	heartbeatTicker := time.NewTicker(20 * time.Second)
	defer pollTicker.Stop()
	defer heartbeatTicker.Stop()

	for {
		events, err := service.ListSessionEvents(h.db, userID(c), sessionID, cursor)
		if err != nil {
			return
		}
		for _, event := range events {
			if err := writeSessionEvent(c, event); err != nil {
				return
			}
			cursor = event.ID
		}

		var state struct {
			Status  string     `json:"status"`
			EndTime *time.Time `json:"endTime"`
		}
		if err := h.db.Model(&models.Session{}).Select("status, end_time").Where("user_id = ? AND id = ?", userID(c), sessionID).First(&state).Error; err != nil {
			return
		}
		if len(events) == 0 && (state.Status == "finished" || state.Status == "aborted" || state.Status == "failed") {
			endPayload, marshalErr := json.Marshal(map[string]string{"status": state.Status})
			if marshalErr != nil {
				return
			}
			_, _ = fmt.Fprintf(c.Writer, "event: stream_end\ndata: %s\n\n", endPayload)
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
