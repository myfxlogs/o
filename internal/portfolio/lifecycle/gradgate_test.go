package lifecycle

import (
	"context"
	"strings"
	"testing"
)

// T-0521 毕业判词桩测——锚①confGate：样本不足 fail-closed/同额稳流 LB>0 放/
// 混号流 LB≤0 拦；O-41389 硬化锚=零方差条款+符号游程检验（TestConfGateHardening）；
// 锚②fundingEntryGate：perp 方向×rate 同号放/逆号拦/
// 非 perp 腿无条件放/行缺拦；锚③volEntryGate：APR<中位×0.5 拦；
// 锚④macroEventGate：爆点在窗拦/窗内无件放。

func seedFills(t *testing.T, ctx context.Context, r *Runner, sid int64, amts ...float64) {
	t.Helper()
	for _, a := range amts {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd)
			VALUES($1,'funding','X-USDT-SWAP',$2)`, sid, a); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConfGate(t *testing.T) {
	r, ctx := frictionFixture(t)
	sid := mkPaperStrategy(t, ctx, r)
	if ok, _ := r.confGate(ctx, sid, "funding-hedge"); ok {
		t.Fatal("零收入笔应拦（样本不足 fail-closed）")
	}
	seedFills(t, ctx, r, sid, 0.1, 0.1)
	if ok, _ := r.confGate(ctx, sid, "funding-hedge"); ok {
		t.Fatal("2 笔应拦（<5）")
	}
	// O-41389：5 笔同额正 sd=0∧N<8=零方差捷径拦（旧闸三笔即放——本锚先见红）。
	seedFills(t, ctx, r, sid, 0.1, 0.1, 0.1)
	if ok, why := r.confGate(ctx, sid, "funding-hedge"); ok || !strings.Contains(why, "零方差") {
		t.Fatalf("5笔同额正应拦零方差捷径: ok=%v why=%s", ok, why)
	}
	// 8 笔同额正——零方差条款放行径（等额八连=真一致可辩护）。
	seedFills(t, ctx, r, sid, 0.1, 0.1, 0.1)
	if ok, why := r.confGate(ctx, sid, "funding-hedge"); !ok {
		t.Fatalf("8笔同额稳流应放（sd=0 N≥8 零方差放行径）: %s", why)
	}
	// 混号流——大笔负笔插入→LB 翻负拦。
	seedFills(t, ctx, r, sid, -0.5, -0.5, -0.5, -0.5, -0.5, -0.5)
	if ok, _ := r.confGate(ctx, sid, "funding-hedge"); ok {
		t.Fatal("混号流 LB≤0 应拦")
	}
}

// mkPaperStrategyAt 同 mkPaperStrategy 但 inst_id 可参——多策略同 fixture
// 撞 fin_strategy_active_dedupe（inst+template 唯一），异 inst_id 绕开。
func mkPaperStrategyAt(t *testing.T, ctx context.Context, r *Runner, inst string) int64 {
	t.Helper()
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction,venue)
		VALUES($1,'funding-hedge','paper',50,'short_perp_long_spot','okx')
		RETURNING id`, inst).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

