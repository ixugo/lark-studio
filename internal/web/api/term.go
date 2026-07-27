package api

import (
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
	group := g.Group("/terms", handler...)
	group.GET("", web.WrapH(api.listTerms))
	group.POST("", web.WrapH(api.createTerm))
	group.DELETE("/:id", web.WrapH(api.deleteTerm))
}

func (a TermAPI) listTerms(c *gin.Context, _ *struct{}) (any, error) {
	terms, err := a.core.ListAll(c.Request.Context())
	if err != nil {
		return nil, reason.ErrDB.Withf("查询术语失败: %s", err)
	}
	return gin.H{"items": terms, "total": len(terms)}, nil
}

type createTermInput struct {
	Text        string `json:"text" binding:"required,max=100"`
	Translation string `json:"translation"`
}

func (a TermAPI) createTerm(c *gin.Context, in *createTermInput) (*term.Term, error) {
	t, err := a.core.Add(c.Request.Context(), in.Text, in.Translation)
	if err != nil {
		return nil, reason.ErrDB.Withf("添加术语失败: %s", err)
	}
	return t, nil
}

type deleteTermInput struct {
	ID int64 `uri:"id" binding:"required"`
}

func (a TermAPI) deleteTerm(c *gin.Context, in *deleteTermInput) (any, error) {
	if err := a.core.Remove(c.Request.Context(), in.ID); err != nil {
		return nil, reason.ErrDB.Withf("删除术语失败: %s", err)
	}
	return gin.H{"ok": true}, nil
}
