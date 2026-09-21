// Package dbtx 提供通过 context 在跨模块调用之间传递数据库事务的能力。
//
// 业务流程经常需要跨模块联动(例如删除维修记录后同步故障与路灯状态),
// 通过 context 携带 *gorm.Tx, 各仓储的 session(ctx) 会自动复用同一事务,
// 从而保证一次业务操作内的全部写入要么一起提交, 要么一起回滚。
package dbtx

import (
	"context"

	"gorm.io/gorm"
)

type txContextKey struct{}

// WithTx 将事务句柄写入 context 并返回新的 context。
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// FromContext 取出 context 中的事务句柄, 不存在时第二个返回值为 false。
func FromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	return tx, ok
}

// InTx 在一个数据库事务中执行 fn, 并把事务经 context 传递给 fn 内的全部仓储调用。
// fn 返回 nil 时事务提交; 返回任意错误(含 panic)时事务回滚, 错误继续向上返回。
func InTx(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(WithTx(ctx, tx))
	})
}
