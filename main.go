package main

import (
	"benz-sniper/api"
	"benz-sniper/config"
	"benz-sniper/database"
	"benz-sniper/engine"
	"context"
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed index.html
var indexHTML embed.FS

//go:embed assets
var assetsFS embed.FS

func main() {
	// 加载配置
	cfg := config.Load()
	
	// 初始化数据库
	if err := database.Init(cfg); err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}
	defer database.Close()
	
	// 创建策略管理器（虚实盘系统，使用默认配置）
	manager := engine.NewStrategyManager(database.GetDB())
	
	// 创建并启动分析引擎（后台单goroutine）
	eng := engine.New(database.GetDB(), manager)
	go eng.Run()
	
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)
	
	// 创建路由（自定义日志，只记录错误和慢请求）
	router := gin.New()
	router.Use(customLogger())
	router.Use(gin.Recovery())
	
	// 启用 CORS
	router.Use(corsMiddleware())
	
	// 设置 API 路由（读写锁保护）
	apiHandler := api.New(manager)
	apiHandler.SetupRoutes(router)
	
	// 使用嵌入的静态文件（支持 CI/CD 部署）
	// 首页
	router.GET("/", func(c *gin.Context) {
		data, err := indexHTML.ReadFile("index.html")
		if err != nil {
			log.Printf("❌ 读取 index.html 失败: %v", err)
			c.String(http.StatusInternalServerError, "页面加载失败")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
	
	// 静态资源目录
	assetsSubFS, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		log.Fatalf("❌ 加载静态资源失败: %v", err)
	}
	router.StaticFS("/assets", http.FS(assetsSubFS))
	
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
		log.Printf("🚀 服务器启动在端口: %s (虚实盘模式)", port)
		
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

// customLogger 自定义日志中间件（只记录慢请求和错误）
func customLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		
		c.Next()
		
		// 跳过静态资源
		if path == "/" || 
		   c.Request.URL.Path == "/assets/css/style.css" ||
		   c.Request.URL.Path == "/assets/js/app.js" {
			return
		}
		
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		
		// 只记录错误请求或慢请求（>500ms）
		if statusCode >= 400 || latency > 500*time.Millisecond {
			log.Printf("[GIN] %d | %13v | %s | %s",
				statusCode,
				latency,
				c.Request.Method,
				path,
			)
		}
	}
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
