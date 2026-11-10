package lifecycle

import (
	"context"
	"testing"
)

// T-0521 毕业判词桩测——锚①confGate：<3 笔 fail-closed/同额稳流 LB>0 放/
// 混号流 LB≤0 拦；锚②fundingEntryGate：perp 方向×rate 同号放/逆号拦/
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
	if ok, _ := r.confGate(ctx, sid); ok {
		t.Fatal("零收入笔应拦（样本不足 fail-closed）")
	}
	seedFills(t, ctx, r, sid, 0.1, 0.1)
	if ok, _ := r.confGate(ctx, sid); ok {
		t.Fatal("2 笔应拦（<3）")
	}
	seedFills(t, ctx, r, sid, 0.1)
	if ok, _ := r.confGate(ctx, sid); !ok {
		t.Fatal("3 笔同额稳流应放（sd=0 LB=mean>0）")
	}
	// 混号流——大笔负笔插入→LB 翻负拦。
	seedFills(t, ctx, r, sid, -0.5, -0.5, -0.5, -0.5, -0.5, -0.5)
	if ok, _ := r.confGate(ctx, sid); ok {
		t.Fatal("混号流 LB≤0 应拦")
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
