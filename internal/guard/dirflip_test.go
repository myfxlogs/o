package guard

// dirflip_test.go —— O-41393 dir_signal_flip 谓词校准对抗锚：
// 一拍拒读判死刑→连续 K 拍全拒读判消亡（lifecycle.dir.flip_consec
// 缺省 2）。eval 稀疏事件驱动——最新 K 条全 rejected 方杀，
// proposed/unevaluable 任一在窗=断连续不杀（读不出非消亡证词）。

import (
	"context"
	"testing"
	"time"
)

// mkDirSingleLeg 造方向族单腿仓 fixture（legs.single 契约形——
// dir_perp_mom 真户径：strategy.direction='long_perp'+单腿 sell
// BTC-USDT-SWAP px=100 +新鲜 md_ticks mark=100（frac=0 不触
// stop/take/ttl）+fin_opp_eval 种子行集）。
// evals 形=[(verdict, ts偏移h)]——最新在前由调用方按 ts 排。
func mkDirSingleLeg(t *testing.T, ctx context.Context, w *Warden,
	evals []struct {
		verdict string
		ageH    float64
	}) (posRow, legSet) {
	t.Helper()
	// fin_opp_eval 不在 setup 清场域（守卫测试面零写家）——本夹具自清：
	// 同 template+inst+direction 行=flip 查询域，跨测残留=串判据实证坑。
	if _, err := w.pool.Exec(ctx,
		`DELETE FROM fin_opp_eval WHERE inst_id='BTC-USDT-SWAP' AND template='dir-flip-test'`); err != nil {
		t.Fatal(err)
	}
	var stratID, execID, posID int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,direction,notional_usd,venue)
		 VALUES('BTC-USDT-SWAP','dir-flip-test','live','long_perp',100,'okx') RETURNING id`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state) VALUES($1,'BTC-USDT-SWAP',100,'open') RETURNING id`,
		stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,px,state,sz_unit)
		 VALUES($1,1,'cl-dir-1','okx','sell','BTC-USDT-SWAP',1,100,'filled','contract')`, execID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		 VALUES($1,$2,'BTC-USDT-SWAP',100,$3) RETURNING id`,
		execID, stratID, time.Now()).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	for _, e := range evals {
		// fin_0094 行级不变量：proposed 行 net_apr NOT NULL（违者写败）——
		// rejected/unevaluable 留 NULL 如实缺席态。
		var apr any
		if e.verdict == "proposed" {
			apr = 0.05
		}
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO fin_opp_eval(ts,inst_id,template,direction,verdict,net_apr,legs,evidence)
			 VALUES(now()-make_interval(hours=>$1),'BTC-USDT-SWAP','dir-flip-test','long_perp',$2,$3,'{}'::jsonb,'{}'::jsonb)`,
			e.ageH, e.verdict, apr); err != nil {
			t.Fatal(err)
		}
	}
	// mark=100=entry（frac=0——flip 判据隔离：stop/take/ttl 不抢戏）。
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'okx','BTC-USDT-SWAP',100)`); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_funding(ts,venue,inst_id,funding_rate) VALUES(now(),'okx','BTC-USDT-SWAP',0.0001)`); err != nil {
		t.Fatal(err)
	}
	w.AttachCtVal(func(ctx context.Context, venue, instID string) (float64, bool) {
		return instrCtVal[instID], instrCtVal[instID] != 0
	})
	p := posRow{id: posID, execID: execID, stratRef: stratID,
		inst: "BTC-USDT-SWAP", notional: 100, opened: time.Now(), stratState: "live"}
	return p, w.legs(ctx, execID)
}

// flipKind 直调 evaluate 取触发 kind（判据面隔离——tend/gate 不入本票验收）。
func flipKind(t *testing.T, ctx context.Context, w *Warden, p posRow, ls legSet) string {
	t.Helper()
	return w.evaluate(ctx, p, ls).kind
}

// 对抗 a)：latest2=[rej,rej] → 连续消亡证成→杀（K=2 缺省）。
func TestDirFlipConsecKill(t *testing.T) {
	ctx, w, _ := setup(t)
	p, ls := mkDirSingleLeg(t, ctx, w, []struct {
		verdict string
		ageH    float64
	}{{"rejected", 1}, {"rejected", 30}})
	if k := flipKind(t, ctx, w, p, ls); k != "dir_signal_flip" {
		t.Fatalf("最新2条全rejected应杀 dir_signal_flip，got %q", k)
	}
}

// 对抗 b)：latest2=[rej,prop] → 呼吸非消亡→不杀（断连续）。
func TestDirFlipBreathNotKill(t *testing.T) {
	ctx, w, _ := setup(t)
	p, ls := mkDirSingleLeg(t, ctx, w, []struct {
		verdict string
		ageH    float64
	}{{"rejected", 1}, {"proposed", 30}})
	if k := flipKind(t, ctx, w, p, ls); k != "" {
		t.Fatalf("rejected后接proposed=断连续不杀，got %q", k)
	}
}

// 对抗 c)：latest2=[rej,unevaluable] → 读不出非消亡证词→不杀。
func TestDirFlipUnevaluableNotKill(t *testing.T) {
	ctx, w, _ := setup(t)
	p, ls := mkDirSingleLeg(t, ctx, w, []struct {
		verdict string
		ageH    float64
	}{{"rejected", 1}, {"unevaluable", 30}})
	if k := flipKind(t, ctx, w, p, ls); k != "" {
		t.Fatalf("unevaluable断连续不杀（读不出非证词），got %q", k)
	}
}

// 对抗 d)：仅1行rejected（历史独生行）→ len<K 不杀（单读不构成连续）。
func TestDirFlipSingleRowNotKill(t *testing.T) {
	ctx, w, _ := setup(t)
	p, ls := mkDirSingleLeg(t, ctx, w, []struct {
		verdict string
		ageH    float64
	}{{"rejected", 1}})
	if k := flipKind(t, ctx, w, p, ls); k != "" {
		t.Fatalf("独生rejected行 len=1<K=2 不杀，got %q", k)
	}
}

// 对抗 e)：flip_consec=1 配面 → 复现旧行为（幂等回退锚——最新1条
// rejected 即杀，LIMIT 1 语义=现行）。
func TestDirFlipConsecOneLegacy(t *testing.T) {
	ctx, w, _ := setup(t)
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO app_config(key,value) VALUES('lifecycle.dir.flip_consec','1')
		 ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	reloadCfg(t, ctx, w)
	p, ls := mkDirSingleLeg(t, ctx, w, []struct {
		verdict string
		ageH    float64
	}{{"rejected", 1}})
	if k := flipKind(t, ctx, w, p, ls); k != "dir_signal_flip" {
		t.Fatalf("flip_consec=1 应复现旧行为杀，got %q", k)
	}
}

// 对抗 f)：零 eval 行→不杀；读败径=err→空 trig 同语义（缺证不虚构）。
func TestDirFlipNoRowsNotKill(t *testing.T) {
	ctx, w, _ := setup(t)
	p, ls := mkDirSingleLeg(t, ctx, w, nil)
	if k := flipKind(t, ctx, w, p, ls); k != "" {
		t.Fatalf("零eval行不杀，got %q", k)
	}
	// 读败径：ctx 取消→dirTrig 首读败→空 trig（evaluate 早退同不杀语义）。
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	if tr := w.dirTrig(cctx, p, ls); tr.kind != "" {
		t.Fatalf("读败应空 trig 不杀，got %q", tr.kind)
	}
}
