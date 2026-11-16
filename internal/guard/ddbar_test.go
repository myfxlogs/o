package guard

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// ddRig 硬线柱节拍假件（O-DD1 工单#20799）：可变权益件+测试时钟驱拍。
// clk 起柱内第 1 秒；feed 逐值拍（同柱内 +1s）；close 推到下一柱起点+1s
// 触发收上一柱。
const ddTestBarSec = 300 // 与测面 risk.dd_bar_seconds 种子同值（柱界对齐用）

type ddRig struct {
	t   *testing.T
	ctx context.Context
	w   *Warden
	clk time.Time
	eq  float64
	err error
}

func newDDRig(t *testing.T, ctx context.Context, w *Warden) *ddRig {
	r := &ddRig{t: t, ctx: ctx, w: w,
		clk: time.Now().Truncate(ddTestBarSec * time.Second).Add(time.Second)}
	w.AttachEquity(func(ctx context.Context) (float64, error) { return r.eq, r.err })
	w.now = func() time.Time { return r.clk }
	return r
}

func (r *ddRig) scan() {
	r.t.Helper()
	r.clk = r.clk.Add(time.Second)
	r.w.ddScan(r.ctx)
}

// feed 逐值令权益件返回该值、clk += 1s、调一次 w.ddScan（同柱内）。
func (r *ddRig) feed(vals ...float64) {
	r.t.Helper()
	for _, v := range vals {
		r.eq = v
		r.scan()
	}
}

// close clk 推到下一柱起点+1s，令权益件返回 v，调一次 w.ddScan——收上一柱。
func (r *ddRig) close(v float64) {
	r.t.Helper()
	r.clk = r.clk.Truncate(ddTestBarSec * time.Second).Add((ddTestBarSec + 1) * time.Second)
	r.eq = v
	r.w.ddScan(r.ctx)
}

func ddBarRows(t *testing.T, ctx context.Context, w *Warden) []float64 {
	t.Helper()
	rows, err := w.pool.Query(ctx, `SELECT equity FROM md_equity_curve
		WHERE src='guard_dd_bar' ORDER BY ts`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		out = append(out, v)
	}
	return out
}

// 产线形种子：fin_0008 迁移种的 risk.kill_switch 是 jsonb 布尔 false（非字符串）。
func TestDDProdBoolSeedTrips(t *testing.T) {
	ctx, w, _ := setup(t)
	if _, err := w.pool.Exec(ctx, "UPDATE app_config SET value='false'::jsonb WHERE key='risk.kill_switch'"); err != nil {
		t.Fatal(err)
	}
	if err := w.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	ddSeed(t, ctx, w, 96, map[string]float64{"'2 days'": 100})
	posID := mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	r := newDDRig(t, ctx, w)
	r.feed(96, 96, 96)
	r.close(96)
	if !killSwitchDB(t, ctx, w) {
		t.Fatal("产线布尔形 kill_switch 下回撤 4% 未触线——硬线失明")
	}
	if w.rej.Snapshot()["guard/dd_hardline_tripped"] != 1 {
		t.Fatalf("dd_hardline_tripped=%v（应恰 1）", w.rej.Snapshot()["guard/dd_hardline_tripped"])
	}
	if closed, _, trig := posClosed(t, ctx, w, posID); !closed || trig != ddTrigger {
		t.Fatalf("在仓未按 dd_30d 平: closed=%v trig=%v", closed, trig)
	}
}

