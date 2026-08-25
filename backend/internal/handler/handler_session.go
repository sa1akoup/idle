// 行动与地图路由：处理节点、敌人、Session 查询、中止和护甲维修。
package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"idle/internal/models"
	"idle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) ListNodes(c *gin.Context) {
	var list []models.NodeDef
	if err := h.db.Order("route_order asc, id asc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "节点数据读取失败"})
		return
	}
	ids := make([]string, 0, len(list))
	for _, node := range list {
		ids = append(ids, node.ID)
	}
	var assignments []models.NodeContainerDef
	if err := h.db.Where("node_id IN ?", ids).Order("id asc").Find(&assignments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "节点容器读取失败"})
		return
	}
	var containers []models.LootContainerDef
	if err := h.db.Order("id asc").Find(&containers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "容器定义读取失败"})
		return
	}
	containerByID := make(map[string]models.LootContainerDef, len(containers))
	for _, container := range containers {
		containerByID[container.ID] = container
	}
	type containerView struct {
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		Pool       string   `json:"pool"`
		Tags       []string `json:"tags"`
		ValueTier  int      `json:"valueTier"`
		Weight     int      `json:"weight"`
		SearchRisk int      `json:"searchRisk"`
		SearchTime int      `json:"searchTime"`
		Count      int      `json:"count"`
	}
	byNodeID := make(map[string][]containerView, len(list))
	for _, assignment := range assignments {
		container := containerByID[assignment.ContainerID]
		pool := assignment.Pool
		if pool == "" {
			pool = models.NodeContainerPoolSearch
		}
		byNodeID[assignment.NodeID] = append(byNodeID[assignment.NodeID], containerView{
			ID: assignment.ContainerID, Name: container.Name, Pool: pool, Tags: container.Tags, ValueTier: container.ValueTier,
			Weight: assignment.Weight, SearchRisk: container.SearchRisk, SearchTime: container.SearchTime, Count: assignment.Count,
		})
	}
	type nodeView struct {
		models.NodeDef
		Containers []containerView `json:"containers"`
	}
	result := make([]nodeView, 0, len(list))
	for _, node := range list {
		containersForNode := byNodeID[node.ID]
		if containersForNode == nil {
			containersForNode = []containerView{}
		}
		result = append(result, nodeView{NodeDef: node, Containers: containersForNode})
	}
	c.JSON(http.StatusOK, result)
}
func (h *Handler) ListEnemies(c *gin.Context) {
	var list []models.EnemyDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "敌人数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) StartSession(c *gin.Context) {
	var req service.StartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	sess, err := h.sessionService(c).Start(req)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, sess)
}
func (h *Handler) GetSession(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "行动编号无效"})
		return
	}
	sess, runs, err := h.sessionService(c).GetSession(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "行动不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"session": sess, "runs": runs})
}
func (h *Handler) ListSessions(c *gin.Context) {
	list, err := h.sessionService(c).ListSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) AbortSession(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "行动编号无效"})
		return
	}
	if err := h.sessionService(c).Abort(uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "行动不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (h *Handler) RepairArmor(c *gin.Context) {
	var req struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	newMax, err := service.RepairArmorForUser(h.db, userID(c), req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		if errors.Is(err, service.ErrActiveSessionResourceLocked) || strings.Contains(err.Error(), "维修") || strings.Contains(err.Error(), "报废") {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "newMax": newMax})
}
