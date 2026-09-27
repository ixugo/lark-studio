package wails

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/ixugo/vdub/internal/app"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/data"
	"github.com/ixugo/vdub/internal/web/api"
)

//go:embed appicon.png
var appIcon []byte

// RunApp 启动基于 Wails3 的单一二进制一体化桌面应用。
func RunApp(bc *conf.Bootstrap, assets fs.FS) error {
	db, err := data.SetupDB(bc)
	if err != nil {
		return err
	}

	taskCore := api.NewTaskCore(db)
	termCore := api.NewTermCore(db)
	recipeCore := api.NewRecipeCore(db)

	eventHub := NewWailsEventHub(nil)
	scheduler, asrRouter, cleanupScheduler := app.NewPipelineSchedulerWithASR(bc, taskCore, termCore, eventHub)
	defer cleanupScheduler()

	// 启动配置文件热重载监听
	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()
	go conf.WatchConfig(watchCtx, bc)

	svc := NewAppService(nil, bc, taskCore, termCore, recipeCore, scheduler, eventHub)
	svc.SetASRRouter(asrRouter)

	// 自动恢复因程序终止、关机等情况中断的任务，使其自动往后跑
	go svc.AutoResumeInterruptedTasks(context.Background())

	wailsApp := application.New(application.Options{
		Name:        "lark-studio",
		Description: "Lark Studio Video Translation & Dubbing Desktop Client",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	svc.app = wailsApp
	eventHub.SetApp(wailsApp)

	mainWindow := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:           "Lark Studio",
		Width:           1260,
		Height:          840,
		MinWidth:        960,
		MinHeight:       640,
		EnableFileDrop:  true,
		DevToolsEnabled: true,
		InitialPosition: application.WindowCentered,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
	})

	mainWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		fmt.Printf("[Wails] 捕获文件拖放事件: %v\n", files)
		if len(files) > 0 {
			wailsApp.Event.Emit("files-dropped", files)
			wailsApp.Event.Emit("common:WindowFilesDropped", files)
			if payload, err := json.Marshal(files); err == nil {
				mainWindow.ExecJS(fmt.Sprintf(`
					if (window.__onWailsFilesDropped) {
						window.__onWailsFilesDropped(%s);
					}
					window.dispatchEvent(new CustomEvent('wails:files-dropped', {detail: %s}));
				`, payload, payload))
			}
		}
	})

	return wailsApp.Run()
}
