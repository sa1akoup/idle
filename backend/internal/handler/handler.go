package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"idle/internal/models"
	"idle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db      *gorm.DB
	session *service.SessionService
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db, session: service.NewSessionService(db)}
}

func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api")
	api.GET("/player", h.GetPlayer)
	api.PUT("/player", h.UpdatePlayer)
	api.GET("/weapons", h.ListWeapons)
	api.GET("/armors", h.ListArmors)
	api.GET("/armor-instances", h.ListArmorInstances)
	api.GET("/consumables", h.ListConsumables)
	api.GET("/loot", h.ListLoot)
	api.GET("/chestrigs", h.ListChestRigs)
	api.GET("/backpacks", h.ListBackpacks)
	api.GET("/helmets", h.ListHelmets)
	api.GET("/headsets", h.ListHeadsets)
	api.GET("/maps", h.ListMaps)
	api.GET("/inventory", h.ListInventory)
	api.GET("/inventory/capacity", h.GetInventoryCapacity)
	api.GET("/loadout", h.GetLoadout)
	api.GET("/loadout/capacity", h.GetCarryCapacity)
	api.PUT("/loadout", h.UpdateLoadout)
	api.GET("/merchants", h.ListMerchants)
	api.GET("/merchants/:id/catalog", h.MerchantCatalog)
	api.POST("/merchant/purchase", h.Purchase)
	api.POST("/merchant/sell", h.Sell)
	api.POST("/session/start", h.StartSession)
	api.GET("/session/:id", h.GetSession)
	api.GET("/sessions", h.ListSessions)
	api.POST("/session/:id/abort", h.AbortSession)
	api.POST("/session/preview", h.Preview)
	api.GET("/nodes", h.ListNodes)
	api.GET("/enemies", h.ListEnemies)
	api.POST("/armor/repair", h.RepairArmor)
}

// GetPlayer 返回唯一的玩家角色。
func (h *Handler) GetPlayer(c *gin.Context) {
	var player models.Character
	if err := h.db.First(&player, models.PlayerCharacterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "玩家角色不存在"})
		return
	}
	c.JSON(http.StatusOK, player)
}

// UpdatePlayer 更新玩家可自定义的基础资料。
func (h *Handler) UpdatePlayer(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入角色名称"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 16 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色名称长度需为 1-16 个字符"})
		return
	}
	if err := h.db.Model(&models.Character{}).
		Where("id = ?", models.PlayerCharacterID).
		Update("name", name).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "角色名称保存失败"})
		return
	}

	var player models.Character
	if err := h.db.First(&player, models.PlayerCharacterID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "玩家角色读取失败"})
		return
	}
	c.JSON(http.StatusOK, player)
}
func (h *Handler) ListWeapons(c *gin.Context) {
	var list []models.WeaponDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "武器数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListArmors(c *gin.Context) {
	var list []models.ArmorDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "护甲数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListArmorInstances(c *gin.Context) {
	var list []models.ArmorInstance
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "护甲状态读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListConsumables(c *gin.Context) {
	var list []models.ConsumableDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "补给数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListLoot(c *gin.Context) {
	var list []models.LootItemDef
	if err := h.db.Order("category asc, id asc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "loot 数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListChestRigs(c *gin.Context) {
	var list []models.ChestRigDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "胸挂数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListBackpacks(c *gin.Context) {
	var list []models.BackpackDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "背包数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListHelmets(c *gin.Context) {
	var list []models.HelmetDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "头盔数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListHeadsets(c *gin.Context) {
	var list []models.HeadsetDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "耳机数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetCarryCapacity 返回当前携行容量与占用。
func (h *Handler) GetCarryCapacity(c *gin.Context) {
	capInfo, err := service.GetCarryCapacity(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, capInfo)
}
func (h *Handler) ListMaps(c *gin.Context) {
	var list []models.MapDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "地图数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListInventory(c *gin.Context) {
	var list []models.Inventory
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "仓库数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetInventoryCapacity(c *gin.Context) {
	capacity, err := service.GetStorageCapacity(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, capacity)
}

// GetLoadout 返回当前装备和失能后的自动补购预设。
func (h *Handler) GetLoadout(c *gin.Context) {
	loadout, err := service.GetPlayerLoadout(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loadout)
}

// UpdateLoadout 保存角色装备配置。
func (h *Handler) UpdateLoadout(c *gin.Context) {
	var req service.SaveLoadoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请完整选择武器与护甲"})
		return
	}
	loadout, err := service.SavePlayerLoadout(h.db, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loadout)
}

// Purchase 从指定商人处购买一项商品并存入仓库。
func (h *Handler) Purchase(c *gin.Context) {
	var req struct {
		MerchantID string `json:"merchantId" binding:"required"`
		ItemID     string `json:"itemId" binding:"required"`
		Quantity   int    `json:"quantity" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商人、商品和购买数量不能为空"})
		return
	}
	if err := service.PurchaseFromMerchant(h.db, req.MerchantID, req.ItemID, req.Quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListMerchants 返回全部商人。
func (h *Handler) ListMerchants(c *gin.Context) {
	list, err := service.GetMerchants(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// MerchantCatalog 返回某商人可售商品。
func (h *Handler) MerchantCatalog(c *gin.Context) {
	m, err := service.GetMerchantByID(h.db, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items, err := service.MerchantCatalog(h.db, m)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// Sell 将局内带出物品出售给指定商人。
func (h *Handler) Sell(c *gin.Context) {
	var req struct {
		MerchantID string `json:"merchantId" binding:"required"`
		ItemID     string `json:"itemId" binding:"required"`
		Quantity   int    `json:"quantity" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商人、商品和出售数量不能为空"})
		return
	}
	total, err := service.SellItem(h.db, req.MerchantID, req.ItemID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "total": total})
}
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
	sess, err := h.session.Start(req)
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
	sess, runs, err := h.session.GetSession(uint(id))
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
	list, err := h.session.ListSessions()
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
	if err := h.session.Abort(uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "行动不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (h *Handler) Preview(c *gin.Context) {
	var req service.StartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	res, err := h.session.SimulatePreview(req)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, res)
}
func (h *Handler) RepairArmor(c *gin.Context) {
	var req struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	var inst models.ArmorInstance
	if err := h.db.First(&inst, req.ID).Error; err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	if inst.RepairCount >= 1 {
		c.JSON(400, gin.H{"error": "已达维修上限，报废"})
		return
	}
	if inst.CurDurability > 0 {
		c.JSON(400, gin.H{"error": "仅归零护甲可维修"})
		return
	}
	newMax := inst.MaxDurability / 2
	if err := h.db.Model(&inst).Updates(map[string]interface{}{
		"max_durability": newMax, "cur_durability": newMax, "repair_count": inst.RepairCount + 1, "status": "normal",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "护甲维修失败"})
		return
	}
	c.JSON(200, gin.H{"ok": true, "newMax": newMax})
}

// Health for frontend
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "V0.2-MVP"})
}