// O-41389 硬化对抗锚：①confMinFills 3→5+零方差条款（sd=0 须 N≥8 方放——
// 杀「三笔等额正秒过」捷径）；②N≥30 符号游程检验（正/非正序列游程过少
// =同向聚簇=独立假设证伪→Z<−1.96 拦；N<30 判词升显式「证据期」行；
// n₁∨n₂=0 全同号=检验退化按零方差条款判）。
func TestConfGateHardening(t *testing.T) {
	r, ctx := frictionFixture(t)
	// 3 笔等额正——旧闸 sd=0 LB=mean>0 秒放；新闸 N=3<5 样本不足拦。
	sid := mkPaperStrategyAt(t, ctx, r, "H1-USDT-SWAP")
	seedFills(t, ctx, r, sid, 0.1, 0.1, 0.1)
	if ok, why := r.confGate(ctx, sid, "funding-hedge"); ok || !strings.Contains(why, "样本不足") {
		t.Fatalf("3笔等额正应拦(N<5): ok=%v why=%s", ok, why)
	}
	// 5 笔异额正 sd>0 LB>0——正常径放行+「证据期」判词在案（N<30 检验未达阈）。
	sid2 := mkPaperStrategyAt(t, ctx, r, "H2-USDT-SWAP")
	seedFills(t, ctx, r, sid2, 0.4, 0.5, 0.6, 0.45, 0.55)
	if ok, why := r.confGate(ctx, sid2, "funding-hedge"); !ok || !strings.Contains(why, "证据期") {
		t.Fatalf("5笔异额正应放+证据期判词: ok=%v why=%s", ok, why)
	}
	// 30 笔符号聚簇（20 正+10 负两簇，序=插入序）——游程检验手工算验：
	// n₁=20 n₂=10 N=30 R=2，E=2·20·10/30+1≈14.33，Var=400·370/(900·29)≈5.67，
	// Z=(2−14.33)/√5.67≈−5.18<−1.96→独立性证伪拦（LB 本身>0 不误伤判据序）。
	sid3 := mkPaperStrategyAt(t, ctx, r, "H3-USDT-SWAP")
	clustered := make([]float64, 0, 30)
	for i := 0; i < 20; i++ {
		clustered = append(clustered, 0.5)
	}
	for i := 0; i < 10; i++ {
		clustered = append(clustered, -0.1)
	}
	seedFills(t, ctx, r, sid3, clustered...)
	if ok, why := r.confGate(ctx, sid3, "funding-hedge"); ok || !strings.Contains(why, "独立性证伪") {
		t.Fatalf("30笔符号聚簇应拦(独立性证伪): ok=%v why=%s", ok, why)
	}
	// 30 笔正负交错——R=30，E=16，Var=450·420/(900·29)≈7.24，
	// Z=(30−16)/√7.24≈+5.21 游程充足+LB>0 放。
	sid4 := mkPaperStrategyAt(t, ctx, r, "H4-USDT-SWAP")
	alt := make([]float64, 0, 30)
	for i := 0; i < 15; i++ {
		alt = append(alt, 0.5, -0.1)
	}
	seedFills(t, ctx, r, sid4, alt...)
	if ok, why := r.confGate(ctx, sid4, "funding-hedge"); !ok {
		t.Fatalf("30笔交错应放(runs test 过): %s", why)
	}
	// 30 笔全同号异额——游程检验退化（n₂=0）按零方差条款判：sd>0→LB 决放。
	sid5 := mkPaperStrategyAt(t, ctx, r, "H5-USDT-SWAP")
	uni := make([]float64, 0, 30)
	for i := 0; i < 30; i++ {
		uni = append(uni, 0.4+0.02*float64(i%5))
	}
	seedFills(t, ctx, r, sid5, uni...)
	if ok, why := r.confGate(ctx, sid5, "funding-hedge"); !ok {
		t.Fatalf("30笔全同号异额应放(退化零方差条款判): %s", why)
	}
}

