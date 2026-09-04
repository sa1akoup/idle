package handler

import (
	"errors"
	"net/http"
	"strings"

	"idle/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListQuests(c *gin.Context) {
	list, err := service.ListQuestsForUser(h.db, userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) AcceptQuest(c *gin.Context) {
	questID := strings.TrimSpace(c.Param("id"))
	if questID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "合同编号无效"})
		return
	}
	if err := service.AcceptQuestForUser(h.db, userID(c), questID); err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, service.ErrQuestUnavailable) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	list, err := service.ListQuestsForUser(h.db, userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) TurnInQuest(c *gin.Context) {
	questID := strings.TrimSpace(c.Param("id"))
	if questID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "合同编号无效"})
		return
	}
	if err := service.TurnInQuestForUser(h.db, userID(c), questID); err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, service.ErrQuestUnavailable) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	list, err := service.ListQuestsForUser(h.db, userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}
