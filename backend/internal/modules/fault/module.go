package fault

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 故障登记模块, 负责故障上报受理与状态流转。
type Module struct {
	repository *Repository
	service    *Service
	handler    *Handler
}

// New 构造故障登记模块, lamps 为路灯台账模块提供的端口实现。
func New(db *gorm.DB, lamps LampPort) *Module {
	repository := NewRepository(db)
	service := NewService(repository, lamps)
	return &Module{
		repository: repository,
		service:    service,
		handler:    NewHandler(service),
	}
}

// Service 暴露业务服务, 供维修模块装配端口使用。
func (m *Module) Service() *Service { return m.service }

// Repository 暴露仓储, 供路灯模块装配未闭环故障计数器。
func (m *Module) Repository() *Repository { return m.repository }

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "故障登记" }

// Models 实现 module.Module 接口。
func (m *Module) Models() []any { return []any{&Fault{}} }

// RegisterRoutes 实现 module.Module 接口。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/faults")
	{
		group.GET("", m.handler.List)
		group.POST("", m.handler.Create)
		group.GET("/meta", m.handler.Metadata)
		group.GET("/:id", m.handler.Get)
		group.PUT("/:id", m.handler.Update)
		group.DELETE("/:id", m.handler.Delete)
		group.POST("/:id/close", m.handler.Close)
	}
}
