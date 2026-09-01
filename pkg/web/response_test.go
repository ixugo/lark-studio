package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// reqBind 同时带 form 和 uri tag 的请求结构体
type reqBind struct {
	ID   int    `uri:"id"  form:"id"`
	Name string `uri:"name" form:"name"`
	Page int    `form:"page"`
}

// setupBindRouter 构建测试路由
func setupBindRouter() *http.ServeMux {
	mux := http.NewServeMux()

	handler := func(r *http.Request, req *reqBind) (map[string]any, error) {
		return map[string]any{
			"id":   req.ID,
			"name": req.Name,
			"page": req.Page,
		}, nil
	}

	mux.HandleFunc("GET /items", WrapH(handler))
	mux.HandleFunc("GET /items/{id}/{name}", WrapH(handler))
	mux.HandleFunc("POST /items/{id}/{name}", WrapH(handler))
	mux.HandleFunc("PUT /items/{id}/{name}", WrapH(handler))
	mux.HandleFunc("PATCH /items/{id}/{name}", WrapH(handler))
	mux.HandleFunc("DELETE /items/{id}/{name}", WrapH(handler))

	return mux
}

func TestBind_GetQueryOnly(t *testing.T) {
	r := setupBindRouter()

	req := httptest.NewRequest(http.MethodGet, "/items?id=1&name=alice&page=3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"].(float64) != 1 {
		t.Errorf("id = %v, 期望 1", resp["id"])
	}
	if resp["name"].(string) != "alice" {
		t.Errorf("name = %v, 期望 alice", resp["name"])
	}
	if resp["page"].(float64) != 3 {
		t.Errorf("page = %v, 期望 3", resp["page"])
	}
}

func TestBind_GetWithURI(t *testing.T) {
	r := setupBindRouter()

	req := httptest.NewRequest(http.MethodGet, "/items/42/bob?page=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"].(float64) != 42 {
		t.Errorf("id = %v, 期望 42", resp["id"])
	}
	if resp["name"].(string) != "bob" {
		t.Errorf("name = %v, 期望 bob", resp["name"])
	}
	if resp["page"].(float64) != 5 {
		t.Errorf("page = %v, 期望 5", resp["page"])
	}
}

func TestBind_PostWithURIAndBody(t *testing.T) {
	r := setupBindRouter()

	body := bytes.NewBufferString(`{"id":99,"name":"charlie","page":7}`)
	req := httptest.NewRequest(http.MethodPost, "/items/99/charlie", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"].(float64) != 99 {
		t.Errorf("id = %v, 期望 99", resp["id"])
	}
	if resp["name"].(string) != "charlie" {
		t.Errorf("name = %v, 期望 charlie", resp["name"])
	}
}

func TestBind_PutWithURIAndBody(t *testing.T) {
	r := setupBindRouter()

	body := bytes.NewBufferString(`{"id":11,"name":"dave","page":2}`)
	req := httptest.NewRequest(http.MethodPut, "/items/11/dave", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d, body: %s", w.Code, w.Body.String())
	}
}

func TestBind_DeleteQueryOnly(t *testing.T) {
	r := setupBindRouter()

	req := httptest.NewRequest(http.MethodDelete, "/items/8/frank?page=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"].(float64) != 8 {
		t.Errorf("id = %v, 期望 8", resp["id"])
	}
}
