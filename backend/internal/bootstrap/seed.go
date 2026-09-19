package bootstrap

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/internal/modules/repair"
)

const hour = time.Hour

// seedRepairCase 描述一条演示维修记录, 时间字段为距当前时刻的时长。
type seedRepairCase struct {
	repairman   string
	team        string
	startedAgo  time.Duration
	finishedAgo time.Duration // 为 0 表示仍在维修中
	result      string
	content     string
	materials   string
	cost        float64
}

// seedFaultCase 描述一条演示故障记录。
type seedFaultCase struct {
	lampIndex   int
	faultType   string
	level       string
	source      string
	description string
	reporter    string
	reportedAgo time.Duration
	status      string
	closed      bool
	repairs     []seedRepairCase
}

// seed 在数据库为空时写入演示数据, 便于启动后立即体验完整业务流程。
func seed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&lamp.Lamp{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now()
	lamps := buildSeedLamps(now)
	if err := db.Create(&lamps).Error; err != nil {
		return fmt.Errorf("写入路灯台账演示数据失败: %w", err)
	}

	cases := seedFaultCases()
	faults := make([]fault.Fault, 0, len(cases))
	sequences := map[string]int{}
	for _, item := range cases {
		device := lamps[item.lampIndex]
		reportedAt := now.Add(-item.reportedAgo)
		prefix := "GD" + reportedAt.Format("20060102")
		sequences[prefix]++

		faults = append(faults, fault.Fault{
			FaultNo:       fmt.Sprintf("%s%04d", prefix, sequences[prefix]),
			LampID:        device.ID,
			LampCode:      device.Code,
			RoadName:      device.RoadName,
			FaultType:     item.faultType,
			FaultLevel:    item.level,
			Source:        item.source,
			Description:   item.description,
			Reporter:      item.reporter,
			ReporterPhone: "13800001234",
			ReportedAt:    reportedAt,
			Status:        item.status,
		})
	}
	if err := db.Create(&faults).Error; err != nil {
		return fmt.Errorf("写入故障演示数据失败: %w", err)
	}

	repairs := make([]repair.Repair, 0)
	repairRanges := make([][2]int, len(cases))
	sequences = map[string]int{}
	for index, item := range cases {
		device := lamps[item.lampIndex]
		start := len(repairs)
		for _, expect := range item.repairs {
			startedAt := now.Add(-expect.startedAgo)
			prefix := "WX" + startedAt.Format("20060102")
			sequences[prefix]++

			record := repair.Repair{
				RepairNo:     fmt.Sprintf("%s%04d", prefix, sequences[prefix]),
				FaultID:      faults[index].ID,
				FaultNo:      faults[index].FaultNo,
				LampID:       device.ID,
				LampCode:     device.Code,
				Repairman:    expect.repairman,
				RepairTeam:   expect.team,
				ContactPhone: "13900005678",
				StartedAt:    startedAt,
				Status:       repair.StatusOngoing,
				Content:      expect.content,
				Materials:    expect.materials,
				Cost:         expect.cost,
			}
			if expect.finishedAgo > 0 {
				finishedAt := now.Add(-expect.finishedAgo)
				record.FinishedAt = &finishedAt
				record.Status = repair.StatusFinished
				record.Result = expect.result
			}
			repairs = append(repairs, record)
		}
		repairRanges[index] = [2]int{start, len(repairs)}
	}
	if len(repairs) > 0 {
		if err := db.Create(&repairs).Error; err != nil {
			return fmt.Errorf("写入维修记录演示数据失败: %w", err)
		}
	}

	for index, item := range cases {
		start, end := repairRanges[index][0], repairRanges[index][1]
		columns := map[string]any{"repair_count": end - start}
		if end > start {
			columns["latest_repair_id"] = repairs[end-1].ID
		}
		if item.closed {
			closedAt := now.Add(-item.reportedAgo / 2)
			columns["closed_at"] = closedAt
			columns["close_remark"] = "现场已恢复照明并复核确认, 故障闭环"
		}
		if err := db.Model(&fault.Fault{}).Where("id = ?", faults[index].ID).Updates(columns).Error; err != nil {
			return fmt.Errorf("回填故障演示数据失败: %w", err)
		}
	}

	if err := syncSeedLampStatus(db, faults, lamps); err != nil {
		return err
	}

	slog.Info("演示数据初始化完成",
		"路灯", len(lamps),
		"故障", len(faults),
		"维修记录", len(repairs),
	)
	return nil
}

