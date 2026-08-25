// 认证路由与中间件：负责登录 Cookie、Bearer Token 和当前用户注入。
package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"idle/internal/config"
	"idle/internal/models"
	"idle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const authCookieName = "idle_session"

type authRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) RegisterAuth(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}
	var user *models.User
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var err error
		user, err = service.RegisterUser(tx, req.Username, req.Password)
		if err != nil {
			return err
		}
		return config.ProvisionUser(tx, user.ID)
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("注册失败: %v", err)})
		return
	}
	user, token, err := service.LoginUser(h.db, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.setAuthCookie(c, token)
	c.JSON(http.StatusCreated, user)
}

func (h *Handler) LoginAuth(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}
	user, token, err := service.LoginUser(h.db, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	h.setAuthCookie(c, token)
	c.JSON(http.StatusOK, user)
}

func (h *Handler) LogoutAuth(c *gin.Context) {
	if err := service.LogoutToken(h.db, requestAuthToken(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "退出登录失败"})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(authCookieName, "", -1, "/", "", h.secureCookie, true)
	c.Status(http.StatusNoContent)
}

func (h *Handler) MeAuth(c *gin.Context) {
	user, err := service.AuthenticateToken(h.db, requestAuthToken(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func authMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := service.AuthenticateToken(db, requestAuthToken(c))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set("userID", user.ID)
		c.Set("user", user)
		c.Next()
	}
}

func requestAuthToken(c *gin.Context) string {
	if value := c.GetHeader("Authorization"); strings.HasPrefix(value, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
	}
	token, _ := c.Cookie(authCookieName)
	return token
}

func (h *Handler) setAuthCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(authCookieName, token, int(service.AuthSessionTTL/time.Second), "/", "", h.secureCookie, true)
}

func userID(c *gin.Context) uint {
	return c.MustGet("userID").(uint)
}

func (h *Handler) sessionService(c *gin.Context) *service.SessionService {
	return service.NewSessionService(h.db, userID(c))
}
