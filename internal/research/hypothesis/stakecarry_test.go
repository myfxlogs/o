// stakecarry_test.go —— stake_carry 双头臂桩测（T-0320 验收锚③④）：
// 能力面二维积（staking×swap-carry on ccy）→TRX 同构提案链——
// staking 注册+cmc 实收+trx 质押率→卡+评估（工单#21612：stake-carry
// 已入 execfam 册作参考模型面——Drillable 永假无演习腿形，live 闸
// 自然拦；eval=proposed+提案入队，paper 簿记径通）；
// 质押率缺席/carry 面非正=不枚举如实。
package hypothesis

import (
	"context"
	"testing"
)

// seedStakeCarry TRX 双头三件套：staking 注册行+trx 质押率+cmc 实收序列。
func seedStakeCarry(t *testing.T, ctx context.Context, r *Runner) {
	t.Helper()
	for _, q := range []string{
		`INSERT INTO fin_capability_surface(venue,surface,endpoint_family,product_family,rw_class,cost_attrs,enabled) VALUES
		 ('tron','staking-TRX','tron-chain-staking','staking','read',
		  '{"ccy":"TRX","stakable":true,"rate_source":"trx_onchain"}',true),
		 ('cmc','swap-carry-TRX','cmc-rest:history','swap-carry','read',
		  '{"ccy":"TRX","instrument":"X-AFQUL","direction":"short","rate_source":"cmc_realized"}',true)
		 ON CONFLICT (venue,surface) DO NOTHING`,
		`INSERT INTO fin_trx_vote_apr(fetched_at,current_rate,max_rate,min_rate,spread)
		 VALUES(now(),3.33,3.35,3.30,0.05)`,
		`INSERT INTO fin_trx_account(address,votes_total,balance_sun,allowance_sun,energy_delegated,bandwidth_staked,net_limit)
		 VALUES('TTEST',0,0,0,18100000000,0,0)`,
		`INSERT INTO fin_cmc_positions(account_id,instrument_id,direction,quantity,entry_price)
		 VALUES(1,'X-AFQUL','Sell',18100,0.3317)`,
		`INSERT INTO fin_cmc_history(account_id,transaction_id,transaction_type,amount,currency,transaction_time)
		 SELECT 1,'tx-'||g,'OVERNIGHT_SETTLEMENT',1.64,'USD',now()-(g||' days')::interval
		 FROM generate_series(1,10) g`,
	} {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	// 配置三件（阈/缓冲/质押率行龄）。
	xseed(t, ctx, r, map[string]string{
		"research.signal.stake_carry_net_bps":    "300",
		"research.signal.stake_carry_buffer_bps": "50",
		"research.stake.stale_hours":             "168",
	})
}

// 锚④：TRX 标定——枚举面自动产出 cmc:TRX 同构评估（卡 family=stake_carry
// 立+eval 落行；工单#21612：stake-carry 入册参考模型→proposed+
// fin_proposal 行立——纸面簿记径通，真执行臂不建 live 闸 Drillable 拦）。
func TestStakeCarryTrx(t *testing.T) {
	ctx, r := xsetup(t)
	seedStakeCarry(t, ctx, r)
	r.stakeCarryScan(ctx)
	var fam, state string
	if err := r.pool.QueryRow(ctx, `SELECT family,state FROM fin_opp_hypothesis
		WHERE venue='cmc' AND surface='stake_carry-TRX'`).Scan(&fam, &state); err != nil {
		t.Fatal("stake_carry-TRX 卡应立:", err)
	}
	if fam != "stake_carry" || state != "proposed" {
		t.Fatalf("卡 family/state=%s/%s", fam, state)
	}
	var verdict string
	var net float64
	var ev map[string]any
	if err := r.pool.QueryRow(ctx, `SELECT verdict,net_apr,evidence FROM fin_opp_eval
		WHERE inst_id='cmc:TRX' AND template='stake-carry'`).Scan(&verdict, &net, &ev); err != nil {
		t.Fatal("cmc:TRX 评估行应落:", err)
	}
	// stake≈333bps+cmc≈1000bps−50→≈1283bps≈0.1283。
	// 工单#21612：stake-carry 入册参考模型→前置拒解，eval=proposed——
	// 净年化分量如实落 evidence/net_apr 列。
	if verdict != "proposed" || net < 0.10 {
		t.Fatalf("eval verdict/reason/net=%s/%v/%v（应 proposed/≈0.13）",
			verdict, ev["reason"], net)
	}
	if ev["stake_apr_bps"].(float64) <= 0 || ev["short_ann_bps"].(float64) <= 0 {
		t.Fatalf("双腿分量缺: %v", ev)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='cmc:TRX'`); n != 1 {
		t.Fatalf("入册提案应立 1 行: %d", n)
	}
}

// 锚：质押率缺席=不枚举（双头缺一腿如实不产——vote 表空→零卡零评估）。
func TestStakeCarryNoYield(t *testing.T) {
	ctx, r := xsetup(t)
	seedStakeCarry(t, ctx, r)
	if _, err := r.pool.Exec(ctx, `DELETE FROM fin_trx_vote_apr`); err != nil {
		t.Fatal(err)
	}
	r.stakeCarryScan(ctx)
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE surface='stake_carry-TRX'`); n != 0 {
		t.Fatalf("质押率缺席应零卡: %d", n)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_eval
		WHERE template='stake-carry'`); n != 0 {
		t.Fatalf("质押率缺席应零评估: %d", n)
	}
}
