// Package notify —— 飞书推送客户端（NOX internal/ir/notify/feishu.go 考古继承）：
// tenant_access_token 缓存（2h 效期提前 5min 续约不每拍取）+ im/v1/messages
// SendText 发文本。超时 10s；发送失败上抛由调用方计数——禁重试风暴。
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const feishuBase = "https://open.feishu.cn/open-apis"

// FeishuClient 自定义应用机器人客户端（app_id/app_secret 模式）。
type FeishuClient struct {
	appID, appSecret string
	hc               *http.Client
	mu               sync.Mutex
	token            string
	tokenExp         time.Time
}

func NewFeishu(appID, appSecret string) *FeishuClient {
	return &FeishuClient{appID: appID, appSecret: appSecret,
		hc: &http.Client{Timeout: 10 * time.Second}}
}

type feishuTokenResp struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	Expire            int    `json:"expire"`
}

func (c *FeishuClient) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}
	body, _ := json.Marshal(map[string]string{"app_id": c.appID, "app_secret": c.appSecret})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		feishuBase+"/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("feishu token: %w", err)
	}
	defer resp.Body.Close()
	var tr feishuTokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("feishu token decode: %w", err)
	}
	if tr.Code != 0 || tr.TenantAccessToken == "" {
		return "", fmt.Errorf("feishu token: code=%d msg=%s", tr.Code, tr.Msg)
	}
	c.token = tr.TenantAccessToken
	c.tokenExp = time.Now().Add(time.Duration(tr.Expire)*time.Second - 5*time.Minute)
	return c.token, nil
}

// SendText 向 chat_id 发文本消息（msg_type=text），返回 message_id。
func (c *FeishuClient) SendText(ctx context.Context, chatID, text string) (string, error) {
	if chatID == "" {
		return "", fmt.Errorf("feishu send: empty chat_id（未绑定，先给机器人发条消息）")
	}
	tok, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}
	content, _ := json.Marshal(map[string]string{"text": text})
	body, _ := json.Marshal(map[string]string{
		"receive_id": chatID,
		"msg_type":   "text",
		"content":    string(content),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		feishuBase+"/im/v1/messages?receive_id_type=chat_id", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("feishu send: %w", err)
	}
	defer resp.Body.Close()
	var sr struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return "", fmt.Errorf("feishu send decode: %w", err)
	}
	if sr.Code != 0 {
		return "", fmt.Errorf("feishu send: code=%d msg=%s", sr.Code, sr.Msg)
	}
	return sr.Data.MessageID, nil
}

// ListChats 机器人在群清单（chat_id 自动发现径——业主免手工查 id）。
func (c *FeishuClient) ListChats(ctx context.Context) ([]struct{ ID, Name string }, error) {
	tok, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feishuBase+"/im/v1/chats", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("feishu chats: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []struct {
				ChatID string `json:"chat_id"`
				Name   string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Code != 0 {
		return nil, fmt.Errorf("feishu chats: code=%d msg=%s", out.Code, out.Msg)
	}
	res := make([]struct{ ID, Name string }, len(out.Data.Items))
	for i, it := range out.Data.Items {
		res[i].ID, res[i].Name = it.ChatID, it.Name
	}
	return res, nil
}
