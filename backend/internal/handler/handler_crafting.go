// 制造路由：配方目录列表与制造作业入队。
package handler

import (
	"net/http"

	"idle/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListCraftingRecipes(c *gin.Context) {
	recipes, err := service.ListCraftingRecipesForUser(h.db, userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recipes)
}

func (h *Handler) StartCraft(c *gin.Context) {
	var req struct {
		RecipeID string `json:"recipeId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "配方不能为空"})
		return
	}
	if err := service.StartCraftForUser(h.db, userID(c), req.RecipeID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}