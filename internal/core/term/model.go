package term

import "time"

// Term 术语映射条目
// Text=源词 Translation=指定译文；二者相同表示保持原文
type Term struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Text        string    `gorm:"uniqueIndex;notNull" json:"text"`
	Translation string    `gorm:"notNull" json:"translation"`
	CreatedAt   time.Time `json:"created_at"`
}

func (*Term) TableName() string { return "terms" }
