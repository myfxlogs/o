package main

// restlive.go —— REST 保留面活性戳+funding/savings 现读聚（自 venue_readers.go
// 拆出——棘轮 300 行顶线）。

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"phase/internal/trading/okxexec"
)

// restOKAt REST 保留面末次成功读戳（T-0088 真源钉：earn_parking 帧稀疏
// 3min~5h 一跳当活性=假红灯——对拍环每拍实调 funding/savings，成功戳才真）。
var restOKAt atomic.Int64

// fundSaveBalances funding+savings 现读聚（T-0058：任一读败=err 上抛——静默跳腿禁）。
func fundSaveBalances(ctx context.Context, okcli *okxexec.Client, out map[[2]string]float64) error {
	bals, err := okcli.AssetBalances(ctx)
	if err != nil {
		return fmt.Errorf("funding 读败: %w", err)
	}
	for _, b := range bals {
		out[[2]string{"okx", b.Ccy}] += okxexec.ParseF(b.Bal)
	}
	sav, err := okcli.SavingsBalance(ctx)
	if err != nil {
		return fmt.Errorf("savings 读败: %w", err)
	}
	var rows []struct {
		Ccy string `json:"ccy"`
		Amt string `json:"amt"`
	}
	if json.Unmarshal(sav, &rows) != nil {
		return fmt.Errorf("savings parse")
	}
	for _, r2 := range rows {
		out[[2]string{"okx", r2.Ccy}] += okxexec.ParseF(r2.Amt)
	}
	restOKAt.Store(time.Now().UnixNano())
	return nil
}
