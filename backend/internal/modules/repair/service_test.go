package repair_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"streetlight/internal/apperr"
	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/internal/modules/repair"
	"streetlight/internal/modules/status"
)

// harness 使用内存数据库装配真实模块, 用于验证跨模块业务流程。
type harness struct {
	lamps      *lamp.Service
	faults     *fault.Service
	repairs    *repair.Service
	status     *status.Service
	repairRepo *repair.Repository
	db         *gorm.DB
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
	repairService := repair.NewService(repairRepository, faultService)

	return &harness{
		lamps:      lampService,
		faults:     faultService,
		repairs:    repairService,
		status:     status.NewService(db, lampRepository, faultRepository, repairRepository),
		repairRepo: repairRepository,
		db:         db,
	}
}

func (h *harness) createLamp(t *testing.T, code string) *lamp.Lamp {
	t.Helper()
	entity, err := h.lamps.Create(context.Background(), lamp.CreateRequest{
		Code:     code,
		Name:     "测试灯杆",
		RoadName: "测试路",
		LampType: lamp.LampTypeLED,
	})
	require.NoError(t, err)
	return entity
}

func (h *harness) createFault(t *testing.T, lampID uint, description string) *fault.Fault {
	t.Helper()
	entity, err := h.faults.Create(context.Background(), fault.CreateRequest{
		LampID:      lampID,
		FaultType:   "灯不亮",
		FaultLevel:  fault.LevelHigh,
		Source:      fault.SourceInspection,
		Description: description,
		Reporter:    "巡检员",
	})
	require.NoError(t, err)
	return entity
}

// requireConflict 断言错误是 409 业务冲突。
func requireConflict(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	businessErr, ok := apperr.As(err)
	require.True(t, ok, "期望业务错误, 实际: %v", err)
	require.Equal(t, http.StatusConflict, businessErr.Status, "错误信息: %s", businessErr.Message)
}

func TestFaultRepairLifecycle(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-001")

	entity := h.createFault(t, device.ID, "整灯不亮, 疑似驱动电源故障")
	require.Equal(t, fault.StatusPending, entity.Status)
	require.Regexp(t, `^GD\d{8}\d{4}$`, entity.FaultNo)

	afterReport, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusFault, afterReport.RunStatus, "登记故障后路灯应变为故障状态")

	// 同一盏路灯不允许存在多条未闭环故障
	_, err = h.faults.Create(ctx, fault.CreateRequest{
		LampID: device.ID, FaultType: "灯不亮", Description: "重复登记",
	})
	requireConflict(t, err)

	// 维修开工
	record, err := h.repairs.Create(ctx, repair.CreateRequest{
		FaultID: entity.ID, Repairman: "维修工甲", RepairTeam: "市政照明一班",
	})
	require.NoError(t, err)
	require.Equal(t, repair.StatusOngoing, record.Status)
	require.Regexp(t, `^WX\d{8}\d{4}$`, record.RepairNo)

	faultAfterStart, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusProcessing, faultAfterStart.Status)
	require.Equal(t, 1, faultAfterStart.RepairCount)
	require.NotNil(t, faultAfterStart.LatestRepairID)
	require.Equal(t, record.ID, *faultAfterStart.LatestRepairID)

	lampAfterStart, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusMaintenance, lampAfterStart.RunStatus)

	// 同一故障不允许并行开工
	_, err = h.repairs.Create(ctx, repair.CreateRequest{FaultID: entity.ID, Repairman: "维修工乙"})
	requireConflict(t, err)

	// 完工且结果为已修复
	cost := 210.0
	finished, err := h.repairs.Finish(ctx, record.ID, repair.FinishRequest{
		Result: repair.ResultFixed, Content: "更换驱动电源", Cost: &cost,
	})
	require.NoError(t, err)
	require.Equal(t, repair.StatusFinished, finished.Status)
	require.NotNil(t, finished.FinishedAt)

	faultAfterFinish, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusRepaired, faultAfterFinish.Status)

	lampAfterFinish, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusNormal, lampAfterFinish.RunStatus, "修复后路灯应恢复为正常")

	// 关闭故障形成闭环
	closed, err := h.faults.Close(ctx, entity.ID, fault.CloseRequest{Remark: "现场复核通过"})
	require.NoError(t, err)
	require.Equal(t, fault.StatusClosed, closed.Status)
	require.NotNil(t, closed.ClosedAt)

	// 已产生的维修记录使故障不可删除
	requireConflict(t, h.faults.Delete(ctx, entity.ID))
	// 已关闭故障不允许再次关闭
	_, err = h.faults.Close(ctx, entity.ID, fault.CloseRequest{})
	requireConflict(t, err)
}

