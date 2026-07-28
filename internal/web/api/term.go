package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ixugo/goddd/pkg/reason"
	"github.com/ixugo/goddd/pkg/web"
	"github.com/ixugo/vdub/internal/core/term"
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
func RegisterTerm(g gin.IRouter, api TermAPI, handler ...gin.HandlerFunc) {
	// 词库
	glossaryGroup := g.Group("/glossaries", handler...)
	glossaryGroup.GET("", web.WrapH(api.listGlossaries))
	glossaryGroup.POST("", web.WrapH(api.createGlossary))
	glossaryGroup.PUT("/:id", web.WrapH(api.updateGlossary))
	glossaryGroup.DELETE("/:id", web.WrapH(api.deleteGlossary))

	// 词条（嵌套在词库下）
	termGroup := g.Group("/glossaries/:glossary_id/terms", handler...)
	termGroup.GET("", web.WrapH(api.listTerms))
	termGroup.POST("", web.WrapH(api.createTerm))
	termGroup.DELETE("/:id", web.WrapH(api.deleteTerm))

	// 兼容旧 /terms 接口（返回全部启用词条）
	legacy := g.Group("/terms", handler...)
	legacy.GET("", web.WrapH(api.listAllTerms))
	legacy.POST("", web.WrapH(api.legacyCreateTerm))
	legacy.DELETE("/:id", web.WrapH(api.legacyDeleteTerm))
}

// ---- Glossary ----

func (a TermAPI) listGlossaries(c *gin.Context, _ *struct{}) (any, error) {
	glossaries, err := a.core.ListGlossaries(c.Request.Context())
	if err != nil {
		return nil, reason.ErrDB.Withf("查询词库失败: %s", err)
	}
	return gin.H{"items": glossaries, "total": len(glossaries)}, nil
}

type createGlossaryInput struct {
	Name string `json:"name" binding:"required,max=50"`
}

func (a TermAPI) createGlossary(c *gin.Context, in *createGlossaryInput) (*term.Glossary, error) {
	g, err := a.core.CreateGlossary(c.Request.Context(), in.Name)
	if err != nil {
		return nil, reason.ErrDB.Withf("创建词库失败: %s", err)
	}
	return g, nil
}

type updateGlossaryInput struct {
	ID       int64  `uri:"id" binding:"required"`
	Name     string `json:"name"`
	Enabled  *bool  `json:"enabled"`
	Priority *int   `json:"priority"`
}

func (a TermAPI) updateGlossary(c *gin.Context, in *updateGlossaryInput) (any, error) {
	ctx := c.Request.Context()
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
	ID int64 `uri:"id" binding:"required"`
}

func (a TermAPI) deleteGlossary(c *gin.Context, in *deleteGlossaryInput) (any, error) {
	if err := a.core.DeleteGlossary(c.Request.Context(), in.ID); err != nil {
		return nil, reason.ErrDB.Withf("删除词库失败: %s", err)
	}
	return gin.H{"ok": true}, nil
}

// ---- Term ----

type listTermsInput struct {
	GlossaryID int64 `uri:"glossary_id" binding:"required"`
}

func (a TermAPI) listTerms(c *gin.Context, in *listTermsInput) (any, error) {
	query := c.Query("q")
	terms, err := a.core.SearchTerms(c.Request.Context(), in.GlossaryID, query)
	if err != nil {
		return nil, reason.ErrDB.Withf("查询词条失败: %s", err)
	}
	return gin.H{"items": terms, "total": len(terms)}, nil
}

type createTermInput struct {
	GlossaryID  int64  `uri:"glossary_id" binding:"required"`
	Text        string `json:"text" binding:"required,max=100"`
	Translation string `json:"translation"`
	Note        string `json:"note"`
}

func (a TermAPI) createTerm(c *gin.Context, in *createTermInput) (*term.Term, error) {
	t, err := a.core.Add(c.Request.Context(), in.GlossaryID, in.Text, in.Translation, in.Note)
	if err != nil {
		return nil, reason.ErrDB.Withf("添加词条失败: %s", err)
	}
	return t, nil
}

type deleteTermInput struct {
	GlossaryID int64 `uri:"glossary_id"`
	ID         int64 `uri:"id" binding:"required"`
}

func (a TermAPI) deleteTerm(c *gin.Context, in *deleteTermInput) (any, error) {
	if err := a.core.Remove(c.Request.Context(), in.ID); err != nil {
		return nil, reason.ErrDB.Withf("删除词条失败: %s", err)
	}
	return gin.H{"ok": true}, nil
}

// ---- Legacy /terms ----

func (a TermAPI) listAllTerms(c *gin.Context, _ *struct{}) (any, error) {
	terms, err := a.core.ListAll(c.Request.Context())
	if err != nil {
		return nil, reason.ErrDB.Withf("查询术语失败: %s", err)
	}
	return gin.H{"items": terms, "total": len(terms)}, nil
}

type legacyCreateTermInput struct {
	Text        string `json:"text" binding:"required,max=100"`
	Translation string `json:"translation"`
}

func (a TermAPI) legacyCreateTerm(c *gin.Context, in *legacyCreateTermInput) (*term.Term, error) {
	// 兼容旧接口：查找或创建一个默认词库
	ctx := c.Request.Context()
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

func (a TermAPI) legacyDeleteTerm(c *gin.Context, _ *struct{}) (any, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, reason.ErrBadRequest.Withf("无效 ID: %s", idStr)
	}
	if err := a.core.Remove(c.Request.Context(), id); err != nil {
		return nil, reason.ErrDB.Withf("删除术语失败: %s", err)
	}
	return gin.H{"ok": true}, nil
}