// TestDDResumeAfterTripBoolTrue 触线写库是 jsonb 布尔 true——重载快照后
// 触线态续平照跑（同根因的第二现场：旧读器在触线后同样失明）。
func TestDDResumeAfterTripBoolTrue(t *testing.T) {
	ctx, w, fx := setup(t)
	fx.failFirstN = 1 // 首平败
	ddSeed(t, ctx, w, 96, map[string]float64{"'2 days'": 100})
	posID := mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	r := newDDRig(t, ctx, w)
	r.feed(96, 96, 96)
	r.close(96) // 触线（首平败）
	if len(fx.calls) != 1 {
		t.Fatalf("触发拍 calls=%d", len(fx.calls))
	}
	if err := w.cfg.Load(ctx); err != nil { // 快照现为布尔 true
		t.Fatal(err)
	}
	r.feed(96) // 触线态续平在布尔 true 下照跑
	if len(fx.calls) != 2 {
		t.Fatalf("触线态续平未跑: calls=%d", len(fx.calls))
	}
	if closed, _, _ := posClosed(t, ctx, w, posID); !closed {
		t.Fatal("续平后仓未平")
	}
}

// TestDDTransientDipNoTrip 单采点瞬时深跌（划转/演习毛刺形）被柱中位数吸收。
func TestDDTransientDipNoTrip(t *testing.T) {
	ctx, w, _ := setup(t)
	ddSeed(t, ctx, w, 100, map[string]float64{"'2 days'": 100})
	r := newDDRig(t, ctx, w)
	r.feed(100, 100, 100, 100, 19)
	r.close(100)
	if killSwitchDB(t, ctx, w) {
		t.Fatal("瞬时毛刺误触线")
	}
	rows := ddBarRows(t, ctx, w)
	if len(rows) != 2 || rows[1] != 100 {
		t.Fatalf("柱行=%v（末行应中位数 100 非 19）", rows)
	}
	r.feed(99, 99, 99)
	r.close(99)
	if killSwitchDB(t, ctx, w) {
		t.Fatal("99 柱误触线（dd=1%<3%）")
	}
}

// TestDDTransientSpikeNoPeakPoison 瞬时冲高不污染峰值基线。
func TestDDTransientSpikeNoPeakPoison(t *testing.T) {
	ctx, w, _ := setup(t)
	if _, err := w.pool.Exec(ctx, "DELETE FROM md_equity_curve"); err != nil {
		t.Fatal(err)
	}
	r := newDDRig(t, ctx, w)
	r.feed(100, 100, 100, 100, 200)
	r.close(98)
	r.feed(98, 98)
	r.close(98)
	if killSwitchDB(t, ctx, w) {
		t.Fatal("冲高毛刺误触线")
	}
	rows := ddBarRows(t, ctx, w)
	for _, v := range rows {
		if v > 100 {
			t.Fatalf("峰值被毛刺污染: 柱行=%v", rows)
		}
	}
	var mx float64
	if err := w.pool.QueryRow(ctx, `SELECT COALESCE(max(equity),0) FROM md_equity_curve WHERE src='guard_dd_bar'`).Scan(&mx); err != nil || mx != 100 {
		t.Fatalf("柱行 max=%v（应 100，无 200 行）", mx)
	}
}

// TestDDSustainedBreachTripsAtBarClose 持续破线只在收柱时判撤——柱内不动作。
func TestDDSustainedBreachTripsAtBarClose(t *testing.T) {
	ctx, w, fx := setup(t)
	ddSeed(t, ctx, w, 96, map[string]float64{"'2 days'": 100})
	mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	r := newDDRig(t, ctx, w)
	r.feed(96, 96, 96)
	if killSwitchDB(t, ctx, w) || len(fx.calls) != 0 {
		t.Fatalf("柱内误判撤: kill=%v calls=%d", killSwitchDB(t, ctx, w), len(fx.calls))
	}
	r.close(96)
	if !killSwitchDB(t, ctx, w) {
		t.Fatal("收柱持续破线未触")
	}
	if len(fx.calls) != 1 {
		t.Fatalf("触线未平仓: calls=%d", len(fx.calls))
	}
}

