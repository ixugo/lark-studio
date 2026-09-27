package api

import (
	"context"
	"net/http"

	"github.com/ixugo/goddd/domain/uniqueid"
	"github.com/ixugo/goddd/domain/uniqueid/store/uniqueiddb"
	"github.com/ixugo/goddd/pkg/orm"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/recipe"
	"github.com/ixugo/vdub/internal/core/recipe/store/recipedb"
	"github.com/ixugo/vdub/internal/core/term"
	"github.com/ixugo/vdub/internal/core/term/store/termdb"
	"github.com/ixugo/vdub/pkg/web"
	"github.com/ixugo/vdub/pkg/ws"
	"gorm.io/gorm"
)

type Usecase struct {
	Conf    *conf.Bootstrap
	DB      *gorm.DB
	Version VersionAPI

	TaskAPI   TaskAPI
	TermAPI   TermAPI
	Scheduler *pipeline.Scheduler
	Hub       ws.Huber
}

// NewTermCore 创建术语 Core，自动迁移表结构
func NewTermCore(db *gorm.DB) term.Core {
	store := termdb.NewDB(db).AutoMigrate(true)
	core := term.NewCore(store)
	_ = core.EnsureDefaultGlossary(context.Background())
	return core
}

// NewRecipeCore 创建配方 Core，自动迁移配方数据表
func NewRecipeCore(db *gorm.DB) recipe.Core {
	store := recipedb.NewDB(db).AutoMigrate(true)
	return recipe.NewCore(store)
}

// NewHTTPHandler 生成路由，返回 http.Handler。
func NewHTTPHandler(uc *Usecase) http.Handler {
	if !uc.Conf.Runtime.Debug {
		web.SetRelease()
	}
	mux := http.NewServeMux()

	// 404 由 ServeMux 默认处理
	if uc.Conf.Server.HTTP.PProf.Enabled {
		web.SetupPProf(mux, &uc.Conf.Server.HTTP.PProf.AccessIps)
	}

	setupRouter(mux, uc)
	uc.Version.RecordVersion()

	// 中间件链：recover → metrics → logger → body logger → handler
	handler := web.Chain(mux,
		web.Recover(),
		web.Metrics(),
		web.Logger(),
		web.LoggerWithBody(web.DefaultBodyLimit, web.IgnoreBool(!uc.Conf.Runtime.Debug)),
	)
	return handler
}

// NewUniqueID 生成唯一 id
func NewUniqueID(db *gorm.DB) uniqueid.Core {
	store := uniqueiddb.NewDB(db).AutoMigrate(orm.GetEnabledAutoMigrate())
	return uniqueid.NewCore(store, 6)
}
