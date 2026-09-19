package fault

var statusLabels = map[string]string{
	StatusPending:    "待处理",
	StatusProcessing: "维修中",
	StatusRepaired:   "已修复",
	StatusClosed:     "已关闭",
}

var levelLabels = map[string]string{
	LevelLow:    "一般",
	LevelNormal: "普通",
	LevelHigh:   "紧急",
	LevelUrgent: "特急",
}

var sourceLabels = map[string]string{
	SourceInspection: "巡检发现",
	SourceCitizen:    "市民上报",
	SourceMonitoring: "系统告警",
	SourceOther:      "其它",
}

// StatusLabel 返回故障状态的中文名称, 未知取值原样返回。
func StatusLabel(status string) string {
	if label, ok := statusLabels[status]; ok {
		return label
	}
	return status
}

// LevelLabel 返回故障等级的中文名称。
func LevelLabel(level string) string {
	if label, ok := levelLabels[level]; ok {
		return label
	}
	return level
}

// SourceLabel 返回故障来源的中文名称。
func SourceLabel(source string) string {
	if label, ok := sourceLabels[source]; ok {
		return label
	}
	return source
}

// isValidFaultType 校验故障类型是否在允许范围内。
func isValidFaultType(faultType string) bool {
	for _, item := range FaultTypes() {
		if item == faultType {
			return true
		}
	}
	return false
}
