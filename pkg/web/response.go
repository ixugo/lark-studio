package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"unsafe"

	"github.com/ixugo/goddd/pkg/reason"
)

const ResponseErr = "responseErr"

var defaultDebug = true

// IsRelease 是否是生产环境
func IsRelease() bool {
	return !defaultDebug
}

// SetRelease 设置为生产环境，接口不再输出 details 信息。
func SetRelease() {
	defaultDebug = false
}

// SetDebug 设置为开发环境，接口输出 details 信息。
func SetDebug() {
	defaultDebug = true
}

// writeJSON 写入 JSON 响应。
func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError 通用错误响应，解析 reason.Error 结构。
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	out := make(map[string]any)
	if traceID, ok := TraceID(r.Context()); ok {
		out["trace_id"] = traceID
	}

	code := 400
	if e1, ok := err.(reason.ErrorInfoer); ok {
		code = e1.GetHTTPCode()
		out["reason"] = e1.GetReason()
		out["msg"] = e1.GetMessage()
		if defaultDebug {
			d := e1.GetDetails()
			if len(d) > 0 {
				out["details"] = d
			}
		}
	}
	writeJSON(w, code, out)
}

// WrapH 包装业务处理函数，自动绑定参数、返回 JSON 响应。
// 没有入参时应使用 *struct{}。
func WrapH[I any, O any](fn func(r *http.Request, in *I) (O, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in I
		if unsafe.Sizeof(in) > 0 {
			if err := bindURI(r, &in); err != nil {
				WriteError(w, r, reason.ErrBadRequest.With(HanddleJSONErr(err).Error()))
				return
			}
			switch r.Method {
			case http.MethodGet:
				if err := bindQuery(r, &in); err != nil {
					WriteError(w, r, reason.ErrBadRequest.With(HanddleJSONErr(err).Error()))
					return
				}
			case http.MethodDelete:
				if r.ContentLength > 0 {
					contentType := r.Header.Get("Content-Type")
					if contentType == "" {
						WriteError(w, r, reason.ErrBadRequest.With("Content-Type 不能为空"))
						return
					}
					if err := bindJSON(r, &in); err != nil {
						WriteError(w, r, reason.ErrBadRequest.With(HanddleJSONErr(err).Error()))
						return
					}
				} else {
					if err := bindQuery(r, &in); err != nil {
						WriteError(w, r, reason.ErrBadRequest.With(HanddleJSONErr(err).Error()))
						return
					}
				}
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				if r.ContentLength > 0 {
					contentType := r.Header.Get("Content-Type")
					if contentType == "" {
						WriteError(w, r, reason.ErrBadRequest.With("Content-Type 不能为空"))
						return
					}
					if err := bindJSON(r, &in); err != nil {
						WriteError(w, r, reason.ErrBadRequest.With(HanddleJSONErr(err).Error()))
						return
					}
				}
			}
		}
		out, err := fn(r, &in)
		if err != nil {
			WriteError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

type ResponseMsg struct {
	Msg string `json:"msg"`
}

// HandlerResponseMsg 获取响应的结果
func HandlerResponseMsg(resp http.Response) error {
	if resp.StatusCode == 200 {
		return nil
	}
	var out ResponseMsg
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return reason.ErrServer.SetMsg(out.Msg)
	}
	return reason.ErrServer.SetMsg(resp.Status)
}

// HanddleJSONErr 将 JSON 解析错误转为用户可读信息。
func HanddleJSONErr(err error) error {
	if err == nil {
		return nil
	}

	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var invalidUnmarshalError *json.InvalidUnmarshalError

	switch {
	case errors.As(err, &syntaxError):
		return fmt.Errorf("格式错误 (位于 %d)", syntaxError.Offset)
	case errors.Is(err, io.ErrUnexpectedEOF):
		return fmt.Errorf("格式错误")
	case errors.As(err, &unmarshalTypeError):
		if unmarshalTypeError.Field != "" {
			return fmt.Errorf("正文包含不正确的格式类型 %q", unmarshalTypeError.Field)
		}
		return fmt.Errorf("正文包含不正确的格式类型 (位于 %d)", unmarshalTypeError.Offset)
	case errors.Is(err, io.EOF):
		return errors.New("正文不能为空")
	case errors.As(err, &invalidUnmarshalError):
		panic(err)
	default:
		return err
	}
}
