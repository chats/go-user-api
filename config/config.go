package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	AppEnv           string
	ServerPort       string
	GrpcPort         string
	CorsAllowOrigins string
	LogLevel         string

	// Database type (postgres or mongodb)
	DBType string

	// PostgreSQL
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	// MongoDB
	MongoDBHost     string
	MongoDBPort     string
	MongoDBName     string
	MongoDBUser     string
	MongoDBPassword string
	MongoDBAuthDB   string

	// JWT
	JWTSecret       string
	JWTExpireMinute int

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	RedisCacheTTL int

	// Tracing
	JaegerEndpoint string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Warn().Msg("Warning: .env file not found")
	}

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	redisCacheTTL, _ := strconv.Atoi(getEnv("REDIS_CACHE_TTL", "3600"))
	jwtExpireMinute, _ := strconv.Atoi(getEnv("JWT_EXPIRE_MINUTES", "60"))

	return &Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		GrpcPort:         getEnv("GRPC_PORT", "50051"),
		CorsAllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "http://localhost:3000,http://localhost:8080"),
		LogLevel:         getEnv("LOG_LEVEL", "debug"),

		// Database type
		DBType: getEnv("DB_TYPE", "postgres"),

		// PostgreSQL
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "user-api"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// MongoDB
		MongoDBHost:     getEnv("MONGODB_HOST", "localhost"),
		MongoDBPort:     getEnv("MONGODB_PORT", "27017"),
		MongoDBName:     getEnv("MONGODB_NAME", "user-api"),
		MongoDBUser:     getEnv("MONGODB_USER", ""),
		MongoDBPassword: getEnv("MONGODB_PASSWORD", ""),
		MongoDBAuthDB:   getEnv("MONGODB_AUTH_DB", "admin"),

		// JWT
		JWTSecret:       getEnv("JWT_SECRET", "your-super-secret-key-here"),
		JWTExpireMinute: jwtExpireMinute,

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,
		RedisCacheTTL: redisCacheTTL,

		// Tracing
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (c *Config) GetDBConnString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func (c *Config) GetMongoDBConnString() string {
	// If username and password are set, include them in the connection string
	if c.MongoDBUser != "" && c.MongoDBPassword != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s",
			c.MongoDBUser, c.MongoDBPassword, c.MongoDBHost, c.MongoDBPort, c.MongoDBName, c.MongoDBAuthDB)
	}
	// Otherwise, connect without authentication
	return fmt.Sprintf("mongodb://%s:%s/%s",
		c.MongoDBHost, c.MongoDBPort, c.MongoDBName)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func (c *Config) GetJWTExpiration() time.Duration {
	return time.Duration(c.JWTExpireMinute) * time.Minute
}
