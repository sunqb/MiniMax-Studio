package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sunqb/minimax-studio/internal/config"
	"github.com/sunqb/minimax-studio/internal/handler"
	"github.com/sunqb/minimax-studio/internal/history"
	"github.com/sunqb/minimax-studio/internal/minimax"
	"github.com/sunqb/minimax-studio/internal/storage"
	"github.com/sunqb/minimax-studio/internal/webfs"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env not loaded: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	mmClient := minimax.NewClient(cfg.MinimaxAPIKey, cfg.MinimaxBaseURL)
	cloneMMClient := mmClient
	if cfg.MinimaxPaygAPIKey != "" {
		cloneMMClient = minimax.NewClient(cfg.MinimaxPaygAPIKey, cfg.MinimaxBaseURL)
		log.Println("MiniMax PAYG client initialized for voice clone")
	}

	r2Client, err := storage.NewR2Client(
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2SecretAccessKey,
		cfg.R2BucketName,
		cfg.R2PublicURL,
	)
	if err != nil {
		log.Fatalf("R2 init error: %v", err)
	}
	log.Println("R2 storage initialized")

	histStore, err := history.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("history store error: %v", err)
	}
	log.Printf("history store initialized at %s", cfg.DBPath)

	gin.SetMode(cfg.GinMode)
	r := gin.Default()

	// 请求体最大 10MB，防止超大请求
	r.MaxMultipartMemory = 10 << 20

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, X-Site-Password, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	h := handler.New(cfg, mmClient, cloneMMClient, r2Client, histStore)
	h.Register(r)

	// 静态文件：HTML 禁止缓存，保证重新部署后立即生效
	staticFS := http.FS(webfs.FS())
	r.GET("/app/*filepath", func(c *gin.Context) {
		fp := c.Param("filepath")
		if strings.HasSuffix(fp, ".html") || fp == "/" || fp == "" {
			c.Header("Cache-Control", "no-store")
		}
		c.FileFromFS(fp, staticFS)
	})
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app/")
	})

	addr := ":" + cfg.Port
	log.Printf("MiniMax Studio starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
