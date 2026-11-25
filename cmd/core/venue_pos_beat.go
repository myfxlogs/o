package main

// venue_pos_beat.go —— O-33207 B 面 OKX 手动仓在仓快照环：
// /account/positions 全枚举（posLiveFor 同源读件族）→非 O 单
// upsert fin_venue_position（mtm_off=upl 直给——venue 现成 uPnL
// 带号进对账净值，不产 mark 价假算）；快照消失→'stale' 清出。
// O 单排除序（spec 禁线）：venue 仓 instId 命中 O 在持仓集
// （fin_position 未闭）=判 O 的不入册——同 instId 合并仓不可分
// =保守判 O（手动同 inst 残量漏记=宁缺毋滥不双簿，方向安全）。
// 平仓 realized 不在此径——衍生品 bill 族 fee+trade_pnl+unmapped_bill
// 已全量守恒入账（O-10063/#14671/O-39310 在案，O/手动同径）。
// 凭据缺席/读败=跳拍不清场（fail-closed 同 MT5 openPollOnce 径）。

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"o/internal/trading/okxexec"
)

const venuePosEvery = 60 * time.Second // sched-exempt-ok 会话内拍（同 posLiveFor TTL 粒度）

// venuePosAcctID 账户归户——fin_account 注册表解析（venue='okx'
// sub_type='trading' active 行；册缺=跳拍不猜）。
func venuePosAcctID(ctx context.Context, pool *pgxpool.Pool) string {
	var id string
	_ = pool.QueryRow(ctx,
		`SELECT account_id FROM fin_account WHERE venue='okx' AND sub_type='trading' AND state='active' LIMIT 1`).Scan(&id)
	return id
}

// ownInsts O 在持仓 inst 集（排除判据源——fin_position 未闭态全枚举；
// 读败=err 上抛让位本拍，不 fail-open 误判手动）。
func ownInsts(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx,
		`SELECT DISTINCT inst_id FROM fin_position WHERE state NOT IN ('closed','orphaned')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out[s] = true
	}
	return out, rows.Err()
}

// venuePosOnce 单拍：positionsFn 枚举→排除 O 单→upsert+stale 清场
// （函数缝=桩测可注——生产径=okcli.Positions 包一层）。
func venuePosOnce(ctx context.Context, pool *pgxpool.Pool,
	positionsFn func(context.Context) ([]okxexec.Position, error)) error {
	poss, err := positionsFn(ctx)
	if err != nil {
		return err
	}
	acctID := venuePosAcctID(ctx, pool)
	if acctID == "" {
		return nil
	}
	own, err := ownInsts(ctx, pool)
	if err != nil {
		return err
	}
	seen := make([]string, 0, len(poss))
	for _, p := range poss {
		if own[p.InstID] {
			continue // O 在持同 inst——合并仓不可分保守判 O 的（不双簿）
		}
		ticket := p.PosID
		if ticket == "" {
			ticket = p.InstID + ":" + p.PosSide
		}
		side := "buy"
		if p.PosSide == "short" {
			side = "sell"
		} else if p.PosSide == "net" {
			if v, perr := strconv.ParseFloat(p.Pos, 64); perr == nil && v < 0 {
				side = "sell"
			}
		}
		lots, _ := strconv.ParseFloat(p.Pos, 64)
		if lots == 0 {
			continue // 零仓行（venue 全枚举含平仓残行）——不入册
		}
		if lots < 0 {
			lots = -lots
		}
		var avgPx, upl *float64
		if v, perr := strconv.ParseFloat(p.AvgPx, 64); perr == nil {
			avgPx = &v
		}
		if v, perr := strconv.ParseFloat(p.Upl, 64); perr == nil {
			upl = &v
		}
		ccy := p.Ccy
		if ccy == "" {
			ccy = "USDT"
		}
		seen = append(seen, ticket)
		if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_position
			(account_id,venue,ticket,symbol,side,lots,contract_size,open_px,open_ts,ccy,state,mtm_off,last_seen)
			VALUES($1,'okx',$2,$3,$4,$5,0,$6,$7,$8,'open',$9,now())
			ON CONFLICT (account_id,ticket) DO UPDATE SET
			  symbol=EXCLUDED.symbol,side=EXCLUDED.side,lots=EXCLUDED.lots,
			  open_px=EXCLUDED.open_px,ccy=EXCLUDED.ccy,state='open',
			  mtm_off=EXCLUDED.mtm_off,last_seen=now()
			WHERE fin_venue_position.state<>'closed'`,
			acctID, ticket, p.InstID, side, lots, avgPx,
			msTime(p.CTime), ccy, upl); err != nil {
			log.Printf("[venue-pos] okx 手动仓快照写败 票=%s: %v", ticket, err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_venue_position SET state='stale'
		WHERE account_id=$1 AND venue='okx' AND state='open' AND NOT (ticket = ANY($2))`,
		acctID, seen); err != nil {
		return err
	}
	return nil
}

// msTime venue ms 串→time（空/坏=零值——open_ts 可空列容忍）。
func msTime(ms string) *time.Time {
	v, err := strconv.ParseInt(ms, 10, 64)
	if err != nil || v <= 0 {
		return nil
	}
	t := time.UnixMilli(v).UTC()
	return &t
}

// venuePosBeat OKX 手动仓快照环——首拍即跑（在仓全量回填）此后 60s。
// okcli=nil=凭据缺席整环让位（读件缺不假零——posLiveFor 同约）。
func venuePosBeat(ctx context.Context, pool *pgxpool.Pool, okcli *okxexec.Client) {
	if okcli == nil {
		return
	}
	fn := func(ctx context.Context) ([]okxexec.Position, error) {
		return okcli.Positions(ctx)
	}
	for {
		if err := venuePosOnce(ctx, pool, fn); err != nil && ctx.Err() == nil {
			log.Printf("[venue-pos] okx 在仓快照轮询败（下拍重试）: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(venuePosEvery): // sched-exempt-ok 会话内增量拍
		}
	}
}
