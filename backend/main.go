package main

import (
	"fmt"
	"log"
	"os"

	"idle/internal/config"
	"idle/internal/handler"
	database "idle/internal/repository/database"
	"idle/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	settings, err := config.LoadSettings()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Open(settings.DatabaseDriver, settings.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 {
		if err := runCommand(os.Args[1], db, settings); err != nil {
			log.Fatal(err)
		}
		return
	}
	if settings.MigrateOnStart {
		if err := database.Migrate(db, settings.DatabaseDriver); err != nil {
			log.Fatal(err)
		}
	} else if err := database.EnsureMigrated(db, settings.DatabaseDriver); err != nil {
		log.Fatal(err)
	}
	if settings.SeedOnStart {
		if err := seedDatabase(db); err != nil {
			log.Fatal(err)
		}
	}

	if err := startServer(db, settings); err != nil {
		log.Fatal(err)
	}
}

func runCommand(command string, db *gorm.DB, settings config.Settings) error {
	switch command {
	case "migrate":
		return database.Migrate(db, settings.DatabaseDriver)
	case "seed":
		if err := database.Migrate(db, settings.DatabaseDriver); err != nil {
			return err
		}
		return seedDatabase(db)
	default:
		return fmt.Errorf("未知命令 %q，可用命令：migrate、seed", command)
	}
}

func seedDatabase(db *gorm.DB) error {
	if err := config.Seed(db); err != nil {
		return fmt.Errorf("初始化基础数据失败: %w", err)
	}
	if err := service.ValidateEventConfig(db); err != nil {
		return fmt.Errorf("校验事件配置失败: %w", err)
	}
	return nil
}

func startServer(db *gorm.DB, settings config.Settings) error {

	r := gin.Default()
	if err := r.SetTrustedProxies(settings.TrustedProxies); err != nil {
		return fmt.Errorf("设置可信代理失败: %w", err)
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     settings.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "Idempotency-Key"},
		AllowCredentials: true,
	}))
	r.GET("/api/health", handler.Health)
	h := handler.NewHandler(db, settings.Environment == "production")
	h.Register(r)
	service.StartSessionScheduler(db)

	log.Printf("Server running at http://localhost%s", settings.HTTPAddr)
	return r.Run(settings.HTTPAddr)
}