// buildSeedLamps 生成 6 条道路共 30 盏路灯的台账数据。
func buildSeedLamps(now time.Time) []lamp.Lamp {
	roads := []struct {
		name     string
		district string
	}{
		{"中山路", "城东区"},
		{"建设大道", "城东区"},
		{"园区北路", "高新区"},
		{"滨江路", "城西区"},
		{"解放路", "城西区"},
		{"学院路", "高新区"},
	}
	types := []string{lamp.LampTypeLED, lamp.LampTypeSodium, lamp.LampTypeMetal, lamp.LampTypeSolar}
	powers := []int{60, 100, 150, 200, 250}

	result := make([]lamp.Lamp, 0, len(roads)*5)
	index := 0
	for roadIndex, road := range roads {
		for position := 0; position < 5; position++ {
			index++
			installDate := time.Date(
				now.Year()-1-roadIndex%2, time.Month(1+position), 15,
				0, 0, 0, 0, now.Location(),
			)
			result = append(result, lamp.Lamp{
				Code:        fmt.Sprintf("LD-%05d", index),
				Name:        fmt.Sprintf("%s%d号灯杆", road.name, position+1),
				RoadName:    road.name,
				District:    road.district,
				Address:     fmt.Sprintf("%s%d号", road.name, (position+1)*100),
				Longitude:   116.40 + float64(roadIndex)*0.01 + float64(position)*0.001,
				Latitude:    39.90 + float64(roadIndex)*0.01 + float64(position)*0.001,
				LampType:    types[(roadIndex+position)%len(types)],
				Power:       powers[(roadIndex+position)%len(powers)],
				PoleHeight:  8 + float64(position%3)*1.5,
				RunStatus:   lamp.RunStatusNormal,
				InstallDate: &installDate,
				Remark:      "演示数据",
			})
		}
	}
	return result
}

