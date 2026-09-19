package bootstrap

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"streetlight/internal/config"
	"streetlight/internal/database"
	"streetlight/internal/logging"
	"streetlight/internal/middleware"
	"streetlight/internal/module"
	"streetlight/internal/response"
)

// App 承载应用运行期依赖, 负责把配置、数据库、模块与路由装配在一起。
type App struct {
	config  *config.Config
	db      *gorm.DB
	engine  *gin.Engine
	modules []module.Module
}

// New 完成日志、数据库、数据迁移、演示数据与路由的初始化。
func New(cfg *config.Config) (*App, error) {
	logging.Setup(cfg.App.Env)
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	modules := buildModules(db)
	models := make([]any, 0)
	for _, item := range modules {
		models = append(models, item.Models()...)
	}
	if err := database.AutoMigrate(db, models); err != nil {
		return nil, err
	}

	if cfg.App.Seed {
		if err := seed(db); err != nil {
			return nil, fmt.Errorf("初始化演示数据失败: %w", err)
		}
	}

	app := &App{config: cfg, db: db, modules: modules}
	app.engine = app.buildRouter()
	return app, nil
}

// Router 返回 HTTP 处理器。
func (a *App) Router() http.Handler { return a.engine }

// DB 返回数据库连接。
func (a *App) DB() *gorm.DB { return a.db }

// Close 释放数据库连接。
func (a *App) Close() error { return database.Close(a.db) }

// buildRouter 装配全局中间件、健康检查与各模块路由。
func (a *App) buildRouter() *gin.Engine {
	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.Logger(),
		middleware.Recovery(),
		middleware.CORS(a.config.Server.CORSOrigins),
	)

	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.Envelope{
			Code:    "NOT_FOUND",
			Message: "接口不存在: " + c.Request.URL.Path,
		})
	})

	engine.GET("/healthz", func(c *gin.Context) {
		response.OK(c, gin.H{
			"status": "ok",
			"app":    a.config.App.Name,
			"env":    a.config.App.Env,
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	engine.GET("/readyz", func(c *gin.Context) {
		sqlDB, err := a.db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, response.Envelope{
				Code:    "NOT_READY",
				Message: "数据库连接不可用",
			})
			return
		}
		response.OK(c, gin.H{"status": "ready"})
	})

	api := engine.Group("/api/v1")
	names := make([]string, 0, len(a.modules))
	for _, item := range a.modules {
		item.RegisterRoutes(api)
		names = append(names, item.Name())
		slog.Info("业务模块已注册", "模块", item.Name())
	}

	engine.GET("/api/v1", func(c *gin.Context) {
		response.OK(c, gin.H{"modules": names})
	})

	return engine
}
