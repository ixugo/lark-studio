package wails

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ixugo/goddd/pkg/orm"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
)

type recipeGuardStore struct {
	task.Storer
	item *task.Task
}

func (s recipeGuardStore) Task() task.TaskStorer {
	return recipeGuardTaskStore{item: s.item}
}

type recipeGuardTaskStore struct {
	task.TaskStorer
	item *task.Task
}

func (s recipeGuardTaskStore) Get(_ context.Context, out *task.Task, _ ...orm.QueryOption) error {
	*out = *s.item
	return nil
}

func TestRerunRejectsRecipeBeforeCleanup(t *testing.T) {
	for _, mode := range []int{pipeline.ModeTextTranslate, pipeline.ModeSubtitle} {
		t.Run(string(rune('0'+mode)), func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "src.srt")
			if err := os.WriteFile(file, []byte("original subtitle"), 0o600); err != nil {
				t.Fatal(err)
			}
			item := &task.Task{ID: "test", InputPath: "input.mp4", OutputDir: dir, Mode: pipeline.ModeTranslate, Status: 3}
			svc := &AppService{bc: &conf.Bootstrap{}, taskCore: task.NewCore(recipeGuardStore{item: item})}
			// 调度器和写接口没有注入：无效请求必须在调度或写任务之前返回。
			err := svc.RerunTaskWithRecipe(item.ID, RerunTaskOptions{Mode: mode, FromStep: pipeline.StepWhisper})
			if err == nil || !(strings.Contains(err.Error(), "听写") || strings.Contains(err.Error(), "引擎")) {
				t.Fatalf("应在清理前拒绝不兼容配方或未配置识别引擎：%v", err)
			}
			data, err := os.ReadFile(file)
			if err != nil || string(data) != "original subtitle" || item.Status != 3 {
				t.Fatalf("无效重跑破坏原字幕或任务状态：%q %v %+v", data, err, item)
			}
		})
	}
}
