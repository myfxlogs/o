// dcd.go —— sfp/dcd 双币赢持仓读面（T-0101）：GET /finance/sfp/dcd/order-history
// ?state=live 枚举在持产品单——已部署资本=基金的钱第三类别（非 USDT、非腿基币、
// 非托管尘）。端点名钉死文档（猜名=500 非 404 坑在案）；信封 code 数字型已由
// client.do 双型兼容承接。
package okxexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DcdOrder dcd 产品持仓单（order-history 行——字段名=venue 返文如实，
// 2026-10-25 真户实拉取证：productId/annualizedYield 名非猜；
// 结算落果 settleSz/settleCcy/settlePx/yieldSz/yieldCcy=docs-v5 返参核证）。
type DcdOrder struct {
	OrdID           string `json:"ordId"`
	ProductID       string `json:"productId"` // BTC-USDT-261225-95000-C 型
	State           string `json:"state"`     // live/pending_settle/settled/pending_redeem/redeemed 族
	BaseCcy         string `json:"baseCcy"`
	QuoteCcy        string `json:"quoteCcy"`
	NotionalCcy     string `json:"notionalCcy"`
	NotionalSz      string `json:"notionalSz"`
	Strike          string `json:"strike"`          // 行权价
	AnnualizedYield string `json:"annualizedYield"` // 年化
	YieldSz         string `json:"yieldSz"`         // 收益金额
	YieldCcy        string `json:"yieldCcy"`        // 收益币种
	SettleSz        string `json:"settleSz"`        // 结算金额（未结算=""）
	SettleCcy       string `json:"settleCcy"`       // 结算币种（未结算=""——回款币种 venue 明告）
	SettlePx        string `json:"settlePx"`        // 结算价格（未结算=""）
	SettleTime      string `json:"settleTime"`      // 到期（ms 串）
	SettledTime     string `json:"settledTime"`     // 实结时刻（ms 串——未结算=""；现金腿 occurred_at 优先锚）
	ExpTime         string `json:"expTime"`         // 产品到期（ms 串）
	CTime           string `json:"cTime"`
}

// DcdLiveOrders 在持 dcd 单枚举（state=live——SETTLED 单走余额径不入此面；
// 读败=err 上抛不假零）。60s TTL 单飞缓存——端点限流紧（429 实证），
// 卡面/对拍/巡检多方裸打曾撞限；持仓变动粒度=购买/结算日级，窗内陈旧零风险。
// 工单#22919：拉取改无过滤全谱（limit=100——同调用数下兼得结算/在途集，
// 端点限速 1次/s 禁第二发），live 分区客户端侧。
func (c *Client) DcdLiveOrders(ctx context.Context) ([]DcdOrder, error) {
	ords, err := c.DcdOrders(ctx)
	if err != nil {
		return ords, err
	}
	var live []DcdOrder
	for _, od := range ords {
		if strings.EqualFold(od.State, "live") {
			live = append(live, od)
		}
	}
	return live, nil
}

// DcdOrders 全谱 dcd 单枚举（无 state 过滤——live/pending_settle/settled
// 同帧返回；缓存共享 DcdLiveOrders 同槽）。
func (c *Client) DcdOrders(ctx context.Context) ([]DcdOrder, error) {
	c.dcdMu.Lock()
	defer c.dcdMu.Unlock()
	if time.Since(c.dcdAt) < 60*time.Second {
		return c.dcdOrds, c.dcdErr
	}
	raw, err := c.Get(ctx, "/api/v5/finance/sfp/dcd/order-history?limit=100")
	c.dcdAt = time.Now()
	c.dcdErr = err
	if err != nil {
		return c.dcdOrds, err
	}
	var rows []DcdOrder
	if err := json.Unmarshal(raw, &rows); err != nil {
		c.dcdErr = fmt.Errorf("dcd order-history parse: %w", err)
		return c.dcdOrds, c.dcdErr
	}
	c.dcdOrds, c.dcdErr = rows, nil
	return rows, nil
}

// —— 写面（T-0160 DCD 执行席：quote→trade→order-status 链；信封 code
// 数字型已由 flexCode 承接——okx-dcd-endpoint-dialect 条目径）。

// DcdQuote 产品报价（T-0207 真径：POST 体三参——GET+sz 参数名是 405
// 方言坑；notionalCcy=产品锁定币 raw 回填勿猜）。返回 quoteId——
// trade 径只认报价绑单。
func (c *Client) DcdQuote(ctx context.Context, productID, notionalSz, notionalCcy string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"productId": productID, "notionalSz": notionalSz, "notionalCcy": notionalCcy})
	raw, err := c.Post(ctx, "/api/v5/finance/sfp/dcd/quote", body)
	if err != nil {
		return "", err
	}
	var rows []struct {
		QuoteID string `json:"quoteId"`
		SCode   string `json:"sCode"`
		SMsg    string `json:"sMsg"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return "", fmt.Errorf("dcd quote 响应解析: %w", err)
	}
	if len(rows) > 0 && rows[0].QuoteID != "" {
		return rows[0].QuoteID, nil
	}
	// T-0259：code=0 壳内 per-item sCode≠0=venue 业务拒——拒因透 APIError 上抛
	// （envelope 层拒 do() 已产 APIError；此层补内层码位——持久化径取得到原始码）。
	for _, r := range rows {
		if r.SCode != "" && r.SCode != "0" {
			return "", &APIError{SCode: r.SCode, SMsg: r.SMsg}
		}
	}
	return "", fmt.Errorf("dcd quote 响应无 quoteId（rows=%d）", len(rows))
}

// DcdTrade 产品成交（T-0207 真径：POST trade 体唯 quoteId——报价绑单；
// venue 无 clOrdId 幂等键——歧态势（超时/5xx 成否未知）禁补单，
// 先 order-status?ordId 或 order-history 按 productId+ts 反查真态再定）。
func (c *Client) DcdTrade(ctx context.Context, quoteID string) (json.RawMessage, error) {
	body, _ := json.Marshal(map[string]string{"quoteId": quoteID})
	return c.Post(ctx, "/api/v5/finance/sfp/dcd/trade", body)
}

// DcdOrderStatus 单查证（GET order-status?ordId——歧态势定真态入口，
// 禁盲重发）。
func (c *Client) DcdOrderStatus(ctx context.Context, ordID string) (json.RawMessage, error) {
	return c.Get(ctx, "/api/v5/finance/sfp/dcd/order-status?ordId="+ordID)
}
