package term

import (
	"context"
	"fmt"
	"strings"
)

// Storer 术语持久化接口
type Storer interface {
	List(ctx context.Context) ([]Term, error)
	Create(ctx context.Context, t *Term) error
	Delete(ctx context.Context, id int64) error
}

// Core 术语业务核心
type Core struct {
	store Storer
}

// NewCore 创建术语核心
func NewCore(store Storer) Core {
	return Core{store: store}
}

// ListAll 返回全部术语
func (c Core) ListAll(ctx context.Context) ([]Term, error) {
	return c.store.List(ctx)
}

// ListMappings 返回全部术语映射（供翻译流水线使用）
func (c Core) ListMappings(ctx context.Context) ([]Term, error) {
	return c.store.List(ctx)
}

// Add 添加术语映射，空白或重复时返回错误
// translation 为空时默认与 text 相同（保持原文）
func (c Core) Add(ctx context.Context, text, translation string) (*Term, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("术语不能为空")
	}
	translation = strings.TrimSpace(translation)
	if translation == "" {
		translation = text
	}
	t := &Term{Text: text, Translation: translation}
	if err := c.store.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("添加术语失败: %w", err)
	}
	return t, nil
}

// Remove 删除指定 ID 的术语
func (c Core) Remove(ctx context.Context, id int64) error {
	return c.store.Delete(ctx, id)
}
