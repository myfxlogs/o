package risk

import (
	"strconv"
	"testing"
)

// T-0355 验收锚①：dcd_baseline 凭据行计入 exposure_cap 敞口——
// 估值函数直断 + 闸级边界断言（卡 max_exposure 证明币量→USD 真入判据）。
func TestDcdBaselineCountsTowardExposureCap(t *testing.T) {
	ctx, g := setup(t)
	// 清场：迁移种子+他件 dcd 凭据全出——本测敞口=dcd 变量独定。
	if _, err := g.pool.Exec(ctx,
		`DELETE FROM fin_risk_decision WHERE intent->>'op'='dcd_baseline'`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		g.pool.Exec(ctx, `DELETE FROM fin_risk_decision WHERE intent->>'op'='dcd_baseline'`)
	}()
	// fixture：allow 凭据 BTC 1.0；tick BTC-USDT 改 50000（setup 默认 100000）
	// → dcdUSD=50000。
	if _, err := g.pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"kind":"transfer","op":"dcd_baseline","venue":"okx","ccy":"BTC","amt":1}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := g.pool.Exec(ctx,
		`UPDATE md_ticks SET last=50000 WHERE venue='okx' AND inst_id='BTC-USDT'`); err != nil {
		t.Fatal(err)
	}
	// 估值直断：amt×最新 tick=50000 USD。
	usd, ok := g.dcdExposureUSD(ctx)
	if !ok || usd != 50000 {
		t.Fatalf("dcdExposureUSD=%v ok=%v want 50000", usd, ok)
	}
	// mark_index 径同证：ticks 行撤、mark_index px=50000 → 估值不变。
	if _, err := g.pool.Exec(ctx,
		`DELETE FROM md_ticks WHERE venue='okx' AND inst_id='BTC-USDT'`); err != nil {
		t.Fatal(err)
	}
	if _, err := g.pool.Exec(ctx,
		`INSERT INTO md_mark_index(ts,venue,inst_id,channel,px) VALUES(now(),'okx','BTC-USDT','mark',50000)
		 ON CONFLICT(venue,inst_id,channel) DO UPDATE SET ts=EXCLUDED.ts,px=EXCLUDED.px`); err != nil {
		t.Fatal(err)
	}
	if usd, ok = g.dcdExposureUSD(ctx); !ok || usd != 50000 {
		t.Fatalf("mark_index 径 dcdExposureUSD=%v ok=%v want 50000", usd, ok)
	}
	// 闸级边界：cur=exec+inflight 实测基线，max=cur+50000+100-1 →
	// tExp+dcd+notional=cur+50100 > max → exposure_cap deny（dcd 不入则放行）。
	var cur float64
	if err := g.pool.QueryRow(ctx, `SELECT COALESCE(sum(x),0) FROM (
		  SELECT sum(notional_usd) x FROM fin_position WHERE closed_at IS NULL
		  UNION ALL SELECT sum(notional_usd) FROM fin_execution WHERE state='open') t`).Scan(&cur); err != nil {
		t.Fatal(err)
	}
	setMax := func(v float64) {
		t.Helper()
		if _, err := g.pool.Exec(ctx,
			`UPDATE app_config SET value=to_jsonb($1::text) WHERE key='risk.max_exposure_usd'`,
			strconv.FormatFloat(v, 'f', -1, 64)); err != nil {
			t.Fatal(err)
		}
		if err := g.cfg.Load(ctx); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		g.pool.Exec(ctx,
			`UPDATE app_config SET value=to_jsonb('1000000000'::text) WHERE key='risk.max_exposure_usd'`)
	}()
	setMax(cur + 50099)
	if d := g.Check(ctx, intent); d.Allow || d.Reason != "exposure_cap" {
		t.Fatalf("dcd 计入应压线拒: %+v", d)
	}
	// 对照：max 放宽 1100 → 同仓位放行（证明前拒=敞口真计非恒拒）。
	setMax(cur + 51100)
	if d := g.Check(ctx, intent); !d.Allow {
		t.Fatalf("放宽后应放行: %+v", d)
	}
}

