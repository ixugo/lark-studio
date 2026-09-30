package youtube

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	ServiceOrigin       = "https://embed.dlsrv.online"
	maxJSONBytes        = 1 << 20
	maxVideoBytes int64 = 20 << 30
)

var workerHost = regexp.MustCompile(`^yt1s-worker-[0-9]+\.dlsrv\.online$`)

type Session struct {
	Token     string
	UserAgent string
	Expires   time.Time
}

// 这里只读取有效期作本地判断，令牌签名由第三方接口校验。
func ParseSession(token, agent string) (Session, error) {
	if len(token) > 8192 || len(agent) > 1024 || strings.ContainsAny(agent, "\r\n") || agent == "" {
		return Session{}, errors.New("校验会话格式无效")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Session{}, errors.New("校验会话格式无效")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Session{}, errors.New("校验会话格式无效")
	}
	var claims struct {
		Expires int64 `json:"exp"`
	}
	if err = json.Unmarshal(data, &claims); err != nil {
		return Session{}, errors.New("校验会话格式无效")
	}
	expires := time.Unix(claims.Expires, 0)
	if time.Until(expires) < 10*time.Second || time.Until(expires) > 10*time.Minute {
		return Session{}, errors.New("校验会话已失效，请重新校验")
	}
	return Session{Token: token, UserAgent: agent, Expires: expires}, nil
}
func validateTunnel(raw string) error {
	if len(raw) > 8192 {
		return errors.New("下载地址过长")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || !workerHost.MatchString(u.Host) || u.Path != "/tunnel" {
		return errors.New("下载服务返回了不受信任的地址")
	}
	return nil
}
func videoIDFromURL(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return u.Query().Get("v")
}
func requestHeaders(req *http.Request, session Session) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", session.UserAgent)
	req.Header.Set("Origin", ServiceOrigin)
	req.Header.Set("Referer", ServiceOrigin+"/v1/full")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Authorization", "Bearer "+session.Token)
}
func readJSON(resp *http.Response, out any) error {
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxJSONBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxJSONBytes {
		return errors.New("服务响应过大")
	}
	if resp.StatusCode != http.StatusOK {
		if slices.Contains([]int{401, 403}, resp.StatusCode) {
			return errors.New("校验会话无效，请重新校验")
		}
		return fmt.Errorf("下载服务暂不可用（HTTP %d）", resp.StatusCode)
	}
	if err = json.Unmarshal(data, out); err != nil {
		return errors.New("下载服务响应格式无效")
	}
	return nil
}
func (m *Manager) convert(ctx context.Context, link string, height int, session Session) (string, error) {
	payload, err := json.Marshal(struct {
		VideoID string `json:"videoId"`
		Format  string `json:"format"`
		Quality string `json:"quality"`
	}{videoIDFromURL(link), "mp4", fmt.Sprint(height)})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint+"/api/download/mp4", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	requestHeaders(req, session)
	resp, err := m.http.Do(req)
	if err != nil {
		return "", errors.New("无法连接下载服务，请检查网络")
	}
	var result struct {
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	if err = readJSON(resp, &result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("下载服务转换失败：%.500s", result.Error)
	}
	if err = validateTunnel(result.URL); err != nil {
		return "", err
	}
	return result.URL, nil
}

func ValidateDownloadAddress(address string) error { return validateTunnel(address) }
