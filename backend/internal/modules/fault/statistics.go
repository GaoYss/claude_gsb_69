package fault

import (
	"context"
	"fmt"
	"time"
)

// Count 统计故障总数。
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&Fault{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计故障总数失败: %w", err)
	}
	return total, nil
}

// CountOpen 统计未闭环(待处理 + 维修中)的故障数量。
func (r *Repository) CountOpen(ctx context.Context) (int64, error) {
	var total int64
	err := r.session(ctx).Model(&Fault{}).
		Where("status IN ?", []string{StatusPending, StatusProcessing}).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("统计未闭环故障失败: %w", err)
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
	err := r.session(ctx).Model(&Fault{}).
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

// CountReportedBetween 统计上报时间落在 [from, to) 区间内的故障数量。
func (r *Repository) CountReportedBetween(ctx context.Context, from, to time.Time) (int64, error) {
	var total int64
	err := r.session(ctx).Model(&Fault{}).
		Where("reported_at >= ? AND reported_at < ?", from, to).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("统计区间故障数量失败: %w", err)
	}
	return total, nil
}

// CountPendingBefore 统计 before 之前登记且仍未开工的故障数量, 用于超期预警。
func (r *Repository) CountPendingBefore(ctx context.Context, before time.Time) (int64, error) {
	var total int64
	err := r.session(ctx).Model(&Fault{}).
		Where("status = ? AND reported_at < ?", StatusPending, before).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("统计超期未处理故障失败: %w", err)
	}
	return total, nil
}

// ListPendingBefore 查询 before 之前登记且仍未开工的故障。
func (r *Repository) ListPendingBefore(ctx context.Context, before time.Time, limit int) ([]Fault, error) {
	if limit <= 0 {
		limit = 10
	}
	entities := make([]Fault, 0)
	err := r.session(ctx).Model(&Fault{}).
		Where("status = ? AND reported_at < ?", StatusPending, before).
		Order("reported_at ASC, id ASC").
		Limit(limit).
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("查询超期未处理故障失败: %w", err)
	}
	return entities, nil
}

// ListRecent 查询最近登记的故障。
func (r *Repository) ListRecent(ctx context.Context, limit int) ([]Fault, error) {
	if limit <= 0 {
		limit = 10
	}
	entities := make([]Fault, 0)
	err := r.session(ctx).Model(&Fault{}).
		Order("reported_at DESC, id DESC").
		Limit(limit).
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("查询最近故障失败: %w", err)
	}
	return entities, nil
}
