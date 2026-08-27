// 藏身处路由：读取设施总览、启动设施升级和提交护甲维修作业。
package handler

import (
	"net/http"
	"strconv"

	"idle/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetHideout(c *gin.Context) {
	hideout, err := service.GetHideoutForUser(h.db, userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, hideout)
}

func (h *Handler) StartHideoutUpgrade(c *gin.Context) {
	if err := service.StartFacilityUpgradeForUser(h.db, userID(c), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (h *Handler) QueueHideoutRepair(c *gin.Context) {
	var req struct {
		ArmorInstanceID uint `json:"armorInstanceId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "护甲实例不能为空"})
		return
	}
	if err := service.QueueArmorRepairForUser(h.db, userID(c), req.ArmorInstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (h *Handler) ToggleGenerator(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "发电机开关状态无效"})
		return
	}
	if err := service.ToggleGeneratorForUser(h.db, userID(c), req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) LoadGeneratorFuel(c *gin.Context) {
	instanceID, err := strconv.ParseUint(c.Query("instanceId"), 10, 64)
	if err != nil || instanceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "燃料实例不能为空"})
		return
	}
	if err := service.LoadGeneratorFuelForUser(h.db, userID(c), uint(instanceID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) UnloadGeneratorFuel(c *gin.Context) {
	instanceID, err := strconv.ParseUint(c.Query("instanceId"), 10, 64)
	if err != nil || instanceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "燃料实例不能为空"})
		return
	}
	if err := service.UnloadGeneratorFuelForUser(h.db, userID(c), uint(instanceID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
