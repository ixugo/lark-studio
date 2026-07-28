package termdb

import (
	"context"
	"fmt"

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
	if err := d.db.AutoMigrate(&term.Glossary{}, &term.Term{}); err != nil {
		panic(err)
	}
	return d
}

// ---- Glossary ----

// ListGlossaries 返回全部词库（按 priority 降序、创建时间升序）
func (d DB) ListGlossaries(ctx context.Context) ([]term.Glossary, error) {
	var glossaries []term.Glossary
	err := d.db.WithContext(ctx).
		Order("priority DESC, created_at ASC").
		Find(&glossaries).Error
	if err != nil {
		return nil, err
	}
	// 填充 term_count
	for i := range glossaries {
		var count int64
		d.db.WithContext(ctx).Model(&term.Term{}).
			Where("glossary_id = ?", glossaries[i].ID).Count(&count)
		glossaries[i].TermCount = int(count)
	}
	return glossaries, nil
}

// CreateGlossary 新增词库
func (d DB) CreateGlossary(ctx context.Context, g *term.Glossary) error {
	return d.db.WithContext(ctx).Create(g).Error
}

// UpdateGlossary 更新词库
func (d DB) UpdateGlossary(ctx context.Context, g *term.Glossary) error {
	return d.db.WithContext(ctx).Save(g).Error
}

// DeleteGlossary 删除词库及其全部词条
func (d DB) DeleteGlossary(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("glossary_id = ?", id).Delete(&term.Term{}).Error; err != nil {
			return fmt.Errorf("删除词条失败: %w", err)
		}
		if err := tx.Delete(&term.Glossary{}, id).Error; err != nil {
			return fmt.Errorf("删除词库失败: %w", err)
		}
		return nil
	})
}

// ---- Term ----

// ListByGlossary 查询指定词库的全部词条
func (d DB) ListByGlossary(ctx context.Context, glossaryID int64) ([]term.Term, error) {
	var terms []term.Term
	err := d.db.WithContext(ctx).
		Where("glossary_id = ?", glossaryID).
		Order("created_at DESC").
		Find(&terms).Error
	return terms, err
}

// SearchTerms 模糊搜索词条（text/translation/note）
func (d DB) SearchTerms(ctx context.Context, glossaryID int64, query string) ([]term.Term, error) {
	var terms []term.Term
	like := "%" + query + "%"
	err := d.db.WithContext(ctx).
		Where("glossary_id = ? AND (text LIKE ? OR translation LIKE ? OR note LIKE ?)",
			glossaryID, like, like, like).
		Order("created_at DESC").
		Find(&terms).Error
	return terms, err
}

// ListEnabledTerms 查询全部启用词库的词条（翻译流水线使用）
func (d DB) ListEnabledTerms(ctx context.Context) ([]term.Term, error) {
	var terms []term.Term
	err := d.db.WithContext(ctx).
		Joins("JOIN glossaries ON glossaries.id = terms.glossary_id").
		Where("glossaries.enabled = ?", true).
		Order("glossaries.priority DESC, terms.created_at DESC").
		Find(&terms).Error
	return terms, err
}

// CreateTerm 新增词条
func (d DB) CreateTerm(ctx context.Context, t *term.Term) error {
	return d.db.WithContext(ctx).Create(t).Error
}

// UpdateTerm 更新词条的原文、译文、备注
func (d DB) UpdateTerm(ctx context.Context, t *term.Term) error {
	return d.db.WithContext(ctx).Model(t).
		Select("text", "translation", "note").
		Updates(t).Error
}

// DeleteTerm 按 ID 删除词条
func (d DB) DeleteTerm(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&term.Term{}, id).Error
}
