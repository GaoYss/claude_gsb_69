package repair

var statusLabels = map[string]string{
	StatusOngoing:  "维修中",
	StatusFinished: "已完成",
}

var resultLabels = map[string]string{
	ResultFixed:        "已修复",
	ResultPendingParts: "待配件",
	ResultObserving:    "观察中",
	ResultUnfixable:    "无法修复",
}

// StatusLabel 返回维修状态的中文名称。
func StatusLabel(status string) string {
	if label, ok := statusLabels[status]; ok {
		return label
	}
	return status
}

// ResultLabel 返回维修结果的中文名称。
func ResultLabel(result string) string {
	if label, ok := resultLabels[result]; ok {
		return label
	}
	return result
}
