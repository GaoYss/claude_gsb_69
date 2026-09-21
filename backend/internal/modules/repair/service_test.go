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
	lamps   *lamp.Service
	faults  *fault.Service
	repairs *repair.Service
	status  *status.Service
	db      *gorm.DB
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
		db:      db,
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

// startRepair 登记一条维修开工记录, 便于删除流程测试复用。
func (h *harness) startRepair(t *testing.T, faultID uint, repairman string) *repair.Repair {
	t.Helper()
	record, err := h.repairs.Create(context.Background(), repair.CreateRequest{
		FaultID: faultID, Repairman: repairman,
	})
	require.NoError(t, err)
	return record
}

// TestDeleteLastOngoingRepairRevertsFaultAndLamp 删除最后一条(进行中)维修记录:
// 故障回到待处理, 维修次数清零, 最近一次维修清空, 路灯由维修中回到故障状态。
func TestDeleteLastOngoingRepairRevertsFaultAndLamp(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-101")
	entity := h.createFault(t, device.ID, "驱动电源故障")

	record := h.startRepair(t, entity.ID, "维修工甲")

	require.NoError(t, h.repairs.Delete(ctx, record.ID))

	got, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusPending, got.Status, "删掉最后一次维修后故障应回到待处理")
	require.Equal(t, 0, got.RepairCount, "维修次数应回退为 0")
	require.Nil(t, got.LatestRepairID, "最近一次维修应清空")

	lampState, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusFault, lampState.RunStatus, "路灯应从维修中回到故障状态")

	// 回到待处理后允许重新开工, 验证状态机确实回退而非停在旧值
	again := h.startRepair(t, entity.ID, "维修工乙")
	require.Equal(t, repair.StatusOngoing, again.Status)
}

// TestDeleteFinishedFixedRepairReopensFault 删除已修复完工的最后一条维修记录:
// 故障不能停在"已修复", 应回到待处理, 路灯回到故障状态。
func TestDeleteFinishedFixedRepairReopensFault(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-102")
	entity := h.createFault(t, device.ID, "灯具破损")

	record := h.startRepair(t, entity.ID, "维修工甲")
	_, err := h.repairs.Finish(ctx, record.ID, repair.FinishRequest{Result: repair.ResultFixed})
	require.NoError(t, err)

	before, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusRepaired, before.Status)
	lampBefore, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusNormal, lampBefore.RunStatus)

	require.NoError(t, h.repairs.Delete(ctx, record.ID))

	got, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusPending, got.Status, "删除唯一的已修复维修后故障应回到待处理, 不能停留在已修复")
	require.Equal(t, 0, got.RepairCount)
	require.Nil(t, got.LatestRepairID)

	lampAfter, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusFault, lampAfter.RunStatus)
}

// TestDeleteLatestRepairRestoresEarlierState 删除最近一条维修但仍有历史记录时:
// 次数减一、最近一次维修指向剩余最新记录, 状态按该记录回退为维修中。
func TestDeleteLatestRepairRestoresEarlierState(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-103")
	entity := h.createFault(t, device.ID, "线路老化")

	first := h.startRepair(t, entity.ID, "维修工甲")
	// 第一次维修以待配件完工, 故障保持维修中
	firstFinishedAt := time.Now().Add(time.Hour).Format("2006-01-02 15:04:05")
	_, err := h.repairs.Finish(ctx, first.ID, repair.FinishRequest{
		Result: repair.ResultPendingParts, FinishedAt: firstFinishedAt,
	})
	require.NoError(t, err)

	// 物料到场后第二次开工
	second := h.startRepair(t, entity.ID, "维修工乙")

	require.NoError(t, h.repairs.Delete(ctx, second.ID))

	got, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, 1, got.RepairCount, "维修次数应从 2 回退到 1")
	require.NotNil(t, got.LatestRepairID)
	require.Equal(t, first.ID, *got.LatestRepairID, "最近一次维修应指回剩余的第一条记录")
	require.Equal(t, fault.StatusProcessing, got.Status, "最近一次维修未修复, 故障应回到维修中")

	lampState, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusMaintenance, lampState.RunStatus)
}

// TestDeleteRepairUpdatesBoardAggregates 删除维修记录后看板的维修次数、完工数与平均耗时同步变化。
func TestDeleteRepairUpdatesBoardAggregates(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-104")
	entity := h.createFault(t, device.ID, "控制器故障")

	record := h.startRepair(t, entity.ID, "维修工甲")
	finishedAt := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := h.repairs.Finish(ctx, record.ID, repair.FinishRequest{
		Result: repair.ResultFixed, FinishedAt: finishedAt,
	})
	require.NoError(t, err)

	before, err := h.status.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), before.Repair.Total)
	require.Equal(t, int64(1), before.Repair.FinishedTotal)
	require.GreaterOrEqual(t, before.Repair.AverageDurationHr, 1.99)

	require.NoError(t, h.repairs.Delete(ctx, record.ID))

	after, err := h.status.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), after.Repair.Total, "看板维修次数应随删除回退")
	require.Equal(t, int64(0), after.Repair.FinishedTotal)
	require.Equal(t, 0.0, after.Repair.AverageDurationHr, "看板平均维修耗时应随删除回退")
}

// TestDeleteRepairRejectedOnClosedFault 已关闭故障的维修记录不允许删除, 数据保持不变。
func TestDeleteRepairRejectedOnClosedFault(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-105")
	entity := h.createFault(t, device.ID, "误报复核")

	record := h.startRepair(t, entity.ID, "维修工甲")
	_, err := h.repairs.Finish(ctx, record.ID, repair.FinishRequest{Result: repair.ResultFixed})
	require.NoError(t, err)
	_, err = h.faults.Close(ctx, entity.ID, fault.CloseRequest{Remark: "闭环"})
	require.NoError(t, err)

	requireConflict(t, h.repairs.Delete(ctx, record.ID))

	_, err = h.repairs.Get(ctx, record.ID)
	require.NoError(t, err, "删除被拒绝后维修记录应仍然存在")
}

// failDeleteFaultPort 在故障回退环节强制失败, 用于验证删除动作整体回滚。
type failDeleteFaultPort struct {
	*fault.Service
}

func (failDeleteFaultPort) OnRepairDeleted(context.Context, uint, int, *uint, string) error {
	return errors.New("模拟故障状态回退失败")
}

// TestDeleteRollsBackWhenSyncFails 回退故障/路灯失败时, 已执行的删除必须随事务一起回滚,
// 不能只删了维修记录而其它字段停在旧值。
func TestDeleteRollsBackWhenSyncFails(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-T-106")
	entity := h.createFault(t, device.ID, "驱动电源故障")
	record := h.startRepair(t, entity.ID, "维修工甲")

	failing := repair.NewService(h.db, repair.NewRepository(h.db), failDeleteFaultPort{h.faults})
	err := failing.Delete(ctx, record.ID)
	require.Error(t, err, "回退失败时删除接口必须返回错误, 不能吞掉")

	// 维修记录因事务回滚仍然存在
	stillExists, err := h.repairs.Get(ctx, record.ID)
	require.NoError(t, err)
	require.Equal(t, record.ID, stillExists.ID)

	// 故障与路灯字段维持删除前的值
	got, err := h.faults.GetByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, fault.StatusProcessing, got.Status)
	require.Equal(t, 1, got.RepairCount)
	require.NotNil(t, got.LatestRepairID)
	require.Equal(t, record.ID, *got.LatestRepairID)

	lampState, err := h.lamps.Get(ctx, device.ID)
	require.NoError(t, err)
	require.Equal(t, lamp.RunStatusMaintenance, lampState.RunStatus)
}
