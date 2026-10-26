// dcd.go —— sfp/dcd 双币赢持仓读面（T-0101）：GET /finance/sfp/dcd/order-history
// ?state=live 枚举在持产品单——已部署资本=基金的钱第三类别（非 USDT、非腿基币、
// 非托管尘）。端点名钉死文档（猜名=500 非 404 坑在案）；信封 code 数字型已由
// client.do 双型兼容承接。
package okxexec

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DcdOrder dcd 产品持仓单（order-history 行——字段名=venue 返文如实，
// 2026-10-25 真户实拉取证：productId/annualizedYield 名非猜）。
type DcdOrder struct {
	OrdID           string `json:"ordId"`
	ProductID       string `json:"productId"` // BTC-USDT-261225-95000-C 型
	State           string `json:"state"`     // LIVE/SETTLED 族
	BaseCcy         string `json:"baseCcy"`
	QuoteCcy        string `json:"quoteCcy"`
	NotionalCcy     string `json:"notionalCcy"`
	NotionalSz      string `json:"notionalSz"`
	Strike          string `json:"strike"`          // 行权价
	AnnualizedYield string `json:"annualizedYield"` // 年化
	SettleTime      string `json:"settleTime"`      // 到期（ms 串）
	CTime           string `json:"cTime"`
}

// DcdLiveOrders 在持 dcd 单枚举（state=live——SETTLED 单走余额径不入此面；
// 读败=err 上抛不假零）。60s TTL 单飞缓存——端点限流紧（429 实证），
// 卡面/对拍/巡检多方裸打曾撞限；持仓变动粒度=购买/结算日级，窗内陈旧零风险。
func (c *Client) DcdLiveOrders(ctx context.Context) ([]DcdOrder, error) {
	c.dcdMu.Lock()
	defer c.dcdMu.Unlock()
	if time.Since(c.dcdAt) < 60*time.Second {
		return c.dcdOrds, c.dcdErr
	}
	raw, err := c.Get(ctx, "/api/v5/finance/sfp/dcd/order-history?state=live")
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

// DcdQuote 产品报价（GET quote?productId&sz——陈旧提案重 quote 不盲执）。
func (c *Client) DcdQuote(ctx context.Context, productID, sz string) (json.RawMessage, error) {
	return c.Get(ctx, "/api/v5/finance/sfp/dcd/quote?productId="+productID+"&sz="+sz)
}

// DcdTrade 产品成交（POST trade {productId,sz,clOrdId}——幂等键方言同
// transfer clientId：正文参数名 clOrdId 保持；歧态势禁补单纪律同规）。
func (c *Client) DcdTrade(ctx context.Context, productID, sz, clOrdID string) (json.RawMessage, error) {
	body, _ := json.Marshal(map[string]string{
		"productId": productID, "sz": sz, "clOrdId": clOrdID})
	return c.Post(ctx, "/api/v5/finance/sfp/dcd/trade", body)
}

// DcdOrderStatus 单查证（GET order-status?ordId——歧态势定真态入口，
// 禁盲重发）。
func (c *Client) DcdOrderStatus(ctx context.Context, ordID string) (json.RawMessage, error) {
	return c.Get(ctx, "/api/v5/finance/sfp/dcd/order-status?ordId="+ordID)
}
