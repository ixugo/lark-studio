package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// bindQuery 从 URL query 参数绑定到 dst，使用 "form" tag。
func bindQuery(r *http.Request, dst any) error {
	return bindURLValues(r.URL.Query(), dst, "form")
}

// bindURI 从路径参数绑定到 dst，使用 "uri" tag。
// 依赖 Go 1.22+ http.Request.PathValue。
func bindURI(r *http.Request, dst any) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return nil
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}
	return setURIFields(r, v.Type(), v)
}

// bindJSON 从请求体解码 JSON 到 dst。
func bindJSON(r *http.Request, dst any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}
	return json.NewDecoder(r.Body).Decode(dst)
}

func setURIFields(r *http.Request, t reflect.Type, v reflect.Value) error {
	for i := range t.NumField() {
		field := t.Field(i)
		fv := v.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := setURIFields(r, field.Type, fv); err != nil {
				return err
			}
			continue
		}
		tag := field.Tag.Get("uri")
		if tag == "" || tag == "-" {
			continue
		}
		val := r.PathValue(tag)
		if val == "" {
			continue
		}
		if err := setFieldValue(fv, val); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}
	return nil
}

func bindURLValues(vals url.Values, dst any, tagName string) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return nil
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}
	return setFormFields(vals, v.Type(), v, tagName)
}

func setFormFields(vals url.Values, t reflect.Type, v reflect.Value, tagName string) error {
	for i := range t.NumField() {
		field := t.Field(i)
		fv := v.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := setFormFields(vals, field.Type, fv, tagName); err != nil {
				return err
			}
			continue
		}
		raw := field.Tag.Get(tagName)
		if raw == "" || raw == "-" {
			continue
		}
		name := strings.SplitN(raw, ",", 2)[0]
		val := vals.Get(name)
		if val == "" {
			continue
		}
		if err := setFieldValue(fv, val); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}
	return nil
}

var (
	timeType    = reflect.TypeOf(time.Time{})
	timePtrType = reflect.TypeOf((*time.Time)(nil))
)

func setFieldValue(fv reflect.Value, val string) error {
	if !fv.CanSet() {
		return nil
	}
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(val)
	case reflect.Int, reflect.Int64:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Int32:
		n, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Float64:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Ptr:
		if fv.Type() == timePtrType {
			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				return err
			}
			fv.Set(reflect.ValueOf(&t))
		}
	default:
		if fv.Type() == timeType {
			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				return err
			}
			fv.Set(reflect.ValueOf(t))
		}
	}
	return nil
}
