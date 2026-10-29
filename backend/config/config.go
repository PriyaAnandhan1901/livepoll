package config

import (
	"log"
	"os"
)

type Config struct {
	Port       string
	MongoURI   string
	MongoDB    string
	RedisURL   string
	JWTSecret  string
	CORSOrigin string
}

func Load() Config {
	c := Config{
		Port:       get("PORT", "8080"),
		MongoURI:   must("MONGO_URI"),
		MongoDB:    get("MONGO_DB", "livepoll"),
		RedisURL:   must("REDIS_URL"),
		JWTSecret:  must("JWT_SECRET"),
		CORSOrigin: get("CORS_ORIGIN", "http://localhost:5173"),
	}
	if len(c.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters")
	}
	return c
}

func get(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func must(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing required env var %s", k)
	}
	return v
}
