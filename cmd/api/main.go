package main

import (
	"log"

	"cermin-backend/internal/config"
	"cermin-backend/internal/database"
	"cermin-backend/internal/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	if err := database.AutoMigrate(db); err != nil {
		log.Fatal(err)
	}

	r := router.Setup(db, cfg)

	log.Printf("server running on port %s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
