// 认证服务：处理账号注册、登录会话签发、当前用户读取和会话撤销。
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"idle/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const AuthSessionTTL = 30 * 24 * time.Hour

// RegisterUser 创建账号并返回用户记录。
func RegisterUser(db *gorm.DB, username, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	usernameLength := utf8.RuneCountInString(username)
	if usernameLength < 3 || usernameLength > 32 {
		return nil, fmt.Errorf("用户名长度需为 3-32 个字符")
	}
	if len(password) < 8 || len(password) > 72 {
		return nil, fmt.Errorf("密码长度需为 8-72 个字符")
	}
	var exists models.User
	if err := db.Where("username = ?", username).First(&exists).Error; err == nil {
		return nil, fmt.Errorf("用户名已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查用户名: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成密码摘要: %w", err)
	}
	// 明文密码不入库，仅保存 bcrypt 摘要，且长度上限受 bcrypt 限制
	user := &models.User{Username: username, PasswordHash: string(hash), Status: "active"}
	if err := db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("创建账号: %w", err)
	}
	return user, nil
}

// LoginUser 校验账号并签发数据库会话 token。
func LoginUser(db *gorm.DB, username, password string) (*models.User, string, error) {
	var user models.User
	if err := db.Where("username = ? AND status = ?", strings.TrimSpace(username), "active").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", fmt.Errorf("用户名或密码错误")
		}
		return nil, "", fmt.Errorf("读取账号: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", fmt.Errorf("用户名或密码错误")
	}
	token, err := issueAuthToken(db, user.ID)
	if err != nil {
		return nil, "", err
	}
	return &user, token, nil
}

// issueAuthToken 生成随机 token 并签发会话：明文仅返回客户端，库中只存其摘要供校验。
func issueAuthToken(db *gorm.DB, userID uint) (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("生成登录会话: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(buffer)
	hash := sha256.Sum256([]byte(token))
	session := models.AuthSession{
		UserID: userID, TokenHash: base64.RawURLEncoding.EncodeToString(hash[:]),
		// 会话有效期由 AuthSessionTTL 常量统一控制
		ExpiresAt: time.Now().Add(AuthSessionTTL),
	}
	if err := db.Create(&session).Error; err != nil {
		return "", fmt.Errorf("保存登录会话: %w", err)
	}
	return token, nil
}

// AuthenticateToken 根据 token 读取当前用户。
func AuthenticateToken(db *gorm.DB, token string) (*models.User, error) {
	if token == "" {
		return nil, fmt.Errorf("登录已失效")
	}
	hash := sha256.Sum256([]byte(token))
	hashText := base64.RawURLEncoding.EncodeToString(hash[:])
	var session models.AuthSession
	if err := db.Where("token_hash = ? AND expires_at > ?", hashText, time.Now()).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("登录已失效")
		}
		return nil, fmt.Errorf("读取登录会话: %w", err)
	}
	var user models.User
	if err := db.Where("id = ? AND status = ?", session.UserID, "active").First(&user).Error; err != nil {
		return nil, fmt.Errorf("读取当前用户: %w", err)
	}
	return &user, nil
}

// LogoutToken 撤销当前登录会话。
func LogoutToken(db *gorm.DB, token string) error {
	if token == "" {
		return nil
	}
	hash := sha256.Sum256([]byte(token))
	hashText := base64.RawURLEncoding.EncodeToString(hash[:])
	return db.Where("token_hash = ?", hashText).Delete(&models.AuthSession{}).Error
}
