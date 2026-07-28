package term

import "time"

// Glossary 词库，一个词库包含多个词条
type Glossary struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"notNull" json:"name"`
	Enabled   bool      `gorm:"notNull;default:true" json:"enabled"`
	Priority  int       `gorm:"notNull;default:0" json:"priority"`
	TermCount int       `gorm:"-" json:"term_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (*Glossary) TableName() string { return "glossaries" }

// Term 术语映射条目
// Text=源词 Translation=指定译文；二者相同表示保持原文
type Term struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	GlossaryID  int64     `gorm:"notNull;default:1;index" json:"glossary_id"`
	Text        string    `gorm:"notNull" json:"text"`
	Translation string    `gorm:"notNull" json:"translation"`
	Note        string    `gorm:"type:text" json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

func (*Term) TableName() string { return "terms" }
