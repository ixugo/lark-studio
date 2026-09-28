package api

import (
	"expvar"
	"testing"

	"github.com/ixugo/vdub/internal/conf"
)

func TestHealthUsesInjectedRuntimeVersion(t *testing.T) {
	setExpvarString("git_branch", "main")
	setExpvarString("git_hash", "abc1234")

	uc := Usecase{Conf: &conf.Bootstrap{Runtime: conf.Runtime{BuildVersion: "v1.2.3"}}}
	got, err := uc.getHealth(nil, nil)
	if err != nil {
		t.Fatalf("获取健康信息失败: %v", err)
	}
	if got.Version != "v1.2.3" {
		t.Fatalf("健康接口应返回编译注入版本，实际为 %q", got.Version)
	}
	if got.GitBranch != "main" || got.GitHash != "abc1234" {
		t.Fatalf("健康接口构建信息不匹配: %#v", got)
	}
}

func setExpvarString(name, value string) {
	if item := expvar.Get(name); item != nil {
		item.(*expvar.String).Set(value)
		return
	}
	expvar.NewString(name).Set(value)
}
