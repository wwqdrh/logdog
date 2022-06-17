package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wwqdrh/logdog/api"

	"github.com/wwqdrh/logger"
)

var (
	addr    = flag.String("addr", ":8080", "服务端口")
	logPath = flag.String("logpath", "./logs", "监听的日志文件夹路径")
)

func main() {
	engine, err := api.NewEngine(*logPath)
	if err != nil {
		logger.DefaultLogger.Error(err.Error())
		return
	}

	srv := http.Server{
		Addr:    *addr,
		Handler: engine,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.DefaultLogger.Warn(err.Error())
		} else {
			logger.DefaultLogger.Info("exit down")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	// 退出服务
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.DefaultLogger.Warn(err.Error())
	}
}
