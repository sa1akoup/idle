package handler

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"idle/internal/models"
	"idle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db           *gorm.DB
	secureCookie bool
}

func NewHandler(db *gorm.DB, secureCookie ...bool) *Handler {
	return &Handler{db: db, secureCookie: len(secureCookie) > 0 && secureCookie[0]}
}

func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api")
	api.POST("/auth/register", h.RegisterAuth)
	api.POST("/auth/login", h.LoginAuth)
	api.POST("/auth/logout", h.LogoutAuth)
	api.GET("/auth/me", h.MeAuth)
	protected := api.Group("")
	protected.Use(authMiddleware(h.db))
	protected.GET("/player", h.GetPlayer)
	protected.PUT("/player", h.UpdatePlayer)
	protected.GET("/weapons", h.ListWeapons)
	protected.GET("/ammos", h.ListAmmos)
	protected.GET("/armors", h.ListArmors)
	protected.GET("/armor-instances", h.ListArmorInstances)
	protected.GET("/consumables", h.ListConsumables)
	protected.GET("/loot", h.ListLoot)
	protected.GET("/chestrigs", h.ListChestRigs)
	protected.GET("/backpacks", h.ListBackpacks)
	protected.GET("/helmets", h.ListHelmets)
	protected.GET("/headsets", h.ListHeadsets)
	protected.GET("/maps", h.ListMaps)
	protected.GET("/maps/:id/graph", h.GetMapGraph)
	protected.GET("/inventory", h.ListInventory)
	protected.GET("/item-instances", h.ListItemInstances)
	protected.GET("/inventory/capacity", h.GetInventoryCapacity)
	protected.GET("/recovery/current", h.GetCurrentRecovery)
	protected.GET("/hideout", h.GetHideout)
	protected.POST("/hideout/facilities/:id/upgrade", h.StartHideoutUpgrade)
	protected.POST("/hideout/repair", h.QueueHideoutRepair)
	protected.POST("/hideout/generator/toggle", h.ToggleGenerator)
	protected.POST("/hideout/generator/fuel/load", h.LoadGeneratorFuel)
	protected.POST("/hideout/generator/fuel/unload", h.UnloadGeneratorFuel)
	protected.GET("/loadout", h.GetLoadout)
	protected.GET("/loadout/capacity", h.GetCarryCapacity)
	protected.PUT("/loadout", h.UpdateLoadout)
	protected.GET("/merchants", h.ListMerchants)
	protected.GET("/merchants/:id/catalog", h.MerchantCatalog)
	protected.POST("/merchant/purchase", h.Purchase)
	protected.POST("/merchant/sell", h.Sell)
	protected.POST("/session/start", h.StartSession)
	protected.GET("/session/:id", h.GetSession)
	protected.GET("/session/:id/events", h.ListSessionEvents)
	protected.GET("/session/:id/events/stream", h.StreamSessionEvents)
	protected.GET("/sessions", h.ListSessions)
	protected.GET("/nodes", h.ListNodes)
	protected.GET("/enemies", h.ListEnemies)
	protected.POST("/armor/repair", h.RepairArmor)
}

// GetPlayer 返回唯一的玩家角色。
func (h *Handler) GetPlayer(c *gin.Context) {
	if err := service.SettleRecoveryForUser(h.db, userID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var player models.Character
	if err := h.db.Where("user_id = ?", userID(c)).First(&player).Error; err != nil {
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
		Where("user_id = ?", userID(c)).
		Update("name", name).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "角色名称保存失败"})
		return
	}

	var player models.Character
	if err := h.db.Where("user_id = ?", userID(c)).First(&player).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "玩家角色读取失败"})
		return
	}
	c.JSON(http.StatusOK, player)
}

// Health for frontend
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "V0.2-MVP"})
}
