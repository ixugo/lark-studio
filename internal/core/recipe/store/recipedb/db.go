package recipedb

import (
	"context"

	"github.com/ixugo/vdub/internal/core/recipe"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ recipe.Storer = DB{}

// DB 配方 SQLite 存储实现。
type DB struct {
	db *gorm.DB
}

// NewDB 创建配方数据库存储实例。
func NewDB(db *gorm.DB) DB {
	return DB{db: db}
}

// AutoMigrate 自动迁移配方表结构。
func (d DB) AutoMigrate(ok bool) DB {
	if !ok {
		return d
	}
	if err := d.db.AutoMigrate(&recipe.Recipe{}); err != nil {
		panic(err)
	}
	return d
}

// ListRecipes 查询全部配方（按创建时间降序）。
func (d DB) ListRecipes(ctx context.Context) ([]recipe.Recipe, error) {
	var list []recipe.Recipe
	err := d.db.WithContext(ctx).Order("created_at DESC").Find(&list).Error
	return list, err
}

// GetRecipe 根据 ID 查询单个配方。
func (d DB) GetRecipe(ctx context.Context, id string) (*recipe.Recipe, error) {
	var r recipe.Recipe
	if err := d.db.WithContext(ctx).First(&r, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// SaveRecipe 保存或更新配方（主键冲突时覆盖更新）。
func (d DB) SaveRecipe(ctx context.Context, r *recipe.Recipe) error {
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(r).Error
}

// DeleteRecipe 根据 ID 删除配方。
func (d DB) DeleteRecipe(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&recipe.Recipe{}, "id = ?", id).Error
}
