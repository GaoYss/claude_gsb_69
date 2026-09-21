package database

import (
	"context"

	"gorm.io/gorm"
)

// txContextKey 是事务句柄在 context 中的键。
type txContextKey struct{}

// Transaction 在 fn 执行期间开启数据库事务: fn 返回错误时整体回滚, 否则提交。
// 事务句柄通过 context 传递, 各仓储的 session 方法会优先复用它,
// 因此跨模块的多次写入可以纳入同一个事务, 要么全部生效, 要么全部回退。
func Transaction(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txContextKey{}, tx))
	})
}

// Session 返回当前上下文应使用的数据库句柄: 存在事务时复用事务, 否则回退到默认连接。
func Session(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txContextKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return db.WithContext(ctx)
}
