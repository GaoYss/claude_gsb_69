package repair

import "time"

// 维修记录状态。
const (
	StatusOngoing  = "ongoing"  // 维修中
	StatusFinished = "finished" // 已完成
)

// 维修结果。
const (
	ResultFixed        = "fixed"         // 已修复
	ResultPendingParts = "pending_parts" // 待配件
	ResultObserving    = "observing"     // 观察中
	ResultUnfixable    = "unfixable"     // 无法修复
)

// Statuses 返回全部维修记录状态。
func Statuses() []string {
	return []string{StatusOngoing, StatusFinished}
}

// Results 返回全部维修结果取值。
func Results() []string {
	return []string{ResultFixed, ResultPendingParts, ResultObserving, ResultUnfixable}
}

// IsValidResult 校验维修结果取值。
func IsValidResult(result string) bool {
	for _, item := range Results() {
		if item == result {
			return true
		}
	}
	return false
}

// Repair 维修记录, 一条记录对应故障的一次维修过程。
type Repair struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	RepairNo     string     `gorm:"size:64;uniqueIndex;not null" json:"repair_no"`
	FaultID      uint       `gorm:"index;not null" json:"fault_id"`
	FaultNo      string     `gorm:"size:64;index" json:"fault_no"`
	LampID       uint       `gorm:"index" json:"lamp_id"`
	LampCode     string     `gorm:"size:64;index" json:"lamp_code"`
	Repairman    string     `gorm:"size:64;index;not null" json:"repairman"`
	RepairTeam   string     `gorm:"size:64;index" json:"repair_team"`
	ContactPhone string     `gorm:"size:32" json:"contact_phone"`
	StartedAt    time.Time  `gorm:"index;not null" json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Status       string     `gorm:"size:32;index;not null;default:ongoing" json:"status"`
	Result       string     `gorm:"size:32;index" json:"result"`
	Content      string     `gorm:"size:512" json:"content"`
	Materials    string     `gorm:"size:255" json:"materials"`
	Cost         float64    `json:"cost"`
	Remark       string     `gorm:"size:255" json:"remark"`

	// DurationMinutes 仅用于响应展示的维修耗时(分钟), 不落库。
	DurationMinutes *int64 `gorm:"-" json:"duration_minutes,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (Repair) TableName() string { return "repair" }

// FillDuration 依据开工与完工时间计算维修耗时。
func (r *Repair) FillDuration() {
	if r.FinishedAt == nil {
		r.DurationMinutes = nil
		return
	}
	minutes := int64(r.FinishedAt.Sub(r.StartedAt).Minutes())
	if minutes < 0 {
		minutes = 0
	}
	r.DurationMinutes = &minutes
}
