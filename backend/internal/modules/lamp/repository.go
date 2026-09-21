package lamp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"streetlight/internal/apperr"
	"streetlight/pkg/dbtx"
	"streetlight/pkg/pagination"
)

// Repository 负责路灯台账的数据访问。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造路灯台账仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// session 返回带 context 的数据库句柄; context 中存在事务时复用该事务,
// 使故障联动的运行状态写回与故障变更处于同一事务内。
func (r *Repository) session(ctx context.Context) *gorm.DB {
	if tx, ok := dbtx.FromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

// Create 新增路灯。
func (r *Repository) Create(ctx context.Context, entity *Lamp) error {
	if err := r.session(ctx).Create(entity).Error; err != nil {
		return fmt.Errorf("新增路灯失败: %w", err)
	}
	return nil
}

// Update 保存路灯全部字段。
func (r *Repository) Update(ctx context.Context, entity *Lamp) error {
	if err := r.session(ctx).Save(entity).Error; err != nil {
		return fmt.Errorf("更新路灯失败: %w", err)
	}
	return nil
}

// Delete 按主键删除路灯。
func (r *Repository) Delete(ctx context.Context, id uint) error {
	if err := r.session(ctx).Delete(&Lamp{}, id).Error; err != nil {
		return fmt.Errorf("删除路灯失败: %w", err)
	}
	return nil
}

// GetByID 按主键查询路灯, 不存在时返回 404 业务错误。
func (r *Repository) GetByID(ctx context.Context, id uint) (*Lamp, error) {
	var entity Lamp
	err := r.session(ctx).First(&entity, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("路灯不存在: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询路灯失败: %w", err)
	}
	return &entity, nil
}

// GetByCode 按路灯编号查询。
func (r *Repository) GetByCode(ctx context.Context, code string) (*Lamp, error) {
	var entity Lamp
	err := r.session(ctx).Where("code = ?", strings.TrimSpace(code)).First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("路灯不存在: code=%s", code)
	}
	if err != nil {
		return nil, fmt.Errorf("查询路灯失败: %w", err)
	}
	return &entity, nil
}

// ExistsByCode 判断路灯编号是否已被占用, excludeID 用于更新场景排除自身。
func (r *Repository) ExistsByCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	query := r.session(ctx).Model(&Lamp{}).Where("code = ?", strings.TrimSpace(code))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("校验路灯编号失败: %w", err)
	}
	return count > 0, nil
}

// List 分页查询路灯台账, 返回列表与总数。
func (r *Repository) List(ctx context.Context, filter ListQuery, page pagination.Query) ([]Lamp, int64, error) {
	base := func() *gorm.DB {
		return applyFilters(r.session(ctx).Model(&Lamp{}), filter)
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计路灯总数失败: %w", err)
	}

	entities := make([]Lamp, 0)
	if err := base().Order(page.OrderClause()).Offset(page.Offset()).Limit(page.Limit()).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("查询路灯列表失败: %w", err)
	}
	return entities, total, nil
}

// UpdateRunStatus 更新路灯运行状态, 供故障与维修模块联动调用。
func (r *Repository) UpdateRunStatus(ctx context.Context, id uint, status string) error {
	result := r.session(ctx).Model(&Lamp{}).Where("id = ?", id).Update("run_status", status)
	if result.Error != nil {
		return fmt.Errorf("更新路灯运行状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperr.NotFound("路灯不存在: id=%d", id)
	}
	return nil
}

// Count 统计路灯总数。
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&Lamp{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计路灯总数失败: %w", err)
	}
	return total, nil
}

// CountDistinct 统计某列的去重数量, column 仅允许来自内部常量。
func (r *Repository) CountDistinct(ctx context.Context, column string) (int64, error) {
	var total int64
	err := r.session(ctx).Model(&Lamp{}).
		Where(column + " <> ''").
		Distinct(column).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("统计 %s 数量失败: %w", column, err)
	}
	return total, nil
}

// CountByColumn 按列分组统计, column 仅允许来自内部常量。
func (r *Repository) CountByColumn(ctx context.Context, column string) (map[string]int64, error) {
	type row struct {
		Label string
		Total int64
	}
	rows := make([]row, 0)
	err := r.session(ctx).Model(&Lamp{}).
		Select(column + " AS label, COUNT(*) AS total").
		Group(column).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("分组统计 %s 失败: %w", column, err)
	}

	result := make(map[string]int64, len(rows))
	for _, item := range rows {
		result[item.Label] = item.Total
	}
	return result, nil
}

// DistinctValues 返回某列的去重取值, column 仅允许来自内部常量。
func (r *Repository) DistinctValues(ctx context.Context, column string) ([]string, error) {
	values := make([]string, 0)
	err := r.session(ctx).Model(&Lamp{}).
		Where(column+" <> ''").
		Distinct().
		Order(column).
		Pluck(column, &values).Error
	if err != nil {
		return nil, fmt.Errorf("查询 %s 选项失败: %w", column, err)
	}
	return values, nil
}

// NextCode 依据当前最大编号推算下一个建议编号, 例如 LD-000013。
func (r *Repository) NextCode(ctx context.Context) (string, error) {
	var latest string
	err := r.session(ctx).Model(&Lamp{}).Order("id DESC").Limit(1).Pluck("code", &latest).Error
	if err != nil {
		return "", fmt.Errorf("生成路灯编号失败: %w", err)
	}

	next := 1
	if index := strings.LastIndex(latest, "-"); index >= 0 {
		if value, convErr := strconv.Atoi(latest[index+1:]); convErr == nil {
			next = value + 1
		}
	}
	return fmt.Sprintf("LD-%05d", next), nil
}

// applyFilters 统一拼装列表查询条件。
func applyFilters(statement *gorm.DB, filter ListQuery) *gorm.DB {
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		statement = statement.Where(
			"code LIKE ? OR name LIKE ? OR road_name LIKE ? OR address LIKE ?",
			like, like, like, like,
		)
	}
	if value := strings.TrimSpace(filter.RoadName); value != "" {
		statement = statement.Where("road_name = ?", value)
	}
	if value := strings.TrimSpace(filter.District); value != "" {
		statement = statement.Where("district = ?", value)
	}
	if value := strings.TrimSpace(filter.LampType); value != "" {
		statement = statement.Where("lamp_type = ?", value)
	}
	if value := strings.TrimSpace(filter.RunStatus); value != "" {
		statement = statement.Where("run_status = ?", value)
	}
	return statement
}
