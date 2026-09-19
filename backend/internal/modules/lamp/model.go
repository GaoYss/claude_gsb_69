package lamp

import "time"

// 路灯运行状态。
const (
	RunStatusNormal      = "normal"      // 正常
	RunStatusFault       = "fault"       // 故障
	RunStatusMaintenance = "maintenance" // 维修中
	RunStatusOffline     = "offline"     // 停用
)

// 灯具类型。
const (
	LampTypeLED    = "LED"
	LampTypeSodium = "高压钠灯"
	LampTypeMetal  = "金卤灯"
	LampTypeSolar  = "太阳能"
	LampTypeOther  = "其他"
)

// RunStatuses 返回全部运行状态取值。
func RunStatuses() []string {
	return []string{RunStatusNormal, RunStatusFault, RunStatusMaintenance, RunStatusOffline}
}

// LampTypes 返回全部灯具类型取值。
func LampTypes() []string {
	return []string{LampTypeLED, LampTypeSodium, LampTypeMetal, LampTypeSolar, LampTypeOther}
}

// IsValidRunStatus 校验运行状态取值是否合法。
func IsValidRunStatus(status string) bool {
	for _, item := range RunStatuses() {
		if item == status {
			return true
		}
	}
	return false
}

// Lamp 路灯台账, 记录每盏路灯的基础档案与当前运行状态。
type Lamp struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Code        string     `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name        string     `gorm:"size:128" json:"name"`
	RoadName    string     `gorm:"size:128;index;not null" json:"road_name"`
	District    string     `gorm:"size:64;index" json:"district"`
	Address     string     `gorm:"size:255" json:"address"`
	Longitude   float64    `json:"longitude"`
	Latitude    float64    `json:"latitude"`
	LampType    string     `gorm:"size:32;index" json:"lamp_type"`
	Power       int        `json:"power"`
	PoleHeight  float64    `json:"pole_height"`
	RunStatus   string     `gorm:"size:32;index;not null;default:normal" json:"run_status"`
	InstallDate *time.Time `gorm:"type:date" json:"install_date"`
	Remark      string     `gorm:"size:255" json:"remark"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName 指定表名。
func (Lamp) TableName() string { return "lamp" }
