package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sfquery/internal/handler"
	"sfquery/internal/sfclient"
	"sfquery/static"
)

func main() {
	// ---- 命令行参数 ----
	addr := flag.String("addr", ":8080", "监听地址，例如 :8080")
	partnerID := flag.String("partner-id", "XMWL43MUNA8", "顺丰顾客编码")
	checkWord := flag.String("check-word", "PWc8G05FTdp28Oi4ZgrCGUP9SFmgKphS", "顺丰生产校验码")
	sandbox := flag.Bool("sandbox", false, "是否使用沙箱环境")
	flag.Parse()

	// 环境变量可覆盖命令行参数
	if v := os.Getenv("SF_PARTNER_ID"); v != "" {
		*partnerID = v
	}
	if v := os.Getenv("SF_CHECK_WORD"); v != "" {
		*checkWord = v
	}

	// ---- 初始化顺丰客户端 ----
	apiURL := sfclient.ProdURL
	if *sandbox {
		apiURL = sfclient.SandboxURL
		log.Println("[SF] 使用沙箱环境")
	} else {
		log.Println("[SF] 使用生产环境")
	}

	sf := sfclient.New(&sfclient.Config{
		PartnerID: *partnerID,
		CheckWord: *checkWord,
		APIUrl:    apiURL,
		Timeout:   15 * time.Second,
	})

	// ---- 注册路由 ----
	mux := http.NewServeMux()

	// API
	h := handler.New(sf)
	h.RegisterRoutes(mux)

	// 静态文件（embed 进二进制，零外部依赖）
	mux.Handle("/", http.FileServer(http.FS(static.FS)))

	// ---- 启动服务 ----
	srv := &http.Server{
		Addr:         *addr,
		Handler:      logging(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 服务已启动  http://localhost%s", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动失败: %v", err)
		}
	}()

	// ---- 优雅关闭 ----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("强制关闭: %v", err)
	}
	log.Println("已关闭")
}

// logging 简单请求日志中间件
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &logWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lw, r)
		log.Printf("%s %-30s %d  %s", r.Method, r.URL.Path, lw.status, time.Since(start))
	})
}

type logWriter struct {
	http.ResponseWriter
	status int
}

func (l *logWriter) WriteHeader(code int) {
	l.status = code
	l.ResponseWriter.WriteHeader(code)
}
