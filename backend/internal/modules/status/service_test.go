package status_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/internal/modules/repair"
	"streetlight/internal/modules/status"
)

type harness struct {
	lamps   *lamp.Service
	faults  *fault.Service
	repairs *repair.Service
	status  *status.Service
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&lamp.Lamp{}, &fault.Fault{}, &repair.Repair{}))

	lampRepository := lamp.NewRepository(db)
	lampService := lamp.NewService(lampRepository)

	faultRepository := fault.NewRepository(db)
	faultService := fault.NewService(faultRepository, lampService)
	lampService.SetOpenFaultCounter(faultRepository)

	repairRepository := repair.NewRepository(db)
	repairService := repair.NewService(db, repairRepository, faultService)

	return &harness{
		lamps:   lampService,
		faults:  faultService,
		repairs: repairService,
		status:  status.NewService(db, lampRepository, faultRepository, repairRepository),
	}
}

func (h *harness) createLamp(t *testing.T, code, road string) *lamp.Lamp {
	t.Helper()
	entity, err := h.lamps.Create(context.Background(), lamp.CreateRequest{
		Code:     code,
		RoadName: road,
		LampType: lamp.LampTypeLED,
		Power:    intPointer(120),
	})
	require.NoError(t, err)
	return entity
}

func (h *harness) createFault(t *testing.T, lampID uint, faultType string) *fault.Fault {
	t.Helper()
	entity, err := h.faults.Create(context.Background(), fault.CreateRequest{
		LampID:      lampID,
		FaultType:   faultType,
		FaultLevel:  fault.LevelHigh,
		Description: "状态查询模块测试",
		Reporter:    "巡检员",
	})
	require.NoError(t, err)
	return entity
}

func intPointer(value int) *int { return &value }

func TestOverviewAggregatesBusinessState(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	openLamp := h.createLamp(t, "LD-S-001", "中山路")
	closedLamp := h.createLamp(t, "LD-S-002", "建设大道")
	h.createLamp(t, "LD-S-003", "中山路")

	h.createFault(t, openLamp.ID, "灯不亮")

	closedFault := h.createFault(t, closedLamp.ID, "灯光闪烁")
	record, err := h.repairs.Create(ctx, repair.CreateRequest{
		FaultID: closedFault.ID, Repairman: "维修工甲",
	})
	require.NoError(t, err)
	cost := 180.0
	_, err = h.repairs.Finish(ctx, record.ID, repair.FinishRequest{Result: repair.ResultFixed, Cost: &cost})
	require.NoError(t, err)
	_, err = h.faults.Close(ctx, closedFault.ID, fault.CloseRequest{Remark: "闭环"})
	require.NoError(t, err)

	overview, err := h.status.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(3), overview.Lamp.Total)
	require.Equal(t, int64(2), overview.Lamp.RoadCount)
	require.Equal(t, int64(2), overview.Fault.Total)
	require.Equal(t, int64(1), overview.Fault.OpenTotal)
	require.Equal(t, int64(1), overview.Fault.ByStatus[fault.StatusPending])
	require.Equal(t, int64(1), overview.Fault.ByStatus[fault.StatusClosed])
	require.Equal(t, int64(1), overview.Repair.Total)
	require.Equal(t, int64(0), overview.Repair.OngoingTotal)
	require.Equal(t, int64(1), overview.Repair.FinishedTotal)
	require.Equal(t, cost, overview.Repair.TotalCost)
	require.Equal(t, int64(1), overview.Lamp.ByRunStatus[lamp.RunStatusFault])
	require.Equal(t, int64(2), overview.Lamp.ByRunStatus[lamp.RunStatusNormal])
	require.Len(t, overview.FaultByLevel, len(fault.Levels()))
	require.NotEmpty(t, overview.RecentFaults)
	require.Equal(t, status.OverdueThreshold.Hours(), overview.OverdueHours)
}

func TestLampStatusListAndTrack(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	openLamp := h.createLamp(t, "LD-S-101", "解放路")
	closedLamp := h.createLamp(t, "LD-S-102", "解放路")

	openFault := h.createFault(t, openLamp.ID, "灯不亮")

	closedFault := h.createFault(t, closedLamp.ID, "线路故障")
	record, err := h.repairs.Create(ctx, repair.CreateRequest{
		FaultID: closedFault.ID, Repairman: "维修工乙", RepairTeam: "市政照明二班",
	})
	require.NoError(t, err)
	_, err = h.repairs.Finish(ctx, record.ID, repair.FinishRequest{Result: repair.ResultFixed})
	require.NoError(t, err)
	_, err = h.faults.Close(ctx, closedFault.ID, fault.CloseRequest{Remark: "闭环"})
	require.NoError(t, err)

	rows, total, _, err := h.status.Lamps(ctx, status.LampQuery{OnlyOpen: true})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	require.Equal(t, openLamp.Code, rows[0].LampCode)
	require.Equal(t, int64(1), rows[0].OpenFaults)
	require.Equal(t, int64(1), rows[0].TotalFaults)
	require.Equal(t, openFault.FaultNo, rows[0].FaultNo)
	require.Equal(t, fault.StatusPending, rows[0].FaultStatus)

	allRows, allTotal, _, err := h.status.Lamps(ctx, status.LampQuery{RoadName: "解放路"})
	require.NoError(t, err)
	require.Equal(t, int64(2), allTotal)
	require.Len(t, allRows, 2)

	filtered, filteredTotal, _, err := h.status.Lamps(ctx, status.LampQuery{Keyword: closedLamp.Code})
	require.NoError(t, err)
	require.Equal(t, int64(1), filteredTotal)
	require.Equal(t, record.RepairNo, filtered[0].RepairNo)
	require.Equal(t, repair.ResultFixed, filtered[0].RepairResult)
	require.Equal(t, int64(0), filtered[0].OpenFaults)

	track, err := h.status.Track(ctx, status.TrackQuery{FaultNo: closedFault.FaultNo})
	require.NoError(t, err)
	require.Equal(t, "fault", track.SearchType)
	require.NotNil(t, track.Lamp)
	require.Equal(t, closedLamp.Code, track.Lamp.Code)
	require.Len(t, track.Repairs, 1)
	require.Len(t, track.Timeline, 4)
	require.Equal(t, "reported", track.Timeline[0].Stage)
	require.Equal(t, "repair_started", track.Timeline[1].Stage)
	require.Equal(t, "repair_finished", track.Timeline[2].Stage)
	require.Equal(t, "closed", track.Timeline[3].Stage)

	byLamp, err := h.status.Track(ctx, status.TrackQuery{LampCode: closedLamp.Code})
	require.NoError(t, err)
	require.Equal(t, "lamp", byLamp.SearchType)
	require.Len(t, byLamp.RelatedFaults, 1)
	require.NotNil(t, byLamp.Fault)

	_, err = h.status.Track(ctx, status.TrackQuery{})
	require.Error(t, err, "缺少查询条件时应返回错误")
}
