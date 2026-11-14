// dirleg 方向族单腿执行件（棒5c——dir_perp_mom 真户径）：
// 方向=每次开仓一次可证伪信念——一张 perp 腿全载（无对冲腿、无 hedge_band、
// 无腿间 rollback）。open:direction→side(long_perp=buy/short_perp=sell),
// close:Close=true 反向平（buy 平 short/sell 平 long——reduceOnly 域界
// 由 buildOrderParams 合约腿自置）。
// 与 Execute 双腿径的关系：令牌验证/live 闸/赎回/腿行落账/仓行复用同函数；
// 省去的只有对冲专属件（band 判/leg2/rollback——单腿无腿间原子性问题，
// 败即败无半成品）。
package execution

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DirSide direction→腿 side（long_perp/short_perp 闭集——他向=拒构）。
func DirSide(direction string) (string, error) {
	switch direction {
	case "long_perp":
		return "buy", nil
	case "short_perp":
		return "sell", nil
	}
	return "", fmt.Errorf("direction 非闭集: %s", direction)
}

// BuildDirLeg 方向族单腿构：张数=notional→perpContracts→lot 网格→minSz 闸
// （同 BuildLegs 判据链，只省 spot 派生/半额对齐——单腿名义全额=notional）。
func BuildDirLeg(ctx context.Context, pool *pgxpool.Pool, loadMeta MetaLoader,
	venue, instID, direction string, notional float64) (Leg, error) {
	side, err := DirSide(direction)
	if err != nil {
		return Leg{}, err
	}
	m, err := loadMeta(ctx, instID)
	if err != nil {
		return Leg{}, fmt.Errorf("perp 元数据: %w", err)
	}
	px, err := LatestPx(ctx, pool, instID)
	if err != nil {
		return Leg{}, err
	}
	raw, err := perpContracts(notional, m.CtVal, m.CtValCcy, px)
	if err != nil {
		return Leg{}, err
	}
	sz := floorLot(raw, m.LotSz)
	if sz < m.MinSz {
		return Leg{}, fmt.Errorf("perp 最小张起不来: sz=%.4f<minSz=%.4f", sz, m.MinSz)
	}
	return Leg{Venue: venue, InstID: instID, Side: side,
		Sz: strconv.FormatFloat(sz, 'f', -1, 64), Notional: notional}, nil
}

// ExecuteDir 方向族单腿执行（open|close 同径——close 免 live 闸免赎同 Execute
// 契约）。in.Legs[0]=唯一腿（Legs[1] 不消费——调用侧 BuildDirLeg 单腿入槽）。
func (r *Runner) ExecuteDir(ctx context.Context, in Intent) (int64, error) {
	tok, err := r.verifyToken(ctx, in.DecisionID)
	if err != nil {
		return 0, err
	}
	if in.Kind != "close" {
		live, err := r.getB("invest.lifecycle.live_switch")
		if err != nil || !live {
			r.rej.Inc("live_switch_off")
			return 0, fmt.Errorf("live_switch_off")
		}
	}
	leg := in.Legs[0]
	if leg.InstID == "" || leg.Side == "" {
		return 0, fmt.Errorf("单腿件缺: inst=%q side=%q", leg.InstID, leg.Side)
	}
	var execID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES($1,$2,$3,$4,$5,'open') RETURNING id`,
		in.StrategyRef, in.DecisionID, tok, in.InstID, in.NotionalUSD).Scan(&execID); err != nil {
		r.rej.Inc("write_err")
		return 0, err
	}
	var posRef *int64
	if in.Kind == "close" {
		var pid int64
		if err := r.pool.QueryRow(ctx, `SELECT id FROM fin_position
			WHERE inst_id=$1 AND closed_at IS NULL ORDER BY id DESC LIMIT 1`,
			in.InstID).Scan(&pid); err == nil {
			posRef = &pid
		}
	}
	if r.earn != nil && in.Kind != "close" {
		if err := r.earn.RedeemToTrading(ctx, in.NotionalUSD); err != nil {
			r.setState(ctx, execID, "failed")
			r.rej.Inc("earn_redeem_failed")
			return execID, fmt.Errorf("前置赎回: %w", err)
		}
	}
	res, err := r.placeAndResolve(ctx, execID, 1, leg, posRef)
	if err != nil {
		if res.State == "unknown" || res.State == "live" || res.State == "partially_filled" {
			r.alert(ctx, "dir_leg_ambiguous",
				fmt.Sprintf("exec#%d 单腿 %s@%s 歧态势（state=%s）——venue 可能已成仓，残留敞口人查", execID, leg.InstID, leg.Venue, res.State))
		}
		r.setState(ctx, execID, "failed")
		return execID, fmt.Errorf("dir_leg: %w", err)
	}
	if in.Kind != "close" {
		var posID int64
		if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
			VALUES($1,$2,$3,$4) RETURNING id`,
			execID, in.StrategyRef, in.InstID, in.NotionalUSD).Scan(&posID); err != nil {
			r.rej.Inc("write_err")
		} else {
			r.linkLegCash(ctx, execID, posID)
		}
	}
	log.Printf("[execution] #%d %s 单腿成交（clOrdId %s dir）", execID, in.InstID, res.ClOrdID)
	return execID, nil
}
