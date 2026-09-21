package repair

import (
	"context"
	"strings"
	"time"

	"streetlight/internal/apperr"
	"streetlight/internal/modules/fault"
	"streetlight/pkg/pagination"
)

// repairSortSpec 定义维修记录列表允许的排序字段白名单。
var repairSortSpec = pagination.SortSpec{
	Allowed: map[string]string{
		"repair_no":   "repair_no",
		"started_at":  "started_at",
		"finished_at": "finished_at",
		"status":      "status",
		"result":      "result",
		"cost":        "cost",
		"created_at":  "created_at",
	},
	Default: "started_at",
}

// FaultPort 由故障登记模块实现, 维修模块通过它联动故障状态与路灯状态。
type FaultPort interface {
	GetByID(ctx context.Context, id uint) (*fault.Fault, error)
	OnRepairStarted(ctx context.Context, faultID uint, repairID uint) error
	OnRepairFinished(ctx context.Context, faultID uint, fixed bool) error
	SyncRepairStats(ctx context.Context, faultID uint, recap fault.RepairRecap) error
}

// Service 承载维修记录录入的业务规则。
type Service struct {
	repo   *Repository
	faults FaultPort
}

// NewService 构造维修记录服务。
func NewService(repo *Repository, faults FaultPort) *Service {
	return &Service{repo: repo, faults: faults}
}

// Get 查询维修记录详情。
func (s *Service) Get(ctx context.Context, id uint) (*Repair, error) {
	return s.repo.GetByID(ctx, id)
}

// List 分页查询维修记录。
func (s *Service) List(ctx context.Context, query ListQuery) ([]Repair, int64, pagination.Query, error) {
	page := pagination.Parse(query.Params, repairSortSpec)
	filter, err := buildFilter(query)
	if err != nil {
		return nil, 0, page, err
	}
	items, total, err := s.repo.List(ctx, filter, page)
	if err != nil {
		return nil, 0, page, err
	}
	return items, total, page, nil
}

// ListByFault 查询某条故障的维修过程记录。
func (s *Service) ListByFault(ctx context.Context, faultID uint) ([]Repair, error) {
	if _, err := s.faults.GetByID(ctx, faultID); err != nil {
		return nil, err
	}
	return s.repo.ListByFault(ctx, faultID)
}

