package fault

import "streetlight/pkg/pagination"

// CreateRequest 故障登记请求。
type CreateRequest struct {
	LampID        uint   `json:"lamp_id" binding:"required"`
	FaultType     string `json:"fault_type" binding:"required,max=32"`
	FaultLevel    string `json:"fault_level" binding:"omitempty,oneof=low normal high urgent"`
	Source        string `json:"source" binding:"omitempty,oneof=inspection citizen monitoring other"`
	Description   string `json:"description" binding:"required,max=512"`
	Reporter      string `json:"reporter" binding:"max=64"`
	ReporterPhone string `json:"reporter_phone" binding:"max=32"`
	ReportedAt    string `json:"reported_at" binding:"omitempty,max=32"`
}

// UpdateRequest 修改故障登记信息, 仅未关闭的故障允许修改。
type UpdateRequest struct {
	FaultType     *string `json:"fault_type" binding:"omitempty,max=32"`
	FaultLevel    *string `json:"fault_level" binding:"omitempty,oneof=low normal high urgent"`
	Source        *string `json:"source" binding:"omitempty,oneof=inspection citizen monitoring other"`
	Description   *string `json:"description" binding:"omitempty,max=512"`
	Reporter      *string `json:"reporter" binding:"omitempty,max=64"`
	ReporterPhone *string `json:"reporter_phone" binding:"omitempty,max=32"`
	ReportedAt    *string `json:"reported_at" binding:"omitempty,max=32"`
}

// CloseRequest 关闭故障请求, 用于作废或确认闭环。
type CloseRequest struct {
	Remark string `json:"remark" binding:"max=255"`
}

// ListQuery 故障列表查询条件。
type ListQuery struct {
	pagination.Params
	Keyword    string `form:"keyword"` // 故障单号 / 路灯编号 / 道路 / 描述
	Status     string `form:"status"`
	FaultType  string `form:"fault_type"`
	FaultLevel string `form:"fault_level"`
	Source     string `form:"source"`
	LampID     uint   `form:"lamp_id"`
	RoadName   string `form:"road_name"`
	StartDate  string `form:"start_date"` // 上报日期起, 格式 YYYY-MM-DD
	EndDate    string `form:"end_date"`   // 上报日期止, 格式 YYYY-MM-DD
	OnlyOpen   bool   `form:"only_open"`  // 仅查询未闭环故障
}

// Meta 故障模块字典, 供前端渲染下拉框。
type Meta struct {
	Statuses   []string `json:"statuses"`
	Levels     []string `json:"levels"`
	Sources    []string `json:"sources"`
	FaultTypes []string `json:"fault_types"`
}
