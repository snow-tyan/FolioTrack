package main

import (
	"fmt"
	"foliotrack/config"
	"foliotrack/models"
	"foliotrack/routes"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// 1. Load Configurations
	config.LoadConfig()

	// 2. Connect to Database (with retry mechanism)
	var db *gorm.DB
	var err error
	maxRetries := 10

	fmt.Printf("Connecting to database: %s\n", config.AppConfig.DBDSN)

	for i := 1; i <= maxRetries; i++ {
		db, err = gorm.Open(mysql.Open(config.AppConfig.DBDSN), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err == nil {
			break
		}
		log.Printf("[%d/%d] Failed to connect to database. Retrying in 3 seconds... Error: %v", i, maxRetries, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("Fatal: Could not connect to database after %d attempts: %v", maxRetries, err)
	}

	log.Println("Database connection established successfully.")

	// 3. Initialize Global DB Instance
	models.InitDB(db)

	// 3.5 Initialize Redis Client (with retry mechanism)
	redisMaxRetries := 5
	var redisErr error
	for i := 1; i <= redisMaxRetries; i++ {
		redisErr = models.InitRedis()
		if redisErr == nil {
			break
		}
		log.Printf("[%d/%d] Failed to connect to Redis. Retrying in 2 seconds... Error: %v", i, redisMaxRetries, redisErr)
		time.Sleep(2 * time.Second)
	}
	if redisErr != nil {
		// Log warning instead of fatal, so app can fallback to DB caching if Redis is offline
		log.Printf("Warning: Could not connect to Redis: %v. Falling back to DB-only cache.", redisErr)
	}

	// 4. Run DB Migrations
	log.Println("Running database auto-migrations...")
	err = db.AutoMigrate(
		&models.User{},
		&models.Asset{},
		&models.Holding{},
		&models.Combo{},
	)
	if err != nil {
		log.Fatalf("Fatal: AutoMigrate failed: %v", err)
	}
	log.Println("Database migration completed.")

	// 5. Setup Router
	r := routes.SetupRouter()

	// 6. Start Server
	serverAddr := fmt.Sprintf(":%s", config.AppConfig.Port)
	log.Printf("FolioTrack backend server starting on %s\n", serverAddr)
	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("Fatal: Failed to start web server: %v", err)
	}
}
