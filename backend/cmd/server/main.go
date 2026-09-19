package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"streetlight/internal/bootstrap"
	"streetlight/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	app, err := bootstrap.New(cfg)
	if err != nil {
		slog.Error("初始化应用失败", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := app.Close(); err != nil {
			slog.Warn("关闭数据库连接失败", "error", err)
		}
	}()

	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      app.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		slog.Info("路灯故障登记系统已启动",
			"addr", cfg.Server.Addr(),
			"env", cfg.App.Env,
			"driver", cfg.Database.Driver,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP 服务异常退出", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("收到退出信号, 开始优雅关闭")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("服务关闭超时", "error", err)
	}
	slog.Info("服务已退出")
}
