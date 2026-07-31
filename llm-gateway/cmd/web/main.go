package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"agent-in-action/llm-gateway/internal/appconfig"
	"agent-in-action/llm-gateway/internal/factory"
	"agent-in-action/llm-gateway/internal/web"

	"github.com/joho/godotenv"
)

const (
	defaultPort            = "8080"
	defaultRequestTimeout  = 60 * time.Second
	defaultShutdownTimeout = 30 * time.Second
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("未加载 .env 文件或文件不存在")
	}

	if err := run(); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func run() error {
	config, err := appconfig.Load()
	if err != nil {
		return fmt.Errorf("读取配置: %w", err)
	}

	router, err := factory.BuildRouter(config)
	if err != nil {
		return fmt.Errorf("构建路由: %w", err)
	}

	port := getEnv("PORT", defaultPort)
	timeout := parseDurationEnv("WEB_REQUEST_TIMEOUT", defaultRequestTimeout)
	corsOrigins := parseCORSOrigins(os.Getenv("WEB_CORS_ORIGINS"))

	server := web.New(
		router,
		web.WithTimeout(timeout),
		web.WithCORS(corsOrigins),
	)

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on :%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭 HTTP server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("关闭 server: %w", err)
	}
	log.Println("HTTP server 已关闭")
	return nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("%s 格式无效，使用默认值 %s: %v", key, fallback, err)
		return fallback
	}
	if duration <= 0 {
		log.Printf("%s 必须大于 0，使用默认值 %s", key, fallback)
		return fallback
	}
	return duration
}

func parseCORSOrigins(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		origins = append(origins, origin)
	}
	return origins
}