// TestDDThinBarNoJudge 稀柱（样本<ddBarMinSamples）不落行不判。
func TestDDThinBarNoJudge(t *testing.T) {
	ctx, w, _ := setup(t)
	ddSeed(t, ctx, w, 90, map[string]float64{"'2 days'": 100})
	r := newDDRig(t, ctx, w)
	r.feed(90, 90)
	r.close(100)
	if killSwitchDB(t, ctx, w) {
		t.Fatal("稀柱误触线")
	}
	if w.rej.Snapshot()["guard/dd_bar_thin"] != 1 {
		t.Fatalf("dd_bar_thin=%v（应恰 1）", w.rej.Snapshot()["guard/dd_bar_thin"])
	}
	rows := ddBarRows(t, ctx, w)
	for _, v := range rows {
		if v == 90 {
			t.Fatalf("稀柱落行了: %v", rows)
		}
	}
}

// TestDDBlindOnBadKillSwitch kill_switch 非法形=失明告警（计数+出声，不静默）。
func TestDDBlindOnBadKillSwitch(t *testing.T) {
	ctx, w, _ := setup(t)
	if _, err := w.pool.Exec(ctx, "UPDATE app_config SET value='5'::jsonb WHERE key='risk.kill_switch'"); err != nil {
		t.Fatal(err)
	}
	if err := w.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	fs := attachFakeAlerter(w)
	ddSeed(t, ctx, w, 90, nil)
	r := newDDRig(t, ctx, w)
	r.feed(90, 90, 90)
	r.close(90)
	if rows := ddBarRows(t, ctx, w); len(rows) != 0 {
		t.Fatalf("失明目不应落柱行: %v", rows)
	}
	if w.rej.Snapshot()["guard/dd_missing_param"] == 0 {
		t.Fatal("dd_missing_param 未计")
	}
	if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "净值硬线失明") {
		t.Fatalf("失明告警=%v（应恰 1 条含「净值硬线失明」）", fs.sent)
	}
}

// TestDDBlindOnEquityOutage 权益连续读败满一柱长=失明告警；瞬时读败不报。
func TestDDBlindOnEquityOutage(t *testing.T) {
	ctx, w, _ := setup(t)
	if _, err := w.pool.Exec(ctx, "DELETE FROM md_equity_curve"); err != nil {
		t.Fatal(err)
	}
	fs := attachFakeAlerter(w)
	r := newDDRig(t, ctx, w)
	r.err = errors.New("venue read down")
	r.scan() // 第一拍：瞬时读败不告警
	if len(fs.sent) != 0 {
		t.Fatalf("瞬时读败误告警: %v", fs.sent)
	}
	r.clk = r.clk.Add((ddTestBarSec + 1) * time.Second)
	r.scan() // 连续读败满一柱长
	if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "净权益连续读败") {
		t.Fatalf("失明告警=%v（应恰 1 条含「净权益连续读败」）", fs.sent)
	}
	if rows := ddBarRows(t, ctx, w); len(rows) != 0 {
		t.Fatalf("读败面不应落柱行: %v", rows)
	}
}

