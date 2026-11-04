// positions.go —— /account/positions 账户持仓全枚举读面（T-0327 残仓类级
// 枚举源：合约/期权/杠杆在持行——余额面外 venue 在持第二族；读败=err 上抛
// 不假零）。无 instId 参=全量枚举（ContractPosSz 是按 inst 定点读，勿混）。
package okxexec

import (
	"context"
	"encoding/json"
	"fmt"
)

// Position 账户持仓行（字段名=venue 返文如实；pos signed——net 模式负=空）。
type Position struct {
	InstID      string `json:"instId"`
	InstType    string `json:"instType"` // SWAP/FUTURES/OPTION/MARGIN
	Pos         string `json:"pos"`
	PosSide     string `json:"posSide"`     // long/short/net
	NotionalUsd string `json:"notionalUsd"` // USD 名义
	CTime       string `json:"cTime"`       // 开仓 ms 串
}

// Positions 全量在持枚举（GET /account/positions 无参—— venue 在持真值源；
// 读败=err 上抛不假零）。
func (c *Client) Positions(ctx context.Context) ([]Position, error) {
	raw, err := c.Get(ctx, "/api/v5/account/positions")
	if err != nil {
		return nil, err
	}
	var rows []Position
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("positions parse: %w", err)
	}
	return rows, nil
}
