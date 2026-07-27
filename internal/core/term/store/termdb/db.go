package termdb

import (
	"context"

	"github.com/ixugo/vdub/internal/core/term"
	"gorm.io/gorm"
)

var _ term.Storer = DB{}

// DB 术语 SQLite 存储实现
type DB struct {
	db *gorm.DB
}

// NewDB 创建存储实例
func NewDB(db *gorm.DB) DB {
	return DB{db: db}
}

// AutoMigrate 自动迁移表结构
func (d DB) AutoMigrate(ok bool) DB {
	if !ok {
		return d
	}
	if err := d.db.AutoMigrate(&term.Term{}); err != nil {
		panic(err)
	}
	return d
}

// List 查询全部术语，按创建时间倒序
func (d DB) List(ctx context.Context) ([]term.Term, error) {
	var terms []term.Term
	err := d.db.WithContext(ctx).Order("created_at DESC").Find(&terms).Error
	return terms, err
}

// Create 新增术语，唯一索引冲突时返回错误
func (d DB) Create(ctx context.Context, t *term.Term) error {
	return d.db.WithContext(ctx).Create(t).Error
}

// Delete 按 ID 删除术语
func (d DB) Delete(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&term.Term{}, id).Error
}
