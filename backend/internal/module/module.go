package module

import "github.com/gin-gonic/gin"

// Module 是一个自包含的业务模块: 拥有自己的数据模型、数据访问、业务规则与路由。
// 新增业务时只需要实现该接口, 并在 bootstrap 中注册, 无需改动其它模块。
type Module interface {
	// Name 返回模块名称, 用于日志与启动信息。
	Name() string
	// Models 返回需要自动迁移的数据模型。
	Models() []any
	// RegisterRoutes 将模块路由挂载到指定 API 分组。
	RegisterRoutes(api *gin.RouterGroup)
}
