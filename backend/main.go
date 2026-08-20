package main

import (
	"log"
	"os"

	"idle/internal/config"
	"idle/internal/handler"
	"idle/internal/models"
	"idle/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "idle.db"
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	// migrate
	if err := db.AutoMigrate(
		&models.Character{},
		&models.WeaponDef{},
		&models.ArmorDef{},
		&models.ConsumableDef{},
		&models.ChestRigDef{},
		&models.BackpackDef{},
		&models.HelmetDef{},
		&models.HeadsetDef{},
		&models.LootItemDef{},
		&models.LootContainerDef{},
		&models.LootContainerRule{},
		&models.NodeContainerDef{},
		&models.MapDef{},
		&models.NodeDef{},
		&models.EnemyDef{},
		&models.EventDef{},
		&models.EventBinding{},
		&models.EncounterPoolEntry{},
		&models.MerchantDef{},
		&models.Session{},
		&models.SessionRun{},
		&models.PlayerLoadout{},
		&models.Inventory{},
		&models.ArmorInstance{},
	); err != nil {
		log.Fatal(err)
	}
	if err := config.Seed(db); err != nil {
		log.Fatal(err)
	}
	if err := service.ValidateEventConfig(db); err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))
	r.GET("/api/health", handler.Health)
	h := handler.NewHandler(db)
	h.Register(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Server running at http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
