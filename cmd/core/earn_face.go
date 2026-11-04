// earn_face.go —— Earn 自动申购停用开关面（T-0342——业主令：前端临时停用
// O 自动转入 Simple Earn，不赎回现有持仓；开仓赎回 RedeemToTrading 不经此闸）。
// 写面=全仓唯一前端窄写径：
//
//	· 只许写 earn.auto_purchase_enabled 一键（config.Client.Set actor=
//	  dashboard-ui——updated_by/updated_at 审计列留痕；禁通用 config 写面）；
//	· Origin 同源闸：浏览器 Origin scheme+host 与请求 Host 同源才放行——
//	  缺席/跨源=CodePermissionDenied（connect-web POST 浏览器恒带 Origin；
//	  Host 经 connecthttp 注入请求头面可读）；
//	· reason 落 ops_event 审计行（seat=dashboard-ui——pipeline 只收 ticket
//	  非空行，工单板面零污染）；读面 note=末次翻位理由同表回取。
//
// 真值读径=app_config 直查（cfg 快照只载 value——updated_at/updated_by
// 审计列须库面）；缺键/坏值=enabled=false fail-closed 显形（worker 同口径）。
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"strings"
	"time"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"

	"o/internal/gen/dashboardv1"
)

const (
	earnAutoPurchaseKey  = "earn.auto_purchase_enabled" // T-0342 钦定键——写面只许此一键
	earnToggleEvent      = "earn_auto_purchase"         // ops_event 翻位审计事件名
	earnToggleActor      = "dashboard-ui"               // app_config.updated_by 写者名（钦定）
	errOriginDenied      = "同源写面拒——Origin 缺席/跨源/不可解析"
	errCfgAbsent         = "config 装配缺席——写径不可用"
	errDeniedMissingInfo = "调用面非 HTTP——Origin 不可证"
)

// earnOriginGate 同源写闸（T-0342 首版同源页面窄写钦定）：Origin scheme+host
// 与请求 Host 同源才放行；Origin 缺席=拒（浏览器同源 POST 恒带——缺席者非
// 页面径）。闸门齿=host 同源；scheme 钉 http/https 双形+TLS 直连面必 https
// （http 伺服面上游 TLS 终结不可知——host 等值已是齿，scheme 严判会误杀
// 终结代理径）。
func earnOriginGate(ctx context.Context) error {
	info, ok := connect.CallInfoForServerContext(ctx)
	if !ok {
		return connect.NewError(connect.CodePermissionDenied, errDeniedMissingInfo)
	}
	h := info.RequestHeader()
	u, err := url.Parse(h.Get("Origin"))
	if h.Get("Origin") == "" || err != nil || u.Host == "" || u.Scheme == "" {
		return connect.NewError(connect.CodePermissionDenied, errOriginDenied)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return connect.NewError(connect.CodePermissionDenied, errOriginDenied)
	}
	if host := h.Get("Host"); host == "" || !strings.EqualFold(u.Host, host) {
		return connect.NewError(connect.CodePermissionDenied, errOriginDenied)
	}
	if si, ok := connecthttp.ServerInfoForContext(ctx); ok && si.TLS() != nil && u.Scheme != "https" {
		return connect.NewError(connect.CodePermissionDenied, errOriginDenied)
	}
	return nil
}

// earnBool 库面值→bool（标量契约宽容读器同律：jsonb bool 原生或 "true"/
// "false" 字符串两形；其余形=fail-closed false）。
func earnBool(raw json.RawMessage) bool {
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s == "true"
	}
	return false
}

// earnAutoPurchaseRow 库面真值+审计列（缺行=零值——enabled=false fail-closed）。
func (c coreService) earnAutoPurchaseRow(ctx context.Context) (*dashboardv1.GetEarnAutoPurchaseResponse, error) {
	r := &dashboardv1.GetEarnAutoPurchaseResponse{}
	var raw json.RawMessage
	var at time.Time
	var by string
	err := c.pool.QueryRow(ctx, `SELECT value,updated_at,updated_by FROM app_config
		WHERE key=$1`, earnAutoPurchaseKey).Scan(&raw, &at, &by)
	switch {
	case err == nil:
		r.Enabled = earnBool(raw)
		r.UpdatedAt = tsProto(at)
		r.UpdatedBy = by
	default:
		var n int
		if cerr := c.pool.QueryRow(ctx, `SELECT count(*) FROM app_config WHERE key=$1`,
			earnAutoPurchaseKey).Scan(&n); cerr == nil && n == 0 {
			break // 键无行=fail-closed 停用显形（enabled=false 零值带出）
		}
		return nil, err
	}
	_ = c.pool.QueryRow(ctx, `SELECT note FROM ops_event WHERE event=$1
		ORDER BY id DESC LIMIT 1`, earnToggleEvent).Scan(&r.Note) // 无行=空（注缺席不炸读面）
	return r, nil
}

// GetEarnAutoPurchase 开关真值读面（前端开关态供数——缺键=停用显形不假开）。
func (c coreService) GetEarnAutoPurchase(ctx context.Context,
	_ *dashboardv1.GetEarnAutoPurchaseRequest) (*dashboardv1.GetEarnAutoPurchaseResponse, error) {
	return c.earnAutoPurchaseRow(ctx)
}

// SetEarnAutoPurchase 窄写面：Origin 同源闸→config.Client.Set 一键→ops_event
// 审计行（reason 留痕）→库面重读回真值（非回显——写后真态为准）。
func (c coreService) SetEarnAutoPurchase(ctx context.Context,
	req *dashboardv1.SetEarnAutoPurchaseRequest) (*dashboardv1.SetEarnAutoPurchaseResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	if c.cfg == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errCfgAbsent)
	}
	if err := c.cfg.Set(ctx, earnAutoPurchaseKey,
		json.RawMessage(map[bool]string{true: "true", false: "false"}[req.GetEnabled()]),
		earnToggleActor); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "app_config 写败: "+err.Error())
	}
	note := map[bool]string{true: "自动转入恢复", false: "自动转入停用"}[req.GetEnabled()]
	if r := strings.TrimSpace(req.GetReason()); r != "" {
		note += "——" + r
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,$2,'',$3)`, earnToggleActor, earnToggleEvent, note); err != nil {
		log.Printf("[earn-face] ops_event 审计行写败（键已翻位——审计痕缺）: %v", err)
	}
	row, err := c.earnAutoPurchaseRow(ctx)
	if err != nil {
		return nil, err
	}
	return &dashboardv1.SetEarnAutoPurchaseResponse{
		Enabled: row.Enabled, UpdatedAt: row.UpdatedAt, UpdatedBy: row.UpdatedBy}, nil
}
