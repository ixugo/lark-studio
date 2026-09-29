package wails

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRemoteModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer api-key" {
			t.Errorf("请求不符: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"data":[{"id":"remote-asr"}]}`)
	}))
	defer srv.Close()
	got, err := (&AppService{}).ListRemoteModels(srv.URL+"/v1", "api-key")
	if err != nil || len(got) != 1 || got[0].ID != "remote-asr" {
		t.Fatalf("查询结果 = %#v, %v", got, err)
	}
	if _, err := (&AppService{}).ListRemoteModels("", ""); err == nil {
		t.Fatal("非法地址应拒绝")
	}
}
