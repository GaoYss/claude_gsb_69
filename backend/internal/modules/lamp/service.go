package lamp

import (
	"context"
	"strings"
	"time"

	"streetlight/internal/apperr"
	"streetlight/pkg/pagination"
)

// lampSortSpec 定义路灯列表接口允许的排序字段白名单。
var lampSortSpec = pagination.SortSpec{
	Allowed: map[string]string{
		"code":         "code",
		"name":         "name",
		"road_name":    "road_name",
		"lamp_type":    "lamp_type",
		"run_status":   "run_status",
		"install_date": "install_date",
		"created_at":   "created_at",
		"updated_at":   "updated_at",
	},
	Default: "id",
}

// OpenFaultCounter 由故障模块实现, 用于删除路灯前校验是否仍有未关闭故障。
type OpenFaultCounter interface {
	CountOpenByLamp(ctx context.Context, lampID uint) (int64, error)
}

// Service 承载路灯台账的业务规则。
type Service struct {
	repo   *Repository
	faults OpenFaultCounter
}

// NewService 构造路灯台账服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// SetOpenFaultCounter 注入未关闭故障计数器。
// 在 bootstrap 中装配, 以避免路灯模块与故障模块之间的构造顺序耦合。
func (s *Service) SetOpenFaultCounter(counter OpenFaultCounter) {
	s.faults = counter
}

// Get 查询路灯详情。
func (s *Service) Get(ctx context.Context, id uint) (*Lamp, error) {
	return s.repo.GetByID(ctx, id)
}

// List 分页查询路灯台账, 同时返回归一化后的分页信息供响应封装使用。
func (s *Service) List(ctx context.Context, filter ListQuery) ([]Lamp, int64, pagination.Query, error) {
	page := pagination.Parse(filter.Params, lampSortSpec)
	items, total, err := s.repo.List(ctx, filter, page)
	if err != nil {
		return nil, 0, page, err
	}
	return items, total, page, nil
}

// Create 新增路灯台账, 路灯编号全局唯一。
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Lamp, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, apperr.BadRequest("路灯编号不能为空")
	}

	exists, err := s.repo.ExistsByCode(ctx, code, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperr.Conflict("路灯编号已存在: %s", code)
	}

	installDate, err := parseDate(req.InstallDate)
	if err != nil {
		return nil, err
	}

	runStatus := strings.TrimSpace(req.RunStatus)
	if runStatus == "" {
		runStatus = RunStatusNormal
	}
	if !IsValidRunStatus(runStatus) {
		return nil, apperr.BadRequest("非法的路灯运行状态: %s", runStatus)
	}

	roadName := strings.TrimSpace(req.RoadName)
	if roadName == "" {
		return nil, apperr.BadRequest("所在道路不能为空")
	}

	entity := &Lamp{
		Code:        code,
		Name:        strings.TrimSpace(req.Name),
		RoadName:    roadName,
		District:    strings.TrimSpace(req.District),
		Address:     strings.TrimSpace(req.Address),
		Longitude:   valueOrFloat(req.Longitude, 0),
		Latitude:    valueOrFloat(req.Latitude, 0),
		LampType:    strings.TrimSpace(req.LampType),
		Power:       valueOrInt(req.Power, 0),
		PoleHeight:  valueOrFloat(req.PoleHeight, 0),
		RunStatus:   runStatus,
		InstallDate: installDate,
		Remark:      strings.TrimSpace(req.Remark),
	}

	if err := s.repo.Create(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// Update 更新路灯台账, 仅覆盖请求中显式提交的字段。
func (s *Service) Update(ctx context.Context, id uint, req UpdateRequest) (*Lamp, error) {
	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if code == "" {
			return nil, apperr.BadRequest("路灯编号不能为空")
		}
		if code != entity.Code {
			exists, err := s.repo.ExistsByCode(ctx, code, id)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, apperr.Conflict("路灯编号已存在: %s", code)
			}
			entity.Code = code
		}
	}
	if req.Name != nil {
		entity.Name = strings.TrimSpace(*req.Name)
	}
	if req.RoadName != nil {
		roadName := strings.TrimSpace(*req.RoadName)
		if roadName == "" {
			return nil, apperr.BadRequest("所在道路不能为空")
		}
		entity.RoadName = roadName
	}
	if req.District != nil {
		entity.District = strings.TrimSpace(*req.District)
	}
	if req.Address != nil {
		entity.Address = strings.TrimSpace(*req.Address)
	}
	if req.Longitude != nil {
		entity.Longitude = *req.Longitude
	}
	if req.Latitude != nil {
		entity.Latitude = *req.Latitude
	}
	if req.LampType != nil {
		entity.LampType = strings.TrimSpace(*req.LampType)
	}
	if req.Power != nil {
		entity.Power = *req.Power
	}
	if req.PoleHeight != nil {
		entity.PoleHeight = *req.PoleHeight
	}
	if req.RunStatus != nil {
		status := strings.TrimSpace(*req.RunStatus)
		if !IsValidRunStatus(status) {
			return nil, apperr.BadRequest("非法的路灯运行状态: %s", status)
		}
		entity.RunStatus = status
	}
	if req.InstallDate != nil {
		installDate, err := parseDate(*req.InstallDate)
		if err != nil {
			return nil, err
		}
		entity.InstallDate = installDate
	}
	if req.Remark != nil {
		entity.Remark = strings.TrimSpace(*req.Remark)
	}

	if err := s.repo.Update(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// Delete 删除路灯, 存在未关闭故障时拒绝删除, 保证故障链路数据完整。
func (s *Service) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	if s.faults != nil {
		count, err := s.faults.CountOpenByLamp(ctx, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return apperr.Conflict("该路灯存在 %d 条未关闭的故障记录, 请先处理后再删除", count)
		}
	}
	return s.repo.Delete(ctx, id)
}

