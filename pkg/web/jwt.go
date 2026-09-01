package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/ixugo/goddd/pkg/reason"
)

// JWT context key constants
const (
	KeyUserID      = "uid"
	KeyLevel       = "level"
	KeyRoleID      = "role_id"
	KeyUsername    = "username"
	KeyTokenString = "token"
)

// Claims JWT 声明
type Claims struct {
	Data map[string]any
	jwt.RegisteredClaims
}

type ClaimsData map[string]any

type TokenOptions func(*Claims)

func NewClaimsData() ClaimsData {
	return make(ClaimsData)
}

func (c ClaimsData) SetUserID(uid int) ClaimsData {
	c[KeyUserID] = uid
	return c
}

func (c ClaimsData) SetLevel(level int) ClaimsData {
	c[KeyLevel] = level
	return c
}

func (c ClaimsData) SetRoleID(roleID int) ClaimsData {
	c[KeyRoleID] = roleID
	return c
}

func (c ClaimsData) SetUsername(username string) ClaimsData {
	c[KeyUsername] = username
	return c
}

func (c ClaimsData) Set(key string, value any) ClaimsData {
	c[key] = value
	return c
}

// claimsCtxKey 用于在 context 中传递 JWT 声明。
const claimsCtxKey ctxKey = "jwt_claims"

// AuthMiddleware 鉴权中间件。
func AuthMiddleware(secret string, skip ...func(*http.Request) bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, fn := range skip {
				if fn(r) {
					next.ServeHTTP(w, r)
					return
				}
			}

			auth := r.Header.Get("Authorization")
			if auth == "" {
				auth = r.URL.Query().Get("token")
			}
			const prefix = "Bearer "
			if len(auth) <= len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
				WriteError(w, r, reason.ErrUnauthorizedToken.SetMsg("身份验证失败"))
				return
			}
			claims, err := ParseToken(auth[len(prefix):], secret)
			if err != nil {
				WriteError(w, r, reason.ErrUnauthorizedToken.SetMsg("身份验证失败"))
				return
			}
			if err := claims.Valid(); err != nil {
				WriteError(w, r, reason.ErrUnauthorizedToken.SetMsg("请重新登录"))
				return
			}

			ctx := r.Context()
			ctx = WithTraceID(ctx, auth)
			for k, v := range claims.Data {
				ctx = setCtxValue(ctx, ctxKey(k), v)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUID 获取用户 ID
func GetUID(ctx interface{ Value(any) any }) int {
	return getCtxInt(ctx, ctxKey(KeyUserID))
}

// GetUsername 获取用户名
func GetUsername(ctx interface{ Value(any) any }) string {
	v := ctx.Value(ctxKey(KeyUsername))
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetRoleID 获取用户角色
func GetRoleID(ctx interface{ Value(any) any }) int {
	return getCtxInt(ctx, ctxKey(KeyRoleID))
}

// GetLevel 获取用户等级
func GetLevel(ctx interface{ Value(any) any }) int {
	return getCtxInt(ctx, ctxKey(KeyLevel))
}

func getCtxInt(ctx interface{ Value(any) any }, key ctxKey) int {
	v := ctx.Value(key)
	if v == nil {
		return 0
	}
	switch v := v.(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// ParseToken 解析 token
func ParseToken(tokenString string, secret string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(tokenString, &claims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithoutClaimsValidation())
	return &claims, err
}

// WithExpiresAt 设置指定过期时间
func WithExpiresAt(expiresAt time.Time) TokenOptions {
	return func(c *Claims) { c.ExpiresAt = jwt.NewNumericDate(expiresAt) }
}

// WithExpires 设置多久过期
func WithExpires(duration time.Duration) TokenOptions {
	return func(c *Claims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(duration)) }
}

// WithIssuedAt 设置签发时间
func WithIssuedAt(issuedAt time.Time) TokenOptions {
	return func(c *Claims) { c.IssuedAt = jwt.NewNumericDate(issuedAt) }
}

// WithIssuer 设置签发人
func WithIssuer(issuer string) TokenOptions {
	return func(c *Claims) { c.Issuer = issuer }
}

// WithNotBefore 设置生效时间
func WithNotBefore(notBefore time.Time) TokenOptions {
	return func(c *Claims) { c.NotBefore = jwt.NewNumericDate(notBefore) }
}

// NewToken 创建 token，默认 6 小时过期。
func NewToken(data map[string]any, secret string, opts ...TokenOptions) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("secret is required")
	}
	now := time.Now()
	claims := Claims{
		Data: data,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(6 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "goddd.golang.space",
		},
	}
	for _, opt := range opts {
		opt(&claims)
	}
	tc := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tc.SignedString([]byte(secret))
}
