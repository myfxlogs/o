package lifecycle

import (
	"context"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0521 摩擦记账/回灌桩测——锚①开户扣点(kind='fee' fill 负额,detail 携 src);
// 锚②回灌：30d 已平仓 fee 聚合→lifecycle.friction_bps.<venue> DMSet 落账
// （薄样<3 不覆写）；锚③滑点样本足→实测替代缓冲；锚④sim 域剔除。

func frictionFixture(t *testing.T) (*Runner, context.Context) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return NewRunner(pool, cfg), ctx
}

func mkPaperStrategy(t *testing.T, ctx context.Context, r *Runner) int64 {
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction,venue)
		VALUES('BTC-USDT-SWAP','funding-hedge','paper',50,'short_perp_long_spot','okx')
		RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

func TestFrictionChargeDefaultRate(t *testing.T) {
	r, ctx := frictionFixture(t)
	sid := mkPaperStrategy(t, ctx, r)
	r.chargePaperFriction(ctx, sid, "okx", "BTC-USDT-SWAP", 50)
	var amt float64
	var det []byte
	if err := r.pool.QueryRow(ctx, `SELECT amount_usd,detail FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='fee'`, sid).Scan(&amt, &det); err != nil {
		t.Fatalf("fee fill 缺: %v", err)
	}
	want := -50 * frictionDefaultBps / 2 / 1e4 // 单程口径=往返率/2
	if amt != want {
		t.Fatalf("兜底率扣点: amt=%.6f want=%.6f", amt, want)
	}
}

func TestFrictionChargeCalibratedRate(t *testing.T) {
	r, ctx := frictionFixture(t)
	if _, err := r.cfg.DMSet(ctx, "lifecycle.friction_bps.okx",
		[]byte("12.5"), "桩测校准值", "T-0521"); err != nil {
		t.Fatal(err)
	}
	if err := r.cfg.Load(ctx); err != nil { // DMSet 直写库——桩内刷缓存（产线 notify 链自热载）
		t.Fatal(err)
	}
	sid := mkPaperStrategy(t, ctx, r)
	r.chargePaperFriction(ctx, sid, "okx", "BTC-USDT-SWAP", 50)
	var amt float64
	if err := r.pool.QueryRow(ctx, `SELECT amount_usd FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='fee'`, sid).Scan(&amt); err != nil {
		t.Fatalf("fee fill 缺: %v", err)
	}
	if want := -50 * 12.5 / 2 / 1e4; amt != want {
		t.Fatalf("校准率扣点: amt=%.6f want=%.6f", amt, want)
	}
}

// mkClosedPosition 已平 live 仓桩景（execution+position+fee 三行——
// 回灌样本单元）。
func mkClosedPosition(t *testing.T, ctx context.Context, r *Runner, notional, fee float64) {
	t.Helper()
	var execID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('BTC-USDT-SWAP',$1,'closed') RETURNING id`, notional).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	var pid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at,closed_at)
		VALUES($1,'BTC-USDT-SWAP',$2,now()-interval '5 days',now()-interval '1 day')
		RETURNING id`, execID, notional).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('fee','okx','USDT',$1,$2,0,'leg',now()-interval '1 day')`, -fee, pid); err != nil {
		t.Fatal(err)
	}
}

func TestFrictionCalibrateMeasuredFee(t *testing.T) {
	r, ctx := frictionFixture(t)
	// 3 已平仓（n≥3 达阈）+fee 行；总额 0.6/6000=1bps 费+5 缓冲=6bps。
	for i := 0; i < 3; i++ {
		mkClosedPosition(t, ctx, r, 2000, 0.2)
	}
	// sim 域行——剔除锚（不计入）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('fee','sim','USDT',-99,999999,0,'simvenue',now())`); err != nil {
		t.Fatal(err)
	}
	if err := r.frictionCalibrate(ctx); err != nil {
		t.Fatal(err)
	}
	var bps float64
	if err := r.pool.QueryRow(ctx, `SELECT value::float FROM app_config
		WHERE key='lifecycle.friction_bps.okx'`).Scan(&bps); err != nil {
		t.Fatalf("校准键未落: %v", err)
	}
	if want := 1.0 + frictionBufferBps; bps != want {
		t.Fatalf("校准率: bps=%.4f want=%.4f（fee=0.6/6000·1e4=1+buf5）", bps, want)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='参数调整'
		AND note LIKE '%friction_bps.okx%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("参数调整账行: n=%d err=%v", n, err)
	}
}

func TestFrictionCalibrateThinSampleNoWrite(t *testing.T) {
	r, ctx := frictionFixture(t)
	mkClosedPosition(t, ctx, r, 2000, 0.2) // 单仓=薄样<3
	if err := r.frictionCalibrate(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM app_config
		WHERE key='lifecycle.friction_bps.okx'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("薄样<3 不该覆写: n=%d err=%v", n, err)
	}
}

func TestFrictionCalibrateSlipMeasured(t *testing.T) {
	r, ctx := frictionFixture(t)
	for i := 0; i < 3; i++ {
		mkClosedPosition(t, ctx, r, 2000, 0.2)
	}
	// 滑点样本 4 行(avg=2bps；mark 鲜度内)+第四仓携 2 filled 腿行供 legs_per_pos=2。
	for i := 0; i < 4; i++ {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_slip_sample(leg_id,venue,inst_id,side,px,mark_px,mark_ts,slip_bps)
			VALUES($1,'okx','BTC-USDT-SWAP','buy',100,99.9,now(),2.0)`, 9000+i); err != nil {
			t.Fatal(err)
		}
	}
	var execID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('BTC-USDT-SWAP',2000,'closed') RETURNING id`).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at,closed_at)
		VALUES($1,'BTC-USDT-SWAP',2000,now()-interval '5 days',now()-interval '1 day')`, execID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state,px)
			VALUES($1,$2,'c','okx','buy','BTC-USDT-SWAP',0.02,'filled',100)`, execID, i+1); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.frictionCalibrate(ctx); err != nil {
		t.Fatal(err)
	}
	var bps float64
	if err := r.pool.QueryRow(ctx, `SELECT value::float FROM app_config
		WHERE key='lifecycle.friction_bps.okx'`).Scan(&bps); err != nil {
		t.Fatal(err)
	}
	// fee=0.6/6000·1e4=1bps；slip 实测=avg2×legs_per_pos 2=4bps→总 5bps
	// （仓均腿数只算 execID 那仓=2；其余三仓 execution_id 虚指不入 join）。
	want := 1.0 + 2.0*2.0
	if bps != want {
		t.Fatalf("滑点实测替代缓冲: bps=%.4f want=%.4f", bps, want)
	}
}
