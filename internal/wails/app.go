package wails

import (
	"context"
	"io/fs"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/ixugo/vdub/internal/app"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/data"
	"github.com/ixugo/vdub/internal/web/api"
)

// RunApp 启动基于 Wails3 的单一二进制一体化桌面应用。
func RunApp(bc *conf.Bootstrap, assets fs.FS) error {
	db, err := data.SetupDB(bc)
	if err != nil {
		return err
	}

	taskCore := api.NewTaskCore(db)
	termCore := api.NewTermCore(db)

	eventHub := NewWailsEventHub(nil)
	scheduler, cleanupScheduler := app.NewPipelineScheduler(bc, taskCore, termCore, eventHub)
	defer cleanupScheduler()

	// 启动配置文件热重载监听
	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()
	go conf.WatchConfig(watchCtx, bc, nil)

	svc := NewAppService(nil, bc, taskCore, termCore, scheduler)

	wailsApp := application.New(application.Options{
		Name:        "VDub",
		Description: "Video Translation & Dubbing Desktop Client",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	svc.app = wailsApp
	eventHub.app = wailsApp

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "VDub",
		Width:          1260,
		Height:         840,
		MinWidth:       960,
		MinHeight:      640,
		EnableFileDrop: true,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
	})

	return wailsApp.Run()
}