func TestRepairPendingPartsKeepsFaultProcessing(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-002")
	entity := h.createFault(t, device.ID, "线路老化需要更换电缆")

	record, err := h.repairs.Create(ctx, repair.CreateRequest{
		FaultID: entity.ID, Repairman: "维修工丙",
	})
	require.NoError(t, err)

	_, err = h.repairs.Finish(ctx, record.ID, repair.FinishRequest{Result: repair.ResultPendingParts})
	require.NoError(t, err)

	faultAfterFinish, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusProcessing, faultAfterFinish.Status, "非已修复结果不应结束故障")

	lampAfterFinish, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusMaintenance, lampAfterFinish.RunStatus)

	// 可继续登记第二次维修(返修)
	second, err := h.repairs.Create(ctx, repair.CreateRequest{
		FaultID: entity.ID, Repairman: "维修工丙", Content: "物料到场后更换电缆",
	})
	require.NoError(t, err)
	require.Equal(t, repair.StatusOngoing, second.Status)

	faultAfterSecond, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, 2, faultAfterSecond.RepairCount)
}

func TestRepairRejectedOnClosedFault(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-003")
	entity := h.createFault(t, device.ID, "误报故障需要作废")

	_, err := h.faults.Close(ctx, entity.ID, fault.CloseRequest{Remark: "误报作废"})
	require.NoError(t, err)

	_, err = h.repairs.Create(ctx, repair.CreateRequest{FaultID: entity.ID, Repairman: "维修工丁"})
	requireConflict(t, err)
}

func TestFaultValidation(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-004")

	// 非法故障类型
	_, err := h.faults.Create(ctx, fault.CreateRequest{
		LampID: device.ID, FaultType: "不存在的类型", Description: "测试",
	})
	businessErr, ok := apperr.As(err)
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, businessErr.Status)

	// 路灯不存在
	_, err = h.faults.Create(ctx, fault.CreateRequest{
		LampID: 99999, FaultType: "灯不亮", Description: "测试",
	})
	businessErr, ok = apperr.As(err)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, businessErr.Status)

	// 重复路灯编号
	_, err = h.lamps.Create(ctx, lamp.CreateRequest{
		Code: device.Code, RoadName: "测试路", LampType: lamp.LampTypeLED,
	})
	requireConflict(t, err)

	// 存在未闭环故障时不允许删除路灯
	h.createFault(t, device.ID, "删除校验")
	requireConflict(t, h.lamps.Delete(ctx, device.ID))
}

func TestRepairDeleteRestoresPendingAndStats(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-101")
	entity := h.createFault(t, device.ID, "灯头烧毁")

	record, err := h.repairs.Create(ctx, repair.CreateRequest{FaultID: entity.ID, Repairman: "维修工甲"})
	require.NoError(t, err)

	before, err := h.status.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), before.Repair.Total)
	require.Equal(t, int64(1), before.Fault.ByStatus[fault.StatusProcessing])

	require.NoError(t, h.repairs.Delete(ctx, record.ID))

	// 删掉最后一次维修后: 故障回到待处理, 维修次数与最近一次维修一并回退
	after, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusPending, after.Status)
	require.Equal(t, 0, after.RepairCount)
	require.Nil(t, after.LatestRepairID)

	// 故障仍未闭环, 路灯应回到故障状态而不是维修中
	deviceAfter, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusFault, deviceAfter.RunStatus)

	// 看板统计随删除同步变化
	overview, err := h.status.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), overview.Repair.Total)
	require.Equal(t, int64(1), overview.Fault.ByStatus[fault.StatusPending])
	require.Equal(t, int64(0), overview.Fault.ByStatus[fault.StatusProcessing])
}

func TestRepairDeleteFixedRepairRestoresPending(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-102")
	entity := h.createFault(t, device.ID, "驱动电源故障")

	record, err := h.repairs.Create(ctx, repair.CreateRequest{FaultID: entity.ID, Repairman: "维修工乙"})
	require.NoError(t, err)
	finishedAt := record.StartedAt.Add(4 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = h.repairs.Finish(ctx, record.ID, repair.FinishRequest{Result: repair.ResultFixed, FinishedAt: finishedAt})
	require.NoError(t, err)

	repaired, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusRepaired, repaired.Status)

	before, err := h.status.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), before.Repair.Total)
	require.InDelta(t, 4.0, before.Repair.AverageDurationHr, 0.01)

	require.NoError(t, h.repairs.Delete(ctx, record.ID))

	// 已修复的维修记录被删除后, 故障不应停留在已修复, 而是回到待处理
	after, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusPending, after.Status)
	require.Equal(t, 0, after.RepairCount)
	require.Nil(t, after.LatestRepairID)

	deviceAfter, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusFault, deviceAfter.RunStatus)

	overview, err := h.status.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), overview.Repair.Total)
	require.Equal(t, 0.0, overview.Repair.AverageDurationHr)
}