// Statistics 汇总路灯台账统计信息。
func (s *Service) Statistics(ctx context.Context) (*Statistics, error) {
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}
	byRunStatus, err := s.repo.CountByColumn(ctx, "run_status")
	if err != nil {
		return nil, err
	}
	byLampType, err := s.repo.CountByColumn(ctx, "lamp_type")
	if err != nil {
		return nil, err
	}
	roadCount, err := s.repo.CountDistinct(ctx, "road_name")
	if err != nil {
		return nil, err
	}
	return &Statistics{
		Total:       total,
		RoadCount:   roadCount,
		ByRunStatus: byRunStatus,
		ByLampType:  byLampType,
	}, nil
}

// Options 返回下拉选项与建议的下一个路灯编号。
func (s *Service) Options(ctx context.Context) (*Options, error) {
	roads, err := s.repo.DistinctValues(ctx, "road_name")
	if err != nil {
		return nil, err
	}
	districts, err := s.repo.DistinctValues(ctx, "district")
	if err != nil {
		return nil, err
	}
	nextCode, err := s.repo.NextCode(ctx)
	if err != nil {
		return nil, err
	}
	return &Options{
		Roads:       roads,
		Districts:   districts,
		LampTypes:   LampTypes(),
		RunStatuses: RunStatuses(),
		NextCode:    nextCode,
	}, nil
}

// UpdateRunStatus 更新路灯运行状态, 供故障与维修模块联动调用。
func (s *Service) UpdateRunStatus(ctx context.Context, id uint, status string) error {
	if !IsValidRunStatus(status) {
		return apperr.BadRequest("非法的路灯运行状态: %s", status)
	}
	return s.repo.UpdateRunStatus(ctx, id, status)
}

// parseDate 解析 YYYY-MM-DD 格式的日期, 空字符串表示不设置。
func parseDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	date, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, apperr.BadRequest("日期格式应为 YYYY-MM-DD, 当前值: %s", value)
	}
	return &date, nil
}

func valueOrInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func valueOrFloat(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}
