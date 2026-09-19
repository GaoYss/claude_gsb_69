package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Setup 初始化全局 slog: 生产环境输出 JSON, 开发环境输出可读文本。
func Setup(env string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}

	var handler slog.Handler
	if strings.EqualFold(env, "production") {
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
