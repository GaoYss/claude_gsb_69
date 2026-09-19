package status

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/internal/modules/repair"
)

// Module 维修状态查询模块: 只读视图, 不拥有数据表。
type Module struct {
	service *Service
	handler *Handler
}

// New 构造维修状态查询模块, 依赖路灯 / 故障 / 维修三个模块的只读仓储。
func New(db *gorm.DB, lamps *lamp.Repository, faults *fault.Repository, repairs *repair.Repository) *Module {
	service := NewService(db, lamps, faults, repairs)
	return &Module{service: service, handler: NewHandler(service)}
}

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "维修状态查询" }

// Models 实现 module.Module 接口: 该模块为只读视图, 无需迁移模型。
func (m *Module) Models() []any { return nil }

// RegisterRoutes 实现 module.Module 接口。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/status")
	{
		group.GET("/overview", m.handler.Overview)
		group.GET("/lamps", m.handler.Lamps)
		group.GET("/track", m.handler.Track)
	}
}
