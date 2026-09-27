package recipe_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/core/recipe"
	"github.com/ixugo/vdub/internal/core/recipe/store/recipedb"
	"gorm.io/gorm"
)

// TestRecipePersistenceAndLifecycle 验证配方在 SQLite 中的创建、查询与删除流程。
func TestRecipePersistenceAndLifecycle(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接内存数据库失败: %v", err)
	}

	store := recipedb.NewDB(db).AutoMigrate(true)
	core := recipe.NewCore(store)
	ctx := context.Background()

	item := &recipe.Recipe{
		ID:             "rcp_test_1",
		Title:          "科技视频精配",
		Subtitle:       "全流程译制",
		Badge:          "我的配方",
		IsCustom:       true,
		DoSub:          true,
		DoTranslate:    true,
		DoDub:          true,
		DoVideo:        true,
		TargetLang:     "en",
		TTSVoice:       "en-US-JennyNeural",
		SpeechRate:     1.0,
		SubtitleOutput: "soft",
	}

	if err := core.Save(ctx, item); err != nil {
		t.Fatalf("保存配方失败: %v", err)
	}

	list, err := core.List(ctx)
	if err != nil {
		t.Fatalf("查询配方列表失败: %v", err)
	}
	if len(list) != 1 || list[0].Title != "科技视频精配" {
		t.Fatalf("配方查询结果异常: %+v", list)
	}

	if err := core.Delete(ctx, "rcp_test_1"); err != nil {
		t.Fatalf("删除配方失败: %v", err)
	}

	listAfter, err := core.List(ctx)
	if err != nil {
		t.Fatalf("查询配方列表失败: %v", err)
	}
	if len(listAfter) != 0 {
		t.Fatalf("配方未被删除成功: %+v", listAfter)
	}
}