// Create 录入维修记录(维修开工), 并联动故障与路灯状态。
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Repair, error) {
	target, err := s.faults.GetByID(ctx, req.FaultID)
	if err != nil {
		return nil, err
	}
	if target.Status == fault.StatusClosed {
		return nil, apperr.Conflict("故障 %s 已关闭, 不允许再登记维修记录", target.FaultNo)
	}
	if target.Status == fault.StatusRepaired {
		return nil, apperr.Conflict("故障 %s 已修复, 如需返修请先登记新的维修记录并重新开工", target.FaultNo)
	}

	ongoing, err := s.repo.GetOngoingByFault(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	if ongoing != nil {
		return nil, apperr.Conflict("故障 %s 已有进行中的维修记录 %s, 请先完成后再录入", target.FaultNo, ongoing.RepairNo)
	}

	repairman := strings.TrimSpace(req.Repairman)
	if repairman == "" {
		return nil, apperr.BadRequest("维修人员不能为空")
	}

	startedAt, err := parseTime(req.StartedAt, time.Now())
	if err != nil {
		return nil, err
	}
	if startedAt.Before(target.ReportedAt) {
		return nil, apperr.BadRequest("开工时间不能早于故障上报时间 %s", target.ReportedAt.Format("2006-01-02 15:04:05"))
	}

	entity := &Repair{
		FaultID:      target.ID,
		FaultNo:      target.FaultNo,
		LampID:       target.LampID,
		LampCode:     target.LampCode,
		Repairman:    repairman,
		RepairTeam:   strings.TrimSpace(req.RepairTeam),
		ContactPhone: strings.TrimSpace(req.ContactPhone),
		StartedAt:    startedAt,
		Status:       StatusOngoing,
		Content:      strings.TrimSpace(req.Content),
		Materials:    strings.TrimSpace(req.Materials),
		Cost:         valueOrZero(req.Cost),
		Remark:       strings.TrimSpace(req.Remark),
	}

	if err := s.repo.CreateWithUniqueNo(ctx, entity, "WX"+startedAt.Format("20060102")); err != nil {
		return nil, err
	}

	// 开工后: 故障转为维修中, 路灯转为维修状态
	if err := s.faults.OnRepairStarted(ctx, target.ID, entity.ID); err != nil {
		return nil, err
	}

	entity.FillDuration()
	return entity, nil
}

// Update 修改维修记录, 已完成的记录不允许修改。
func (s *Service) Update(ctx context.Context, id uint, req UpdateRequest) (*Repair, error) {
	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entity.Status == StatusFinished {
		return nil, apperr.Conflict("维修记录 %s 已完成, 不允许修改", entity.RepairNo)
	}

	if req.Repairman != nil {
		repairman := strings.TrimSpace(*req.Repairman)
		if repairman == "" {
			return nil, apperr.BadRequest("维修人员不能为空")
		}
		entity.Repairman = repairman
	}
	if req.RepairTeam != nil {
		entity.RepairTeam = strings.TrimSpace(*req.RepairTeam)
	}
	if req.ContactPhone != nil {
		entity.ContactPhone = strings.TrimSpace(*req.ContactPhone)
	}
	if req.StartedAt != nil {
		startedAt, err := parseTime(*req.StartedAt, entity.StartedAt)
		if err != nil {
			return nil, err
		}
		entity.StartedAt = startedAt
	}
	if req.Content != nil {
		entity.Content = strings.TrimSpace(*req.Content)
	}
	if req.Materials != nil {
		entity.Materials = strings.TrimSpace(*req.Materials)
	}
	if req.Cost != nil {
		entity.Cost = *req.Cost
	}
	if req.Remark != nil {
		entity.Remark = strings.TrimSpace(*req.Remark)
	}

	if err := s.repo.Update(ctx, entity); err != nil {
		return nil, err
	}
	entity.FillDuration()
	return entity, nil
}

// Finish 完成维修: 记录结果与完工时间, 结果为已修复时联动故障转为已修复。
func (s *Service) Finish(ctx context.Context, id uint, req FinishRequest) (*Repair, error) {
	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entity.Status == StatusFinished {
		return nil, apperr.Conflict("维修记录 %s 已完成, 不允许重复提交", entity.RepairNo)
	}

	result := strings.TrimSpace(req.Result)
	if !IsValidResult(result) {
		return nil, apperr.BadRequest("非法的维修结果: %s", result)
	}

	finishedAt, err := parseTime(req.FinishedAt, time.Now())
	if err != nil {
		return nil, err
	}
	if finishedAt.Before(entity.StartedAt) {
		return nil, apperr.BadRequest("完工时间不能早于开工时间")
	}

	entity.FinishedAt = &finishedAt
	entity.Status = StatusFinished
	entity.Result = result
	if content := strings.TrimSpace(req.Content); content != "" {
		entity.Content = content
	}
	if materials := strings.TrimSpace(req.Materials); materials != "" {
		entity.Materials = materials
	}
	if req.Cost != nil {
		entity.Cost = *req.Cost
	}
	if remark := strings.TrimSpace(req.Remark); remark != "" {
		entity.Remark = remark
	}

	if err := s.repo.Update(ctx, entity); err != nil {
		return nil, err
	}

	if err := s.faults.OnRepairFinished(ctx, entity.FaultID, result == ResultFixed); err != nil {
		return nil, err
	}

	entity.FillDuration()
	return entity, nil
}

// Delete 删除维修记录, 并在同一事务内回退故障的维修次数、最近一次维修与处理状态,
// 同时联动路灯运行状态; 已关闭故障的维修记录不允许删除。
// 整个流程要么全部生效, 要么整体回滚, 不会出现只删了记录而统计停在旧值的情况。
func (s *Service) Delete(ctx context.Context, id uint) error {
	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	target, err := s.faults.GetByID(ctx, entity.FaultID)
	if err != nil {
		return err
	}
	if target.Status == fault.StatusClosed {
		return apperr.Conflict("故障 %s 已关闭, 不允许删除其维修记录", target.FaultNo)
	}

	return s.repo.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.Delete(txCtx, id); err != nil {
			return err
		}
		remaining, err := s.repo.ListByFault(txCtx, entity.FaultID)
		if err != nil {
			return err
		}
		return s.faults.SyncRepairStats(txCtx, entity.FaultID, buildRepairRecap(remaining))
	})
}