// seedFaultCases 返回演示故障场景, 覆盖待处理 / 维修中 / 已修复 / 已关闭四种状态。
func seedFaultCases() []seedFaultCase {
	return []seedFaultCase{
		{
			lampIndex: 0, faultType: "灯不亮", level: fault.LevelHigh, source: fault.SourceInspection,
			description: "夜间巡检发现整灯不亮, 相邻灯杆照明正常", reporter: "王建国",
			reportedAgo: 6 * hour, status: fault.StatusPending,
		},
		{
			lampIndex: 1, faultType: "灯光闪烁", level: fault.LevelNormal, source: fault.SourceCitizen,
			description: "市民来电反馈该路段灯光持续闪烁, 影响行车视线", reporter: "李梅",
			reportedAgo: 30 * hour, status: fault.StatusPending,
		},
		{
			lampIndex: 2, faultType: "灯具常亮", level: fault.LevelLow, source: fault.SourceMonitoring,
			description: "控制平台监测到该灯具白天仍处于点亮状态", reporter: "监控中心",
			reportedAgo: 40 * hour, status: fault.StatusPending,
		},
		{
			lampIndex: 3, faultType: "线路故障", level: fault.LevelUrgent, source: fault.SourceInspection,
			description: "电缆接头烧蚀, 该支路 3 盏路灯同时失电", reporter: "赵强",
			reportedAgo: 3 * hour, status: fault.StatusProcessing,
			repairs: []seedRepairCase{
				{
					repairman: "刘志强", team: "市政照明一班", startedAgo: 2 * hour,
					content: "已到场排查, 确认电缆接头烧蚀, 正在更换接头", materials: "防水接头 2 套",
					cost: 180,
				},
			},
		},
		{
			lampIndex: 4, faultType: "控制箱故障", level: fault.LevelHigh, source: fault.SourceMonitoring,
			description: "控制箱通讯中断, 远程无法下发开关灯指令", reporter: "监控中心",
			reportedAgo: 5 * hour, status: fault.StatusProcessing,
			repairs: []seedRepairCase{
				{
					repairman: "陈鹏", team: "市政照明二班", startedAgo: 1 * hour,
					content: "检查控制箱通讯模块, 疑似模块损坏", materials: "通讯模块 1 个", cost: 260,
				},
			},
		},
		{
			lampIndex: 5, faultType: "灯不亮", level: fault.LevelNormal, source: fault.SourceCitizen,
			description: "该灯杆夜间不亮, 疑似驱动电源损坏", reporter: "张伟",
			reportedAgo: 26 * hour, status: fault.StatusRepaired,
			repairs: []seedRepairCase{
				{
					repairman: "刘志强", team: "市政照明一班", startedAgo: 25 * hour, finishedAgo: 20 * hour,
					result: repair.ResultFixed, content: "更换 LED 驱动电源并复测绝缘",
					materials: "驱动电源 1 个", cost: 220,
				},
			},
		},
		{
			lampIndex: 6, faultType: "灯具破损", level: fault.LevelNormal, source: fault.SourceInspection,
			description: "灯具外罩被外物击碎, 需整体更换灯头", reporter: "王建国",
			reportedAgo: 50 * hour, status: fault.StatusRepaired,
			repairs: []seedRepairCase{
				{
					repairman: "周涛", team: "市政照明二班", startedAgo: 48 * hour, finishedAgo: 44 * hour,
					result: repair.ResultFixed, content: "更换灯头总成并密封处理",
					materials: "LED 灯头 1 套", cost: 460,
				},
			},
		},
		{
			lampIndex: 7, faultType: "灯杆倾斜", level: fault.LevelHigh, source: fault.SourceCitizen,
			description: "车辆剐蹭导致灯杆倾斜约 8 度, 存在安全隐患", reporter: "孙倩",
			reportedAgo: 72 * hour, status: fault.StatusClosed, closed: true,
			repairs: []seedRepairCase{
				{
					repairman: "周涛", team: "市政照明二班", startedAgo: 70 * hour, finishedAgo: 60 * hour,
					result: repair.ResultFixed, content: "重新浇筑基础法兰并校正灯杆垂直度",
					materials: "基础法兰 1 套", cost: 980,
				},
			},
		},
		{
			lampIndex: 8, faultType: "灯不亮", level: fault.LevelNormal, source: fault.SourceMonitoring,
			description: "平台告警该灯杆回路电流为零", reporter: "监控中心",
			reportedAgo: 96 * hour, status: fault.StatusClosed, closed: true,
			repairs: []seedRepairCase{
				{
					repairman: "陈鹏", team: "市政照明一班", startedAgo: 94 * hour, finishedAgo: 90 * hour,
					result: repair.ResultFixed, content: "更换熔断器并紧固接线端子",
					materials: "熔断器 1 只", cost: 60,
				},
			},
		},
		{
			lampIndex: 9, faultType: "灯光闪烁", level: fault.LevelNormal, source: fault.SourceInspection,
			description: "灯具有明显频闪, 疑似驱动电源老化", reporter: "王建国",
			reportedAgo: 120 * hour, status: fault.StatusClosed, closed: true,
			repairs: []seedRepairCase{
				{
					repairman: "刘志强", team: "市政照明一班", startedAgo: 118 * hour, finishedAgo: 112 * hour,
					result: repair.ResultFixed, content: "更换驱动电源, 频闪消除",
					materials: "驱动电源 1 个", cost: 220,
				},
			},
		},
		{
			lampIndex: 10, faultType: "线路故障", level: fault.LevelUrgent, source: fault.SourceInspection,
			description: "地埋电缆绝缘老化, 绝缘电阻不达标", reporter: "赵强",
			reportedAgo: 150 * hour, status: fault.StatusClosed, closed: true,
			repairs: []seedRepairCase{
				{
					repairman: "周涛", team: "市政照明二班", startedAgo: 148 * hour, finishedAgo: 140 * hour,
					result: repair.ResultPendingParts, content: "检测确认需整段更换电缆, 等待物料到场",
					materials: "电缆 40 米", cost: 120,
				},
				{
					repairman: "周涛", team: "市政照明二班", startedAgo: 130 * hour, finishedAgo: 120 * hour,
					result: repair.ResultFixed, content: "更换老化电缆并做绝缘测试, 测试合格",
					materials: "电缆 40 米, 热缩管 4 套", cost: 1560,
				},
			},
		},
		{
			lampIndex: 11, faultType: "灯具常亮", level: fault.LevelLow, source: fault.SourceOther,
			description: "白天常亮, 疑似接触器粘连", reporter: "社区网格员",
			reportedAgo: 10 * hour, status: fault.StatusRepaired,
			repairs: []seedRepairCase{
				{
					repairman: "陈鹏", team: "市政照明二班", startedAgo: 9 * hour, finishedAgo: 7 * hour,
					result: repair.ResultFixed, content: "更换接触器, 恢复远程开关灯控制",
					materials: "交流接触器 1 只", cost: 150,
				},
			},
		},
		{
			lampIndex: 12, faultType: "灯不亮", level: fault.LevelNormal, source: fault.SourceCitizen,
			description: "居民反映该灯杆连续两晚不亮", reporter: "李梅",
			reportedAgo: 8 * hour, status: fault.StatusPending,
		},
		{
			lampIndex: 13, faultType: "控制箱故障", level: fault.LevelHigh, source: fault.SourceMonitoring,
			description: "控制箱电流异常波动, 疑似内部接触不良", reporter: "监控中心",
			reportedAgo: 15 * hour, status: fault.StatusProcessing,
			repairs: []seedRepairCase{
				{
					repairman: "刘志强", team: "市政照明一班", startedAgo: 12 * hour,
					content: "拆检控制箱, 正在逐路测量回路电流", materials: "万用表、绝缘胶带", cost: 40,
				},
			},
		},
	}
}

// syncSeedLampStatus 依据演示故障数据回填路灯运行状态, 保证台账与故障一致。
func syncSeedLampStatus(db *gorm.DB, faults []fault.Fault, lamps []lamp.Lamp) error {
	statuses := map[uint]string{}
	for _, item := range faults {
		switch item.Status {
		case fault.StatusProcessing:
			statuses[item.LampID] = lamp.RunStatusMaintenance
		case fault.StatusPending:
			if statuses[item.LampID] != lamp.RunStatusMaintenance {
				statuses[item.LampID] = lamp.RunStatusFault
			}
		}
	}

	for index, device := range lamps {
		if index >= 18 && index%5 == 4 {
			statuses[device.ID] = lamp.RunStatusOffline
		}
	}

	for lampID, status := range statuses {
		err := db.Model(&lamp.Lamp{}).Where("id = ?", lampID).Update("run_status", status).Error
		if err != nil {
			return fmt.Errorf("同步演示路灯运行状态失败: %w", err)
		}
	}
	return nil
}
