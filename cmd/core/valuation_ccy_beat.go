package main

// valuation_ccy_beat.go —— O-41395 件② SG 侧估值需求币发布器：
// 活跃集=动态持仓联动册（legset.ActiveSetUnion 臂派生），估值仍需币
// 在策略退役后无臂可出=BTC-USDT tick 死订实证（2026-11-17 冻结）。
// 本件=需求源生产端：FundCcy∪fin_position_view.ccy 集合→fin_valuation_ccy
// 集同步（在集 upsert、出集 DELETE——projector ev_valuation 帧反向镜像
// PG-md→legset.ValuationCcyArm 订阅集/写谓词消费，订而不写同源闸在册）。
// 同值零写——ev 触发器语句级 NOTIFY，空写=空醒。

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger"
)

const valuationCcyEvery = 5 * time.Minute

// valuationCcyExcluded 直入币不发布（USDT/USD 估值面直入不需对价；
// legset.ValuationCcyArm 兜底同滤——双闸防脏行）。
func valuationCcyExcluded(ccy string) bool {
	return ccy == "" || ccy == "USDT" || ccy == "USD"
}

// syncValuationCcy 集合同步一拍：want=FundCcy distinct ccy ∪
// fin_position_view.ccy（dcd 凭据币——FundCcy dcd 域已含 baseline/
// settle 溯源币，视图兜底防口径缺口）；与现表全等=零写返 nil。
func syncValuationCcy(ctx context.Context, pool *pgxpool.Pool) error {
	want := map[string]map[string]bool{} // ccy → src 集
	add := func(ccy, src string) {
		if valuationCcyExcluded(ccy) {
			return
		}
		if want[ccy] == nil {
			want[ccy] = map[string]bool{}
		}
		want[ccy][src] = true
	}
	fund, err := ledger.FundCcy(ctx, pool)
	if err != nil {
		return fmt.Errorf("FundCcy 读: %w", err)
	}
	for k := range fund {
		add(k[1], "fundccy")
	}
	vrows, err := pool.Query(ctx,
		`SELECT DISTINCT ccy FROM fin_position_view WHERE ccy IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("持仓视图 ccy 读: %w", err)
	}
	for vrows.Next() {
		var ccy string
		if err := vrows.Scan(&ccy); err != nil {
			vrows.Close()
			return fmt.Errorf("持仓视图 ccy 扫: %w", err)
		}
		add(ccy, "view")
	}
	vrows.Close()
	if err := vrows.Err(); err != nil {
		return fmt.Errorf("持仓视图 ccy 迭代: %w", err)
	}
	cur := map[string]string{}
	crows, err := pool.Query(ctx, `SELECT ccy,src FROM fin_valuation_ccy`)
	if err != nil {
		return fmt.Errorf("现表读: %w", err)
	}
	for crows.Next() {
		var c, s string
		if err := crows.Scan(&c, &s); err != nil {
			crows.Close()
			return fmt.Errorf("现表扫: %w", err)
		}
		cur[c] = s
	}
	crows.Close()
	if err := crows.Err(); err != nil {
		return fmt.Errorf("现表迭代: %w", err)
	}
	// src 串=需求源并集（fundccy/view/fundccy+view——观测面如实标）。
	srcOf := func(srcs map[string]bool) string {
		var parts []string
		for s := range srcs {
			parts = append(parts, s)
		}
		sort.Strings(parts)
		return strings.Join(parts, "+")
	}
	changed := false
	for ccy, srcs := range want {
		if cur[ccy] != srcOf(srcs) {
			changed = true
			break
		}
	}
	if !changed {
		for ccy := range cur {
			if _, ok := want[ccy]; !ok {
				changed = true
				break
			}
		}
	}
	if !changed {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("同步 tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var keys []string
	for ccy := range want {
		keys = append(keys, ccy)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM fin_valuation_ccy WHERE NOT (ccy = ANY($1))`, keys); err != nil {
		return fmt.Errorf("出集删: %w", err)
	}
	for ccy, srcs := range want {
		if _, err := tx.Exec(ctx,
			`INSERT INTO fin_valuation_ccy(ccy,src,updated_at) VALUES($1,$2,now())
			 ON CONFLICT (ccy) DO UPDATE SET src=EXCLUDED.src,updated_at=now()`,
			ccy, srcOf(srcs)); err != nil {
			return fmt.Errorf("入集 upsert %s: %w", ccy, err)
		}
	}
	return tx.Commit(ctx)
}

// valuationCcyBeat 发布节拍（boot 即一拍+5min 周期——持仓/账户变更
// 尾延迟≤5min 入订阅集；币退集同径出集退订）。读败记日志下拍重试
// 不炸环（发布面缺席=沿用现表，collector 侧读败另有闸）。
func valuationCcyBeat(ctx context.Context, pool *pgxpool.Pool) {
	tk := time.NewTicker(valuationCcyEvery) // sched-exempt-ok：估值币集期界同步——多源派生集（FundCcy∪持仓视图）无单一事件面可订，周期对账+同值跳写同 retention 巡面律
	defer tk.Stop()
	for {
		if err := syncValuationCcy(ctx, pool); err != nil {
			log.Printf("[valuation-ccy] 同步败: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
		}
	}
}
