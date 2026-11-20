// launchfund.go —— launchGate 前置资金闸（工单#26337）：实证坑（bills 行
// 575/577 在案）——手动拉起把 $100 划进交易户后，earn 清扫臂 2s 内扫回
// earn 认购，拉起险败 58350。设计根因=Runner 装配未注入 earn 组件，拉起
// 前不查交易户余额不触发赎回。本件接线：拉起前查交易户 quote availBal，
// 不足则同棒内 earn.RedeemToTrading 补缺口再拉起——赎回→划转→Launch
// 同进程零 sleep（清扫臂拍距约秒级，任何间隙=送钱回 earn，实证坑锚）。
// 歧态势禁补单（okx-exec 铁律照旧）：赎回/拉起任一棒不确定态→hold 行
// 落账+升件，禁自动重试——钱径双花防第一性。
package okxgrid

import (
	"context"
	"errors"
	"fmt"
	"log"

	"o/internal/trading/okxexec"
)

// QuoteBal 交易户报价币可用余现读面（产线=*okxexec.Client.TradingAvailBal
// 方法值同签名——availBal 口径非 availEq，拉起决策用可用量）。
type QuoteBal func(ctx context.Context, ccy string) (float64, error)

// EarnRedeem 前置赎回面（产线=earn.Worker.RedeemToTrading——赎回+划转
// 一体，execution.EarnMover 同签名先例；earn 内部自带 redeem_buffer_bps
// 加成+歧态势禁补偿纪律）。
type EarnRedeem func(ctx context.Context, usdAmt float64) error

// quoteCcy 报价币=instId 第二段（grid_spot 族腿=BASE-QUOTE 现货对形——
// legset.InstFitOK 契约同源；BTC-USDT→USDT）。
func quoteCcy(instID string) string {
	for i := 0; i < len(instID); i++ {
		if instID[i] == '-' {
			return instID[i+1:]
		}
	}
	return "USDT"
}

// ensureFunds 拉起前资金闸——交易户 quote availBal 不足 quoteSz 则同棒内
// earn.RedeemToTrading 补缺口（deficit=quoteSz-avail；赎回额=deficit×
// (1+earn buffer)——余量回交易户由清扫臂归位，帽语义=单 bot 投入
// quoteSz≤max_invest_usd 钉死非资金总量）。返回 err=赎回败——任一败
// （定态/歧态不可外部判定：赎回三跳钱动局部态 rich）→hold 禁自动重试
// （票钦歧态势禁补单首义；测试面「赎回败/歧态势：不拉起+升件行，零重试」
// 同组钦定）。余额读败=非钱动下拍重试安全（调用方 log+continue）。
func (r *Runner) ensureFunds(ctx context.Context, ccy string, quoteSz float64) error {
	avail, err := r.quoteBal(ctx, ccy)
	if err != nil {
		return errBalanceRead{err}
	}
	if avail >= quoteSz {
		return nil // 资金足——零赎回零调用（测试锚①断言面）
	}
	if err := r.redeem(ctx, quoteSz-avail); err != nil {
		return fmt.Errorf("前置赎回: %w", err)
	}
	return nil
}

// errBalanceRead 余额读败标记（非钱动——下拍重试安全，不落 hold）。
type errBalanceRead struct{ err error }

func (e errBalanceRead) Error() string { return "quote 余额现读败: " + e.err.Error() }
func (e errBalanceRead) Unwrap() error { return e.err }

// launchDefinite 拉起棒定态判（okxexec 错误分类学——APIError=venue 业务
// 明拒 code≠0 已受理拒=定态；HTTPStatusError 4xx=边缘拒未入业务=定态；
// 5xx/超时/断连/解析败=不确定态（票钦歧态势域）→hold）。
func launchDefinite(err error) bool {
	var ae *okxexec.APIError
	if errors.As(err, &ae) {
		return true
	}
	var he *okxexec.HTTPStatusError
	if errors.As(err, &he) {
		return he.Code < 500
	}
	return false
}

// holdLaunch 歧态势抑重落账——hold 行（PK strategy_id ON CONFLICT 不重）
// +升件行（note 载人工处置径=核查后删 hold 行放行）。launchGate 候选
// 查询 NOT EXISTS hold 行=禁自动重试补单的持久载体（Runner 重启不失忆）。
func (r *Runner) holdLaunch(ctx context.Context, sid int64, inst, stage string, cause error) {
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_bot_launch_hold
		(strategy_id,inst_id,stage,reason) VALUES($1,$2,$3,$4)
		ON CONFLICT (strategy_id) DO NOTHING`, sid, inst, stage, cause.Error()); err != nil {
		log.Printf("[okxgrid] hold 行写败（#%d）: %v", sid, err)
	}
	r.esc(ctx, "okxgrid_launch_ambiguous", "钱径",
		fmt.Sprintf("策略#%d %s 拉起 %s 棒不确定态——hold 行落账禁自动重试补单（歧态势禁补单铁律）；人工核查钱态后删 fin_bot_launch_hold 行放行。cause: %v", sid, inst, stage, cause))
}