func TestRepairDeleteKeepsProcessingWithRemainingRepairs(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-103")
	entity := h.createFault(t, device.ID, "电缆老化")

	first, err := h.repairs.Create(ctx, repair.CreateRequest{FaultID: entity.ID, Repairman: "维修工丙"})
	require.NoError(t, err)
	_, err = h.repairs.Finish(ctx, first.ID, repair.FinishRequest{Result: repair.ResultPendingParts})
	require.NoError(t, err)

	second, err := h.repairs.Create(ctx, repair.CreateRequest{FaultID: entity.ID, Repairman: "维修工丙"})
	require.NoError(t, err)

	// 删掉进行中的第二次维修: 仍有未修复的完工记录, 故障保持维修中, 最近一次维修回退到第一条
	require.NoError(t, h.repairs.Delete(ctx, second.ID))

	after, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusProcessing, after.Status)
	require.Equal(t, 1, after.RepairCount)
	require.NotNil(t, after.LatestRepairID)
	require.Equal(t, first.ID, *after.LatestRepairID)

	deviceAfter, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusMaintenance, deviceAfter.RunStatus)

	// 再删掉最后一条: 故障回到待处理, 路灯回到故障状态
	require.NoError(t, h.repairs.Delete(ctx, first.ID))

	final, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusPending, final.Status)
	require.Equal(t, 0, final.RepairCount)
	require.Nil(t, final.LatestRepairID)

	deviceFinal, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusFault, deviceFinal.RunStatus)
}

func TestRepairDeleteRejectedOnClosedFault(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-104")
	entity := h.createFault(t, device.ID, "修复后闭环")

	record, err := h.repairs.Create(ctx, repair.CreateRequest{FaultID: entity.ID, Repairman: "维修工丁"})
	require.NoError(t, err)
	_, err = h.repairs.Finish(ctx, record.ID, repair.FinishRequest{Result: repair.ResultFixed})
	require.NoError(t, err)
	_, err = h.faults.Close(ctx, entity.ID, fault.CloseRequest{Remark: "复核通过"})
	require.NoError(t, err)

	requireConflict(t, h.repairs.Delete(ctx, record.ID))
}

// stubFaultPort 模拟故障联动失败的端口, 用于验证删除流程的事务回滚。
type stubFaultPort struct {
	entity  *fault.Fault
	syncErr error
}

func (s stubFaultPort) GetByID(ctx context.Context, id uint) (*fault.Fault, error) {
	return s.entity, nil
}

func (s stubFaultPort) OnRepairStarted(ctx context.Context, faultID uint, repairID uint) error {
	return nil
}

func (s stubFaultPort) OnRepairFinished(ctx context.Context, faultID uint, fixed bool) error {
	return nil
}

func (s stubFaultPort) SyncRepairStats(ctx context.Context, faultID uint, recap fault.RepairRecap) error {
	return s.syncErr
}

func TestRepairDeleteRollsBackWhenSyncFails(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-105")
	entity := h.createFault(t, device.ID, "同步失败应整体回滚")

	record, err := h.repairs.Create(ctx, repair.CreateRequest{FaultID: entity.ID, Repairman: "维修工戊"})
	require.NoError(t, err)

	fresh, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	broken := repair.NewService(h.repairRepo, stubFaultPort{entity: fresh, syncErr: errors.New("模拟同步失败")})

	err = broken.Delete(ctx, record.ID)
	require.Error(t, err, "联动失败时删除应返回错误")

	// 删除必须整体回滚: 维修记录仍在, 故障统计保持旧值
	kept, err := h.repairs.Get(ctx, record.ID)
	require.NoError(t, err)
	require.Equal(t, record.RepairNo, kept.RepairNo)

	unchanged, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusProcessing, unchanged.Status)
	require.Equal(t, 1, unchanged.RepairCount)
	require.NotNil(t, unchanged.LatestRepairID)
	require.Equal(t, record.ID, *unchanged.LatestRepairID)
}
