package api

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"gorm.io/gorm"
)

func TestHTTPBatchSharesOutputRoot(t *testing.T) {
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "tasks.db")))
	if err != nil {
		t.Fatal(err)
	}
	bc := conf.DefaultConfig()
	bc.Pipeline.DefaultOutputDir = filepath.Join(dir, "results")
	bc.Pipeline.WhisperMode = "openai"
	bc.Pipeline.ASRBaseURL = "http://localhost:8000/v1"
	bc.Pipeline.ASRModel = "test-model"
	sched := pipeline.NewScheduler(nil, nil)
	t.Cleanup(sched.Stop)
	service := NewTaskAPI(NewTaskCore(db), sched, &bc)
	var inputs []string
	for _, name := range []string{"one", "two"} {
		input := filepath.Join(dir, name+".mp4")
		if err := os.WriteFile(input, []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, input)
	}
	response, err := service.batchCreateTasks(httptest.NewRequest("POST", "/tasks/batch", nil), &batchCreateInput{Videos: inputs, Mode: pipeline.ModeSubtitle, SubtitleOutput: "burn"})
	if err != nil {
		t.Fatal(err)
	}
	items := response.(map[string]any)["items"].([]*task.Task)
	if len(items) != 2 || items[0].BatchDir != items[1].BatchDir || filepath.Dir(items[0].BatchDir) != bc.Pipeline.DefaultOutputDir || items[0].ResultPath == items[1].ResultPath {
		t.Fatalf("HTTP 批次目录错误: %+v", items)
	}
}
