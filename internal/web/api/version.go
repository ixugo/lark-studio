package api

import (
	"log/slog"
	"net/http"

	"github.com/ixugo/goddd/domain/version"
	"github.com/ixugo/goddd/domain/version/store/versiondb"
	"github.com/ixugo/goddd/pkg/orm"
	"github.com/ixugo/vdub/pkg/web"
	"gorm.io/gorm"
)

// 通过修改版本号来控制是否执行表迁移
var (
	DBVersion = "0.0.1"
	DBRemark  = "debug"
)

// VersionAPI 版本信息 HTTP 接口
type VersionAPI struct {
	versionCore version.Core
}

// NewVersionCore 创建版本 Core
func NewVersionCore(db *gorm.DB) version.Core {
	vdb := versiondb.NewDB(db)
	core := version.NewCore(vdb)
	isOK := core.IsAutoMigrate(DBVersion)
	vdb.AutoMigrate(isOK)
	if isOK {
		slog.Info("更新数据库表结构")
		orm.SetEnabledAutoMigrate(true)
	}
	return core
}

// NewVersionAPI 创建版本 API
func NewVersionAPI(ver version.Core) VersionAPI {
	return VersionAPI{versionCore: ver}
}

// RecordVersion 更新版本号
func (v VersionAPI) RecordVersion() {
	if !orm.GetEnabledAutoMigrate() {
		return
	}
	if err := v.versionCore.RecordVersion(DBVersion, DBRemark); err != nil {
		slog.Error("RecordVersion", "err", err)
	}
}

// RegisterVersion 注册版本路由
func RegisterVersion(mux *http.ServeMux, verAPI VersionAPI) {
	mux.HandleFunc("GET /version", web.WrapH(verAPI.getVersion))
}

func (v VersionAPI) getVersion(_ *http.Request, _ *struct{}) (any, error) {
	return map[string]any{"version": DBVersion, "remark": DBRemark}, nil
}
