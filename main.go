package main

import (
	"benz-sniper/api"
	"benz-sniper/config"
	"benz-sniper/database"
	"benz-sniper/engine"
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.Load()
	
	// 初始化数据库
	if err := database.Init(cfg); err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}
	defer database.Close()
	
	// 创建原子状态容器
	state := &engine.AtomicState{}
	
	// 创建并启动分析引擎（后台单goroutine）
	eng := engine.New(database.GetDB(), state)
	go eng.Run()
	
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)
	
	// 创建路由
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// 启用 CORS
	router.Use(corsMiddleware())
	
	// 设置 API 路由（无锁读取）
	apiHandler := api.New(state)
	apiHandler.SetupRoutes(router)
	
	// 静态文件服务
	router.StaticFile("/", "./index.html")
	router.Static("/assets", "./assets")
	
	// 获取本机IP地址
	ip := getLocalIP()
	port := cfg.ServerPort
	
	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:           "0.0.0.0:" + port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	
	// 启动服务器
	go func() {
		log.Printf("📱 狙击手地址: http://%s:%s", ip, port)
		log.Printf("🚀 服务器启动在端口: %s (无锁模式)", port)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ 服务器启动失败: %v", err)
		}
	}()
	
	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("🛑 正在关闭服务器...")
	
	// 关闭 HTTP 服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("❌ 服务器强制关闭: %v", err)
	}
	
	log.Println("✅ 服务器已关闭")
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

// getLocalIP 获取本机IP地址
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
