package lamp

import "streetlight/pkg/pagination"

// CreateRequest 新增路灯台账请求。
type CreateRequest struct {
	Code        string   `json:"code" binding:"required,max=64"`
	Name        string   `json:"name" binding:"max=128"`
	RoadName    string   `json:"road_name" binding:"required,max=128"`
	District    string   `json:"district" binding:"max=64"`
	Address     string   `json:"address" binding:"max=255"`
	Longitude   *float64 `json:"longitude" binding:"omitempty,min=-180,max=180"`
	Latitude    *float64 `json:"latitude" binding:"omitempty,min=-90,max=90"`
	LampType    string   `json:"lamp_type" binding:"required,max=32"`
	Power       *int     `json:"power" binding:"omitempty,min=0,max=3000"`
	PoleHeight  *float64 `json:"pole_height" binding:"omitempty,min=0,max=100"`
	RunStatus   string   `json:"run_status" binding:"omitempty,oneof=normal fault maintenance offline"`
	InstallDate string   `json:"install_date" binding:"omitempty,datetime=2006-01-02"`
	Remark      string   `json:"remark" binding:"max=255"`
}

// UpdateRequest 更新路灯台账请求, 指针字段用于区分"未提交"与"置空"。
type UpdateRequest struct {
	Code        *string  `json:"code" binding:"omitempty,max=64"`
	Name        *string  `json:"name" binding:"omitempty,max=128"`
	RoadName    *string  `json:"road_name" binding:"omitempty,max=128"`
	District    *string  `json:"district" binding:"omitempty,max=64"`
	Address     *string  `json:"address" binding:"omitempty,max=255"`
	Longitude   *float64 `json:"longitude" binding:"omitempty,min=-180,max=180"`
	Latitude    *float64 `json:"latitude" binding:"omitempty,min=-90,max=90"`
	LampType    *string  `json:"lamp_type" binding:"omitempty,max=32"`
	Power       *int     `json:"power" binding:"omitempty,min=0,max=3000"`
	PoleHeight  *float64 `json:"pole_height" binding:"omitempty,min=0,max=100"`
	RunStatus   *string  `json:"run_status" binding:"omitempty,oneof=normal fault maintenance offline"`
	InstallDate *string  `json:"install_date" binding:"omitempty,datetime=2006-01-02"`
	Remark      *string  `json:"remark" binding:"omitempty,max=255"`
}

// ListQuery 路灯台账列表查询条件。
type ListQuery struct {
	pagination.Params
	Keyword   string `form:"keyword"`   // 编号 / 名称 / 道路 / 地址 模糊匹配
	RoadName  string `form:"road_name"` // 精确匹配
	District  string `form:"district"`
	LampType  string `form:"lamp_type"`
	RunStatus string `form:"run_status"`
}

// Statistics 路灯台账统计结果。
type Statistics struct {
	Total       int64            `json:"total"`
	RoadCount   int64            `json:"road_count"`
	ByRunStatus map[string]int64 `json:"by_run_status"`
	ByLampType  map[string]int64 `json:"by_lamp_type"`
}

// Options 前端下拉选项与建议编号。
type Options struct {
	Roads       []string `json:"roads"`
	Districts   []string `json:"districts"`
	LampTypes   []string `json:"lamp_types"`
	RunStatuses []string `json:"run_statuses"`
	NextCode    string   `json:"next_code"`
}
