package api

import (
	"net/http"
	"strconv"

	"github.com/ixugo/goddd/pkg/reason"
	"github.com/ixugo/vdub/internal/core/term"
	"github.com/ixugo/vdub/pkg/web"
)

// TermAPI 术语管理 HTTP 接口
type TermAPI struct {
	core term.Core
}

// NewTermAPI 创建术语 API
func NewTermAPI(core term.Core) TermAPI {
	return TermAPI{core: core}
}

// RegisterTerm 注册术语路由
func RegisterTerm(mux *http.ServeMux, api TermAPI) {
	mux.HandleFunc("GET /glossaries", web.WrapH(api.listGlossaries))
	mux.HandleFunc("POST /glossaries", web.WrapH(api.createGlossary))
	mux.HandleFunc("PUT /glossaries/{id}", web.WrapH(api.updateGlossary))
	mux.HandleFunc("DELETE /glossaries/{id}", web.WrapH(api.deleteGlossary))
	mux.HandleFunc("GET /glossaries/{id}/terms", web.WrapH(api.listTerms))
	mux.HandleFunc("POST /glossaries/{id}/terms", web.WrapH(api.createTerm))
	mux.HandleFunc("PUT /glossaries/{id}/terms/{term_id}", web.WrapH(api.updateTerm))
	mux.HandleFunc("DELETE /glossaries/{id}/terms/{term_id}", web.WrapH(api.deleteTerm))

	mux.HandleFunc("GET /terms", web.WrapH(api.listAllTerms))
	mux.HandleFunc("POST /terms", web.WrapH(api.legacyCreateTerm))
	mux.HandleFunc("DELETE /terms/{id}", web.WrapH(api.legacyDeleteTerm))
}

// ---- Glossary ----

func (a TermAPI) listGlossaries(r *http.Request, _ *struct{}) (any, error) {
	glossaries, err := a.core.ListGlossaries(r.Context())
	if err != nil {
		return nil, reason.ErrDB.Withf("查询词库失败: %s", err)
	}
	return map[string]any{"items": glossaries, "total": len(glossaries)}, nil
}

type createGlossaryInput struct {
	Name string `json:"name"`
}

func (a TermAPI) createGlossary(r *http.Request, in *createGlossaryInput) (*term.Glossary, error) {
	g, err := a.core.CreateGlossary(r.Context(), in.Name)
	if err != nil {
		return nil, reason.ErrDB.Withf("创建词库失败: %s", err)
	}
	return g, nil
}

type updateGlossaryInput struct {
	ID       int64  `uri:"id"`
	Name     string `json:"name"`
	Enabled  *bool  `json:"enabled"`
	Priority *int   `json:"priority"`
}

func (a TermAPI) updateGlossary(r *http.Request, in *updateGlossaryInput) (any, error) {
	ctx := r.Context()
	glossaries, err := a.core.ListGlossaries(ctx)
	if err != nil {
		return nil, reason.ErrDB.Withf("查询词库失败: %s", err)
	}
	var target *term.Glossary
	for i := range glossaries {
		if glossaries[i].ID == in.ID {
			target = &glossaries[i]
			break
		}
	}
	if target == nil {
		return nil, reason.ErrNotFound.Withf("词库 %d 不存在", in.ID)
	}
	if in.Name != "" {
		target.Name = in.Name
	}
	if in.Enabled != nil {
		target.Enabled = *in.Enabled
	}
	if in.Priority != nil {
		target.Priority = *in.Priority
	}
	if err := a.core.UpdateGlossary(ctx, target); err != nil {
		return nil, reason.ErrDB.Withf("更新词库失败: %s", err)
	}
	return target, nil
}

type deleteGlossaryInput struct {
	ID int64 `uri:"id"`
}

func (a TermAPI) deleteGlossary(r *http.Request, in *deleteGlossaryInput) (any, error) {
	if err := a.core.DeleteGlossary(r.Context(), in.ID); err != nil {
		return nil, reason.ErrDB.Withf("删除词库失败: %s", err)
	}
	return map[string]any{"ok": true}, nil
}

// ---- Term ----

type listTermsInput struct {
	ID int64 `uri:"id"`
}

func (a TermAPI) listTerms(r *http.Request, in *listTermsInput) (any, error) {
	query := r.URL.Query().Get("q")
	terms, err := a.core.SearchTerms(r.Context(), in.ID, query)
	if err != nil {
		return nil, reason.ErrDB.Withf("查询词条失败: %s", err)
	}
	return map[string]any{"items": terms, "total": len(terms)}, nil
}

type createTermInput struct {
	ID          int64  `uri:"id"`
	Text        string `json:"text"`
	Translation string `json:"translation"`
	Note        string `json:"note"`
}

func (a TermAPI) createTerm(r *http.Request, in *createTermInput) (*term.Term, error) {
	t, err := a.core.Add(r.Context(), in.ID, in.Text, in.Translation, in.Note)
	if err != nil {
		return nil, reason.ErrDB.Withf("添加词条失败: %s", err)
	}
	return t, nil
}

type updateTermInput struct {
	ID          int64  `uri:"id"`
	TermID      int64  `uri:"term_id"`
	Text        string `json:"text"`
	Translation string `json:"translation"`
	Note        string `json:"note"`
}

func (a TermAPI) updateTerm(r *http.Request, in *updateTermInput) (*term.Term, error) {
	t, err := a.core.Update(r.Context(), in.TermID, in.Text, in.Translation, in.Note)
	if err != nil {
		return nil, reason.ErrDB.Withf("更新词条失败: %s", err)
	}
	return t, nil
}

type deleteTermInput struct {
	ID     int64 `uri:"id"`
	TermID int64 `uri:"term_id"`
}

func (a TermAPI) deleteTerm(r *http.Request, in *deleteTermInput) (any, error) {
	if err := a.core.Remove(r.Context(), in.TermID); err != nil {
		return nil, reason.ErrDB.Withf("删除词条失败: %s", err)
	}
	return map[string]any{"ok": true}, nil
}

// ---- Legacy /terms ----

func (a TermAPI) listAllTerms(r *http.Request, _ *struct{}) (any, error) {
	terms, err := a.core.ListAll(r.Context())
	if err != nil {
		return nil, reason.ErrDB.Withf("查询术语失败: %s", err)
	}
	return map[string]any{"items": terms, "total": len(terms)}, nil
}

type legacyCreateTermInput struct {
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

func (a TermAPI) legacyCreateTerm(r *http.Request, in *legacyCreateTermInput) (*term.Term, error) {
	ctx := r.Context()
	glossaries, err := a.core.ListGlossaries(ctx)
	if err != nil {
		return nil, reason.ErrDB.Withf("查询词库失败: %s", err)
	}
	var glossaryID int64
	if len(glossaries) > 0 {
		glossaryID = glossaries[0].ID
	} else {
		g, err2 := a.core.CreateGlossary(ctx, "默认词库")
		if err2 != nil {
			return nil, reason.ErrDB.Withf("创建默认词库失败: %s", err2)
		}
		glossaryID = g.ID
	}
	t, err := a.core.Add(ctx, glossaryID, in.Text, in.Translation, "")
	if err != nil {
		return nil, reason.ErrDB.Withf("添加术语失败: %s", err)
	}
	return t, nil
}

func (a TermAPI) legacyDeleteTerm(r *http.Request, _ *struct{}) (any, error) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, reason.ErrBadRequest.Withf("无效 ID: %s", idStr)
	}
	if err := a.core.Remove(r.Context(), id); err != nil {
		return nil, reason.ErrDB.Withf("删除术语失败: %s", err)
	}
	return map[string]any{"ok": true}, nil
}