// TestDDLegacyRowsIgnored 历史非柱行（guard_dd/test 等 src）一律不入峰值。
func TestDDLegacyRowsIgnored(t *testing.T) {
	ctx, w, _ := setup(t)
	ddSeed(t, ctx, w, 99, map[string]float64{"'2 days'": 100})
	for _, src := range []string{"guard_dd", "test"} {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO md_equity_curve(ts,venue,ccy,equity,src) VALUES(now()-interval '1 day','okx','USDT',1000,$1)`, src); err != nil {
			t.Fatal(err)
		}
	}
	r := newDDRig(t, ctx, w)
	r.feed(99, 99, 99)
	r.close(99)
	if killSwitchDB(t, ctx, w) {
		t.Fatal("历史非柱行被计入峰值（1000 基线→误触）")
	}
}

// TestDDNoEquityNoAction 未挂权益件=非硬线岗（simdrill Warden 形）——
// 不续平、不告警、零柱行（前置判必须在一切之前）。
func TestDDNoEquityNoAction(t *testing.T) {
	ctx, w, fx := setup(t)
	// setup 不挂权益件——w.equity==nil 即 simdrill 跨包 Warden 形态。
	if _, err := w.pool.Exec(ctx, "UPDATE app_config SET value='true'::jsonb WHERE key='risk.kill_switch'"); err != nil {
		t.Fatal(err)
	}
	if err := w.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	fs := attachFakeAlerter(w)
	if _, err := w.pool.Exec(ctx, "DELETE FROM md_equity_curve"); err != nil {
		t.Fatal(err)
	}
	mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	w.ddScan(ctx)
	if len(fx.calls) != 0 {
		t.Fatalf("无权益件面不得平仓: calls=%d", len(fx.calls))
	}
	if len(fs.sent) != 0 {
		t.Fatalf("无权益件面不得告警: %v", fs.sent)
	}
	if rows := ddBarRows(t, ctx, w); len(rows) != 0 {
		t.Fatalf("无权益件面不应落柱行: %v", rows)
	}
}

// TestDDResumeEvenIfEquityFails 触线态续平不依赖权益读面——熔断已落后交易所
// 读面出问题时仓照平（复审 R1：权益读败不得挡续平）。
func TestDDResumeEvenIfEquityFails(t *testing.T) {
	ctx, w, fx := setup(t)
	fx.failFirstN = 1
	ddSeed(t, ctx, w, 96, map[string]float64{"'2 days'": 100})
	posID := mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	r := newDDRig(t, ctx, w)
	r.feed(96, 96, 96)
	r.close(96) // 触线，首平败
	if len(fx.calls) != 1 {
		t.Fatalf("触发拍 calls=%d", len(fx.calls))
	}
	if err := w.cfg.Load(ctx); err != nil { // 快照翻闸
		t.Fatal(err)
	}
	r.err = errors.New("venue read down")
	r.scan()
	if len(fx.calls) != 2 {
		t.Fatalf("权益读败下触线态未续平: calls=%d", len(fx.calls))
	}
	if closed, _, _ := posClosed(t, ctx, w, posID); !closed {
		t.Fatal("权益读败下仓未平")
	}
}

// TestDDBlindOnInvalidParams 判据参数越界合法域=失明告警（复审 R2——防静默
// 停判：阈写成 3=永不触发、窗<1d=窗口空集、柱长天文=永不收柱，都必须出声）。
func TestDDBlindOnInvalidParams(t *testing.T) {
	cases := []struct{ key, val string }{
		{"risk.max_dd_30d", "0"},
		{"risk.max_dd_30d", "3"},
		{"risk.dd_window_days", "0.5"},
		{"risk.dd_bar_seconds", "5"},
		{"risk.dd_bar_seconds", "100000"},
	}
	for _, c := range cases {
		t.Run(c.key+"="+c.val, func(t *testing.T) {
			ctx, w, fx := setup(t)
			if _, err := w.pool.Exec(ctx,
				"UPDATE app_config SET value=to_jsonb($2::text) WHERE key=$1", c.key, c.val); err != nil {
				t.Fatal(err)
			}
			if err := w.cfg.Load(ctx); err != nil {
				t.Fatal(err)
			}
			fs := attachFakeAlerter(w)
			ddSeed(t, ctx, w, 90, map[string]float64{"'2 days'": 100})
			mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
			r := newDDRig(t, ctx, w)
			r.feed(90, 90, 90)
			r.close(90)
			if killSwitchDB(t, ctx, w) {
				t.Fatal("非法判据面误落闸")
			}
			if len(fx.calls) != 0 {
				t.Fatalf("非法判据面误平仓: %v", fx.calls)
			}
			var n int
			if err := w.pool.QueryRow(ctx, "SELECT count(*) FROM md_equity_curve").Scan(&n); err != nil || n != 1 {
				t.Fatalf("柱行总数=%d（应恰 1=播种历史柱）", n)
			}
			if w.rej.Snapshot()["guard/dd_missing_param"] == 0 {
				t.Fatal("dd_missing_param 未计")
			}
			if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "净值硬线失明") {
				t.Fatalf("失明告警=%v（应恰 1 条含「净值硬线失明」）", fs.sent)
			}
		})
	}
}