// T-0364 验收锚②③④：业主域凭据剔出策略敞口——域册标记行视图分流
// kind='dcd_owner' 证据行可查，闸面分子=0 放行；视图读败 fail-closed 仍拒。
func TestDcdOwnerDomainExcluded(t *testing.T) {
	ctx, g := setup(t)
	// 迁移面锚：0043 种子凭据（ordId=11448262）=本票收归对象——0115 域册
	// 已标记落册；视图行 kind='dcd_owner' 在账可查（业主域标记证据行）。
	var kind string
	if err := g.pool.QueryRow(ctx, `SELECT kind FROM fin_position_view
		WHERE kind LIKE 'dcd%'`).Scan(&kind); err != nil || kind != "dcd_owner" {
		t.Fatalf("业主域行 kind=%q err=%v 应 dcd_owner", kind, err)
	}
	// 敞口分子剔出：估值=0（业主域行连 tick 估值径都不入）+闸面放行。
	if usd, ok := g.dcdExposureUSD(ctx); !ok || usd != 0 {
		t.Fatalf("业主域剔出后 dcdExposureUSD=%v ok=%v want 0", usd, ok)
	}
	if d := g.Check(ctx, intent); !d.Allow {
		t.Fatalf("业主域剔出应放行: %+v", d)
	}
	// 对照：同币策略域凭据 1 BTC 照计分子（域册无行=strategy 缺省——证明
	// 剔出来源=域册标记非 dcd 全免）。setup tick BTC-USDT=100000。
	if _, err := g.pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"kind":"transfer","op":"dcd_baseline","venue":"okx","ccy":"BTC","amt":1}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		g.pool.Exec(ctx, `DELETE FROM fin_risk_decision
			WHERE intent->>'op'='dcd_baseline' AND intent->>'amt'='1'`)
	}()
	if usd, ok := g.dcdExposureUSD(ctx); !ok || usd != 100000 {
		t.Fatalf("策略域应计 dcdExposureUSD=%v ok=%v want 100000", usd, ok)
	}
	// fail-closed 回归锚：视图读败仍拒放（rename 遮视图合成读败径）。
	if _, err := g.pool.Exec(ctx, `ALTER VIEW fin_position_view RENAME TO fin_position_view_bak`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		g.pool.Exec(ctx, `ALTER VIEW fin_position_view_bak RENAME TO fin_position_view`)
	}()
	if d := g.Check(ctx, intent); d.Allow || d.Reason != "exposure_cap" {
		t.Fatalf("视图读败应 fail-closed 拒放: %+v", d)
	}
}

// T-0355 验收锚②：tick 缺价=fail-closed——估不出敞口拒放（非零计放行），
// 与显示面「无 tick=0」口径相反为钦定语义。
func TestDcdBaselineUnpricedFailsClosed(t *testing.T) {
	ctx, g := setup(t)
	if _, err := g.pool.Exec(ctx,
		`DELETE FROM fin_risk_decision WHERE intent->>'op'='dcd_baseline'`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		g.pool.Exec(ctx, `DELETE FROM fin_risk_decision WHERE intent->>'op'='dcd_baseline'`)
		g.pool.Exec(ctx, `DELETE FROM md_ticks WHERE inst_id LIKE 'ZZ%'`)
	}()
	// 无 tick 币 ZZX（两表皆无 ZZX-USDT 行）凭据 1 枚 → 估不出→拒放。
	if _, err := g.pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"kind":"transfer","op":"dcd_baseline","venue":"okx","ccy":"ZZX","amt":1}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	if d := g.Check(ctx, intent); d.Allow || d.Reason != "exposure_cap" {
		t.Fatalf("无 tick 应 fail-closed 拒放: %+v", d)
	}
	// 混合仓位同拒：已入 ZZX tick 后再添无价 ZZY——任一不可估即拒。
	if _, err := g.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'okx','ZZX-USDT',10)
		 ON CONFLICT(venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`); err != nil {
		t.Fatal(err)
	}
	if _, err := g.pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"kind":"transfer","op":"dcd_baseline","venue":"okx","ccy":"ZZY","amt":1}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	if d := g.Check(ctx, intent); d.Allow || d.Reason != "exposure_cap" {
		t.Fatalf("混合含无价币应拒放: %+v", d)
	}
	// 去 ZZY、ZZX tick 置 NULL 价 → 价面坏同拒（NULL≠可估零）。
	if _, err := g.pool.Exec(ctx,
		`DELETE FROM fin_risk_decision WHERE intent->>'ccy'='ZZY'`); err != nil {
		t.Fatal(err)
	}
	if _, err := g.pool.Exec(ctx,
		`UPDATE md_ticks SET last=NULL WHERE venue='okx' AND inst_id='ZZX-USDT'`); err != nil {
		t.Fatal(err)
	}
	if d := g.Check(ctx, intent); d.Allow || d.Reason != "exposure_cap" {
		t.Fatalf("NULL 价应 fail-closed 拒放: %+v", d)
	}
	// 对照复位：ZZX 价回 10 → 同凭据放行（拒因=无价非他因）。
	if _, err := g.pool.Exec(ctx,
		`UPDATE md_ticks SET last=10 WHERE venue='okx' AND inst_id='ZZX-USDT'`); err != nil {
		t.Fatal(err)
	}
	if d := g.Check(ctx, intent); !d.Allow {
		t.Fatalf("有价后应放行: %+v", d)
	}
}
