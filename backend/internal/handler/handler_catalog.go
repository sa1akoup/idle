// 游戏目录路由：提供装备、库存、携行容量和商人目录接口。
package handler

import (
	"net/http"

	"idle/internal/models"
	"idle/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListWeapons(c *gin.Context) {
	var list []models.WeaponDef
	if err := h.db.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "武器数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListAmmos(c *gin.Context) {
	var list []models.AmmoDef
	if err := h.db.Order("caliber_id asc, level asc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "弹药数据读取失败"})
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
	if err := h.db.Where("user_id = ?", userID(c)).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "护甲状态读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *Handler) ListConsumables(c *gin.Context) {
	list, err := service.ListUsableItems(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	capInfo, err := service.GetCarryCapacityForUser(h.db, userID(c))
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
	if err := h.db.Where("user_id = ?", userID(c)).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "仓库数据读取失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// ListItemInstances 返回仓库中的通用耐久物品实例。
func (h *Handler) ListItemInstances(c *gin.Context) {
	list, err := service.ListItemInstancesForUser(h.db, userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetCurrentRecovery 返回当前自动恢复计划，读取时会先结算已经过的时间。
func (h *Handler) GetCurrentRecovery(c *gin.Context) {
	recovery, err := service.GetCurrentRecoveryForUser(h.db, userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recovery)
}

func (h *Handler) GetInventoryCapacity(c *gin.Context) {
	capacity, err := service.GetStorageCapacityForUser(h.db, userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, capacity)
}

// GetLoadout 返回当前装备和失能后的自动补购预设。
func (h *Handler) GetLoadout(c *gin.Context) {
	loadout, err := service.GetPlayerLoadoutForUser(h.db, userID(c))
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
	loadout, err := service.SavePlayerLoadoutForUser(h.db, userID(c), req)
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
	if err := service.PurchaseFromMerchantForUserWithKey(h.db, userID(c), c.GetHeader("Idempotency-Key"), req.MerchantID, req.ItemID, req.Quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListMerchants 返回全部商人。
func (h *Handler) ListMerchants(c *gin.Context) {
	list, err := service.GetMerchantsForUser(h.db, userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// MerchantCatalog 返回某商人可售商品。
func (h *Handler) MerchantCatalog(c *gin.Context) {
	m, err := service.GetMerchantByIDForUser(h.db, userID(c), c.Param("id"))
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
	total, err := service.SellItemForUserWithKey(h.db, userID(c), c.GetHeader("Idempotency-Key"), req.MerchantID, req.ItemID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "total": total})
}
