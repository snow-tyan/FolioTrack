package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          string
	DBDSN         string
	JWTSecret     []byte
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

var AppConfig *Config

func LoadConfig() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		// Default development fallback
		dbDSN = "root:rootpass@tcp(127.0.0.1:3306)/foliotrack?charset=utf8mb4&parseTime=True&loc=Local"
	}

	jwtSecretStr := os.Getenv("JWT_SECRET")
	if jwtSecretStr == "" {
		jwtSecretStr = "foliotrack-super-secret-key-change-in-prod"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if val, err := strconv.Atoi(dbStr); err == nil {
			redisDB = val
		}
	}

	AppConfig = &Config{
		Port:          port,
		DBDSN:         dbDSN,
		JWTSecret:     []byte(jwtSecretStr),
		RedisAddr:     redisAddr,
		RedisPassword: redisPassword,
		RedisDB:       redisDB,
	}
}
