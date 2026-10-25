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