// O-36961 锚：方向动量族置信序列=alpha 本体（kind='pnl'）——funding 租金
// 附带流不再稀释 LB；carry 族混采不变。生产实证：256/259 paper=dir_perp_mom
// 混采下 funding: pnl≈19:1，LB 测的是租金符号非动量。
func TestConfGateDirAlphaStream(t *testing.T) {
	r, ctx := frictionFixture(t)
	sid := mkPaperStrategy(t, ctx, r)
	// funding 噪声流（混号大方差——混采下 LB 必负）；pnl alpha 流（8 笔异额正——
	// O-41389 后 N≥5 且 sd>0 方过闸，旧 3 笔同额形已被零方差条款拦）。
	seedFills(t, ctx, r, sid, -0.5, 0.02, -0.4, 0.01, -0.3, -0.02)
	for _, a := range []float64{0.08, 0.1, 0.12, 0.09, 0.11, 0.1, 0.095, 0.105} {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd)
			VALUES($1,'pnl','X-USDT-SWAP',$2)`, sid, a); err != nil {
			t.Fatal(err)
		}
	}
	// dir 族只采 pnl——8 笔异额 sd>0 LB>0 放（funding 噪声不入列）。
	if ok, why := r.confGate(ctx, sid, "dir_perp_mom"); !ok {
		t.Fatalf("dir 族应只采 pnl 流放行: %s", why)
	}
	// 对照：carry 族混采——funding 噪声入列 LB≤0 拦（分流不误伤余族）。
	if ok, _ := r.confGate(ctx, sid, "funding-hedge"); ok {
		t.Fatal("carry 族混采应拦（funding 噪声 LB≤0）")
	}
	// funding-extreme-fade 留混采（funding 即其 alpha 成分）。
	if ok, _ := r.confGate(ctx, sid, "funding-extreme-fade"); ok {
		t.Fatal("funding-extreme-fade 混采应拦")
	}
}

func TestFundingEntryGate(t *testing.T) {
	r, ctx := frictionFixture(t)
	// 非 perp 腿（xvenue mt5 cfd 向）无条件放。
	if ok, _ := r.fundingEntryGate(ctx, "XAUUSD", "short_cfd"); !ok {
		t.Fatal("非 perp 方向应放行（判词只钉 perp 腿）")
	}
	// perp 腿行缺=拦（事实缺不虚构）。
	if ok, _ := r.fundingEntryGate(ctx, "X-USDT-SWAP", "short_perp_long_spot"); ok {
		t.Fatal("funding 行缺应拦（延评不虚构）")
	}
	// 正率+short_perp=同号放；负率=逆号拦。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate)
		VALUES(now(),'X-USDT-SWAP',0.001)`); err != nil {
		t.Fatal(err)
	}
	if ok, _ := r.fundingEntryGate(ctx, "X-USDT-SWAP", "short_perp_long_spot"); !ok {
		t.Fatal("正率 short_perp 收租位应放")
	}
	if ok, _ := r.fundingEntryGate(ctx, "X-USDT-SWAP", "long_perp_short_spot"); ok {
		t.Fatal("正率 long_perp=付租位应拦")
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate)
		VALUES(now(),'X-USDT-SWAP',-0.002)`); err != nil {
		t.Fatal(err)
	}
	if ok, _ := r.fundingEntryGate(ctx, "X-USDT-SWAP", "short_perp_long_spot"); ok {
		t.Fatal("最新负率 short_perp 收租位失守应拦")
	}
	if ok, _ := r.fundingEntryGate(ctx, "X-USDT-SWAP", "long_perp_short_spot"); !ok {
		t.Fatal("负率 long_perp 收租位应放")
	}
}

func TestVolEntryGate(t *testing.T) {
	r, ctx := frictionFixture(t)
	// 立项 APR 缺=拦（基准不虚构）；快照缺=拦。
	if ok, _ := r.volEntryGate(ctx, "BTC-USDT-261225-68000-P", 0); ok {
		t.Fatal("refAPR=0 应拦（立项基准缺）")
	}
	if ok, _ := r.volEntryGate(ctx, "BTC-USDT-261225-68000-P", 0.12); ok {
		t.Fatal("无快照应拦")
	}
	// 报价状态行 APR=0.03 <立项 0.12×0.5(0.06)→拦；更新 0.14≥0.06→放。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,direction,
		strike,expiry,annualized_yield,spot,raw)
		VALUES(now(),'okx','products','BTC-USDT','BTC-USDT-261225-68000-P','P',
		68000,now()+interval '30 days',0.03,88000,'{}')`); err != nil {
		t.Fatal(err)
	}
	if ok, _ := r.volEntryGate(ctx, "BTC-USDT-261225-68000-P", 0.12); ok {
		t.Fatal("APR 0.03<立项×0.5 应拦（收益率塌于立项半值不卖）")
	}
	if _, err := r.pool.Exec(ctx, `UPDATE md_dcd_product SET annualized_yield=0.14
		WHERE product_id='BTC-USDT-261225-68000-P'`); err != nil {
		t.Fatal(err)
	}
	if ok, _ := r.volEntryGate(ctx, "BTC-USDT-261225-68000-P", 0.12); !ok {
		t.Fatal("APR 0.14≥立项×0.5 应放")
	}
}

func TestMacroEventGate(t *testing.T) {
	r, ctx := frictionFixture(t)
	if ok, _ := r.macroEventGate(ctx); !ok {
		t.Fatal("历空/当日无件应放")
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_macro_event(event_date,kind,title,source)
		VALUES(current_date,'cpi','CPI y/y','ffcal')`); err != nil {
		t.Fatal(err)
	}
	if ok, _ := r.macroEventGate(ctx); ok {
		t.Fatal("当日 CPI=爆点日应拦（全天延开）")
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM md_macro_event`); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_macro_event(event_date,kind,title,source)
		VALUES(current_date-1,'fomc','FOMC','ffcal')`); err != nil {
		t.Fatal(err)
	}
	if ok, _ := r.macroEventGate(ctx); !ok {
		t.Fatal("昨日事件应放")
	}
}
