package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port    string
	Env     string

	// MongoDB
	MongoURI string
	DBName   string

	// Redis
	RedisURL string

	// JWT
	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessExpiry  int // minutes
	JWTRefreshExpiry int // days

	// Paystack
	PaystackSecretKey  string
	PaystackPublicKey  string
	PaystackCurrency   string // e.g. "NGN", "USD", "GHS"

	// Twitter
	TwitterBearerToken string

	// Google OAuth
	GoogleClientID string

	// Email (SMTP)
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// Email (Resend HTTP API — preferred over SMTP on Railway)
	ResendAPIKey string

	// Email (Brevo — fallback if ResendAPIKey not set)
	BrevoAPIKey string

	// App
	ClientURL          string
	UploadDir          string
	PlatformCommission float64 // e.g. 0.20 = 20%
}

var App *Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	App = &Config{
		Port:    getEnv("PORT", "5000"),
		Env:     getEnv("NODE_ENV", "development"),

		MongoURI: getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:   getEnv("DB_NAME", "pulse"),

		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		JWTAccessSecret:  getEnv("JWT_ACCESS_SECRET", "access-secret-change-me"),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", "refresh-secret-change-me"),
		JWTAccessExpiry:  getEnvInt("JWT_ACCESS_EXPIRY_MINUTES", 15),
		JWTRefreshExpiry: getEnvInt("JWT_REFRESH_EXPIRY_DAYS", 7),

		PaystackSecretKey: getEnv("PAYSTACK_SECRET_KEY", ""),
		PaystackPublicKey: getEnv("PAYSTACK_PUBLIC_KEY", ""),
		PaystackCurrency:  getEnv("PAYSTACK_CURRENCY", "NGN"),

		TwitterBearerToken: getEnv("TWITTER_BEARER_TOKEN", ""),

		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),

		SMTPHost: getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort: getEnvInt("SMTP_PORT", 587),
		SMTPUser: getEnv("SMTP_USER", ""),
		SMTPPass: getEnv("SMTP_PASS", ""),
		SMTPFrom: getEnv("SMTP_FROM", "noreply@pulse.app"),

		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		BrevoAPIKey:  getEnv("BREVO_API_KEY", ""),

		ClientURL:          getEnv("CLIENT_URL", "http://localhost:3000"),
		UploadDir:          getEnv("UPLOAD_DIR", "./uploads"),
		PlatformCommission: getEnvFloat("PLATFORM_COMMISSION_RATE", 0.20),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
