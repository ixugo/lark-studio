package recipe

import (
	"context"
	"time"
)

// Recipe 工作流配方实体，用于保存用户自定义的流水线组合与参数配置。
type Recipe struct {
	ID               string    `gorm:"primaryKey" json:"id"`
	Title            string    `gorm:"column:title;notNull;default:''" json:"title"`
	Subtitle         string    `gorm:"column:subtitle;notNull;default:''" json:"subtitle"`
	Badge            string    `gorm:"column:badge;notNull;default:''" json:"badge"`
	IsCustom         bool      `gorm:"column:is_custom;notNull;default:true" json:"is_custom"`
	DoSub            bool      `gorm:"column:do_sub;notNull;default:true" json:"do_sub"`
	DoTranslate      bool      `gorm:"column:do_translate;notNull;default:true" json:"do_translate"`
	DoDub            bool      `gorm:"column:do_dub;notNull;default:true" json:"do_dub"`
	DoVideo          bool      `gorm:"column:do_video;notNull;default:true" json:"do_video"`
	TargetLang       string    `gorm:"column:target_lang;notNull;default:''" json:"target_lang"`
	SourceLang       string    `gorm:"column:source_lang;notNull;default:'auto'" json:"source_lang"`
	WhisperModel     string    `gorm:"column:whisper_model;notNull;default:''" json:"whisper_model"`
	TranslateService string    `gorm:"column:translate_service;notNull;default:''" json:"translate_service"`
	TTSEngine        string    `gorm:"column:tts_engine;notNull;default:'edge'" json:"tts_engine"`
	TTSVoice         string    `gorm:"column:tts_voice;notNull;default:''" json:"tts_voice"`
	SpeechRate       float64   `gorm:"column:speech_rate;notNull;default:1" json:"speech_rate"`
	SubtitleOutput   string    `gorm:"column:subtitle_output;notNull;default:'soft'" json:"subtitle_output"`
	SubtitleStyle    string    `gorm:"column:subtitle_style;notNull;default:''" json:"subtitle_style"`
	VideoQuality     string    `gorm:"column:video_quality;notNull;default:''" json:"video_quality"`
	CreatedAt        time.Time `gorm:"column:created_at;notNull;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at;notNull;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定 SQLite 存储表名。
func (*Recipe) TableName() string {
	return "recipes"
}

// Storer 配方持久化接口。
type Storer interface {
	ListRecipes(ctx context.Context) ([]Recipe, error)
	GetRecipe(ctx context.Context, id string) (*Recipe, error)
	SaveRecipe(ctx context.Context, r *Recipe) error
	DeleteRecipe(ctx context.Context, id string) error
}

// Core 配方业务核心。
type Core struct {
	store Storer
}

// NewCore 创建配方业务核心实例。
func NewCore(store Storer) Core {
	return Core{store: store}
}

// List 列出所有配方。
func (c Core) List(ctx context.Context) ([]Recipe, error) {
	return c.store.ListRecipes(ctx)
}

// Save 保存或更新配方。
func (c Core) Save(ctx context.Context, r *Recipe) error {
	return c.store.SaveRecipe(ctx, r)
}

// Delete 删除指定配方。
func (c Core) Delete(ctx context.Context, id string) error {
	return c.store.DeleteRecipe(ctx, id)
}
