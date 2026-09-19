package repair

import "streetlight/pkg/pagination"

// CreateRequest 维修记录录入请求。
type CreateRequest struct {
	FaultID      uint     `json:"fault_id" binding:"required"`
	Repairman    string   `json:"repairman" binding:"required,max=64"`
	RepairTeam   string   `json:"repair_team" binding:"max=64"`
	ContactPhone string   `json:"contact_phone" binding:"max=32"`
	StartedAt    string   `json:"started_at" binding:"omitempty,max=32"`
	Content      string   `json:"content" binding:"max=512"`
	Materials    string   `json:"materials" binding:"max=255"`
	Cost         *float64 `json:"cost" binding:"omitempty,min=0"`
	Remark       string   `json:"remark" binding:"max=255"`
}

// UpdateRequest 修改维修记录, 仅未完成的记录允许修改。
type UpdateRequest struct {
	Repairman    *string  `json:"repairman" binding:"omitempty,max=64"`
	RepairTeam   *string  `json:"repair_team" binding:"omitempty,max=64"`
	ContactPhone *string  `json:"contact_phone" binding:"omitempty,max=32"`
	StartedAt    *string  `json:"started_at" binding:"omitempty,max=32"`
	Content      *string  `json:"content" binding:"omitempty,max=512"`
	Materials    *string  `json:"materials" binding:"omitempty,max=255"`
	Cost         *float64 `json:"cost" binding:"omitempty,min=0"`
	Remark       *string  `json:"remark" binding:"omitempty,max=255"`
}

// FinishRequest 完成维修请求, 提交后维修记录闭环并联动故障状态。
type FinishRequest struct {
	FinishedAt string   `json:"finished_at" binding:"omitempty,max=32"`
	Result     string   `json:"result" binding:"required,oneof=fixed pending_parts observing unfixable"`
	Content    string   `json:"content" binding:"omitempty,max=512"`
	Materials  string   `json:"materials" binding:"omitempty,max=255"`
	Cost       *float64 `json:"cost" binding:"omitempty,min=0"`
	Remark     string   `json:"remark" binding:"omitempty,max=255"`
}

// ListQuery 维修记录查询条件。
type ListQuery struct {
	pagination.Params
	Keyword    string `form:"keyword"` // 维修单号 / 故障单号 / 路灯编号 / 维修人员
	FaultID    uint   `form:"fault_id"`
	LampID     uint   `form:"lamp_id"`
	Repairman  string `form:"repairman"`
	RepairTeam string `form:"repair_team"`
	Status     string `form:"status"`
	Result     string `form:"result"`
	StartDate  string `form:"start_date"`
	EndDate    string `form:"end_date"`
}

// Meta 维修模块字典, 含历史维修人员与班组。
type Meta struct {
	Statuses  []string `json:"statuses"`
	Results   []string `json:"results"`
	Repairmen []string `json:"repairmen"`
	Teams     []string `json:"teams"`
}

// Statistics 维修统计结果。
type Statistics struct {
	Total             int64   `json:"total"`
	OngoingTotal      int64   `json:"ongoing_total"`
	FinishedTotal     int64   `json:"finished_total"`
	TotalCost         float64 `json:"total_cost"`
	AverageCost       float64 `json:"average_cost"`
	AverageDurationHr float64 `json:"average_duration_hours"`
}
