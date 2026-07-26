//go:build wireinject

package app

import (
	"log/slog"
	"net/http"

	"github.com/google/wire"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/data"
	"github.com/ixugo/vdub/internal/web/api"
)

func WireApp(bc *conf.Bootstrap, log *slog.Logger) (http.Handler, func(), error) {
	panic(wire.Build(data.ProviderSet, api.ProviderVersionSet, api.ProviderSet))
}