// buildRepairRecap 统计故障剩余维修记录的概况, 供故障模块回推处理状态。
// repairs 按开工时间正序排列, 因此最后一条即最近一次维修。
func buildRepairRecap(repairs []Repair) fault.RepairRecap {
	recap := fault.RepairRecap{Count: len(repairs)}
	if len(repairs) == 0 {
		return recap
	}
	latestID := repairs[len(repairs)-1].ID
	recap.LatestRepairID = &latestID

	latestFinished := -1
	for index := range repairs {
		switch repairs[index].Status {
		case StatusOngoing:
			recap.HasOngoing = true
		case StatusFinished:
			latestFinished = index
		}
	}
	if latestFinished >= 0 {
		recap.LatestFixed = repairs[latestFinished].Result == ResultFixed
	}
	return recap
}

// Metadata 返回维修模块字典。
func (s *Service) Metadata(ctx context.Context) (*Meta, error) {
	repairmen, err := s.repo.DistinctValues(ctx, "repairman")
	if err != nil {
		return nil, err
	}
	teams, err := s.repo.DistinctValues(ctx, "repair_team")
	if err != nil {
		return nil, err
	}
	return &Meta{
		Statuses:  Statuses(),
		Results:   Results(),
		Repairmen: repairmen,
		Teams:     teams,
	}, nil
}

// Statistics 汇总维修统计信息。
func (s *Service) Statistics(ctx context.Context) (*Statistics, error) {
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}
	byStatus, err := s.repo.CountByColumn(ctx, "status")
	if err != nil {
		return nil, err
	}
	totalCost, err := s.repo.SumCost(ctx)
	if err != nil {
		return nil, err
	}
	averageDuration, err := s.repo.AverageDurationHours(ctx)
	if err != nil {
		return nil, err
	}

	result := &Statistics{
		Total:             total,
		OngoingTotal:      byStatus[StatusOngoing],
		FinishedTotal:     byStatus[StatusFinished],
		TotalCost:         totalCost,
		AverageDurationHr: averageDuration,
	}
	if result.FinishedTotal > 0 {
		result.AverageCost = totalCost / float64(result.FinishedTotal)
	}
	return result, nil
}

// buildFilter 将查询参数转换为仓储条件并解析日期区间。
func buildFilter(query ListQuery) (Filter, error) {
	filter := Filter{
		Keyword:    strings.TrimSpace(query.Keyword),
		FaultID:    query.FaultID,
		LampID:     query.LampID,
		Repairman:  strings.TrimSpace(query.Repairman),
		RepairTeam: strings.TrimSpace(query.RepairTeam),
		Status:     strings.TrimSpace(query.Status),
		Result:     strings.TrimSpace(query.Result),
	}
	if filter.Status != "" && filter.Status != StatusOngoing && filter.Status != StatusFinished {
		return filter, apperr.BadRequest("非法的维修状态: %s", filter.Status)
	}
	if filter.Result != "" && !IsValidResult(filter.Result) {
		return filter, apperr.BadRequest("非法的维修结果: %s", filter.Result)
	}

	if value := strings.TrimSpace(query.StartDate); value != "" {
		from, err := parseDay(value)
		if err != nil {
			return filter, err
		}
		filter.StartedFrom = &from
	}
	if value := strings.TrimSpace(query.EndDate); value != "" {
		to, err := parseDay(value)
		if err != nil {
			return filter, err
		}
		to = to.AddDate(0, 0, 1)
		filter.StartedTo = &to
	}
	if filter.StartedFrom != nil && filter.StartedTo != nil && filter.StartedTo.Before(*filter.StartedFrom) {
		return filter, apperr.BadRequest("结束日期不能早于开始日期")
	}
	return filter, nil
}

// parseTime 解析时间字符串, 为空时返回 fallback。
func parseTime(value string, fallback time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, apperr.BadRequest("时间格式不正确, 建议使用 YYYY-MM-DD HH:mm:ss: %s", value)
}

// parseDay 解析 YYYY-MM-DD 日期, 返回当天零点。
func parseDay(value string) (time.Time, error) {
	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), time.Local)
	if err != nil {
		return time.Time{}, apperr.BadRequest("日期格式应为 YYYY-MM-DD, 当前值: %s", value)
	}
	return date, nil
}

func valueOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
