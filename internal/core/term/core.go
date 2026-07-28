package term

import (
	"context"
	"fmt"
	"strings"
)

// Storer 术语持久化接口
type Storer interface {
	// Glossary
	ListGlossaries(ctx context.Context) ([]Glossary, error)
	CreateGlossary(ctx context.Context, g *Glossary) error
	UpdateGlossary(ctx context.Context, g *Glossary) error
	DeleteGlossary(ctx context.Context, id int64) error

	// Term
	ListByGlossary(ctx context.Context, glossaryID int64) ([]Term, error)
	SearchTerms(ctx context.Context, glossaryID int64, query string) ([]Term, error)
	ListEnabledTerms(ctx context.Context) ([]Term, error)
	CreateTerm(ctx context.Context, t *Term) error
	UpdateTerm(ctx context.Context, t *Term) error
	DeleteTerm(ctx context.Context, id int64) error
}

// Core 术语业务核心
type Core struct {
	store Storer
}

// NewCore 创建术语核心
func NewCore(store Storer) Core {
	return Core{store: store}
}

// EnsureDefaultGlossary 确保至少存在一个词库，不存在则创建"默认词库"
func (c Core) EnsureDefaultGlossary(ctx context.Context) error {
	list, err := c.store.ListGlossaries(ctx)
	if err != nil {
		return fmt.Errorf("查询词库失败: %w", err)
	}
	if len(list) > 0 {
		return nil
	}
	g := &Glossary{Name: "默认词库", Enabled: true}
	return c.store.CreateGlossary(ctx, g)
}

// ---- Glossary ----

// ListGlossaries 返回全部词库（按 priority 降序）
func (c Core) ListGlossaries(ctx context.Context) ([]Glossary, error) {
	return c.store.ListGlossaries(ctx)
}

// CreateGlossary 创建词库
func (c Core) CreateGlossary(ctx context.Context, name string) (*Glossary, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("词库名称不能为空")
	}
	g := &Glossary{Name: name, Enabled: true}
	if err := c.store.CreateGlossary(ctx, g); err != nil {
		return nil, fmt.Errorf("创建词库失败: %w", err)
	}
	return g, nil
}

// UpdateGlossary 更新词库（名称/启用/优先级）
func (c Core) UpdateGlossary(ctx context.Context, g *Glossary) error {
	return c.store.UpdateGlossary(ctx, g)
}

// DeleteGlossary 删除词库及其下全部词条
func (c Core) DeleteGlossary(ctx context.Context, id int64) error {
	return c.store.DeleteGlossary(ctx, id)
}

// ---- Term ----

// ListByGlossary 获取指定词库的全部词条
func (c Core) ListByGlossary(ctx context.Context, glossaryID int64) ([]Term, error) {
	return c.store.ListByGlossary(ctx, glossaryID)
}

// SearchTerms 在指定词库中搜索词条（按 text/translation/note 模糊匹配）
func (c Core) SearchTerms(ctx context.Context, glossaryID int64, query string) ([]Term, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return c.store.ListByGlossary(ctx, glossaryID)
	}
	return c.store.SearchTerms(ctx, glossaryID, query)
}

// ListMappings 返回全部启用词库的词条（供翻译流水线使用）
func (c Core) ListMappings(ctx context.Context) ([]Term, error) {
	return c.store.ListEnabledTerms(ctx)
}

// ListAll 兼容旧接口，返回全部启用词库的词条
func (c Core) ListAll(ctx context.Context) ([]Term, error) {
	return c.store.ListEnabledTerms(ctx)
}

// Add 添加词条到指定词库
// translation 为空时默认与 text 相同（保持原文）
func (c Core) Add(ctx context.Context, glossaryID int64, text, translation, note string) (*Term, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("术语不能为空")
	}
	translation = strings.TrimSpace(translation)
	if translation == "" {
		translation = text
	}
	t := &Term{
		GlossaryID:  glossaryID,
		Text:        text,
		Translation: translation,
		Note:        strings.TrimSpace(note),
	}
	if err := c.store.CreateTerm(ctx, t); err != nil {
		return nil, fmt.Errorf("添加术语失败: %w", err)
	}
	return t, nil
}

// Update 更新指定词条的原文、译文、备注
func (c Core) Update(ctx context.Context, id int64, text, translation, note string) (*Term, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("术语不能为空")
	}
	translation = strings.TrimSpace(translation)
	if translation == "" {
		translation = text
	}
	t := &Term{
		ID:          id,
		Text:        text,
		Translation: translation,
		Note:        strings.TrimSpace(note),
	}
	if err := c.store.UpdateTerm(ctx, t); err != nil {
		return nil, fmt.Errorf("更新术语失败: %w", err)
	}
	return t, nil
}

// Remove 删除指定 ID 的术语
func (c Core) Remove(ctx context.Context, id int64) error {
	return c.store.DeleteTerm(ctx, id)
}
