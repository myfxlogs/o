package hunger

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// O-39480 细胞B 桩测（oppCensus 机会面普查眼）：
// ①funding+借率夹具三形——可执行/结构死/净负各落 opp_map 判定；
// ②同机制同 venue spread 1bp vs 200bp——宽 spread 格判净负不与窄格同判；
// ③eval 臂已评格不升件；④拍距窗内 Quiet 不重跑+抑复窗防重升；
// ⑤opp_map 幂等 upsert（重拍行数不增）；⑥阈件缺 fail-closed；
// ⑦basis 腿（FUT×perp 基差年化）可执行判定落。

func censusFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
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
	s := New(pool, cfg)
	s.censusFeed = func(context.Context) (*CensusFacts, error) {
		return &CensusFacts{
			SpotLive: map[string]bool{
				"AAA-USDT": true, "HHH-USDT": true, "JKL-USDT": true,
				"DDD-USDT": true, "GGG-USDT": true, "III-USDT": true, "JJJ-USDT": true,
			},
			BorrowRate: map[string]float64{
				"EEE": 0.0004, // 小时借率 0.04%→年化~350%吞光负费率租
			},
		}, nil
	}
	return s, pool
}

// seedCensusInst 普查夹具三件套：md_funding 费率行+md_ticks 价签+
// opp_map spread 样本续存（samples=nil=零样本→spread 不可判）。
func seedCensusInst(t *testing.T, pool *pgxpool.Pool, swap, spot string,
	rate float64, last, spreadBps float64, samples []float64) {
	t.Helper()
	ctx := context.Background()
	ts := time.Now().Add(-8 * time.Hour)
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_funding(ts,venue,inst_id,funding_rate,next_funding_ts)
		 VALUES($1,'okx',$2,$3,now())`, ts, swap, rate); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{swap, spot} {
		if id == "" {
			continue
		}
		bid := last * (1 - spreadBps/2e4)
		ask := last * (1 + spreadBps/2e4)
		if _, err := pool.Exec(ctx,
			`INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask,recv_ts)
			 VALUES(now(),'okx',$1,$2,$3,$4,now())
			 ON CONFLICT (venue,inst_id) DO UPDATE SET last=$2,bid=$3,ask=$4`,
			id, last, bid, ask); err != nil {
			t.Fatal(err)
		}
	}
	if samples != nil {
		var arr string
		for i, v := range samples {
			if i > 0 {
				arr += ","
			}
			arr += fmt.Sprintf("%v", v)
		}
		for _, id := range []string{swap, spot} {
			if id == "" {
				continue
			}
			if _, err := pool.Exec(ctx,
				`INSERT INTO fin_opp_map(inst_id,mechanism,legs,verdict,evidence)
				 VALUES($1,'funding','{}','unjudgeable',
				   jsonb_build_object('spread_samples',($2::text)::jsonb))`,
				id, "["+arr+"]"); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func oppMapVerdict(t *testing.T, pool *pgxpool.Pool, inst, mech string) (string, *float64) {
	t.Helper()
	var v string
	var net *float64
	if err := pool.QueryRow(context.Background(),
		`SELECT verdict,net_est_bps FROM fin_opp_map WHERE inst_id=$1 AND mechanism=$2`,
		inst, mech).Scan(&v, &net); err != nil {
		t.Fatalf("opp_map 行缺 inst=%s mech=%s: %v", inst, mech, err)
	}
	return v, net
}

func TestCensusThreeVerdicts(t *testing.T) {
	s, pool := censusFixture(t)
	ctx := context.Background()
	// AAA：高正费率+现货可买+narrow spread → executable 且净估>阈→升件。
	seedCensusInst(t, pool, "AAA-USDT-SWAP", "AAA-USDT", 0.002, 100, 1, []float64{1, 1, 1})
	// BBB：高正费率但现货不可买（feed 无）→structurally-dead。
	seedCensusInst(t, pool, "BBB-USDT-SWAP", "BBB-USDT", 0.002, 100, 1, []float64{1, 1, 1})
	// CCC：负费率但 base 不可借→structurally-dead（借币腿缺）。
	seedCensusInst(t, pool, "CCC-USDT-SWAP", "", -0.002, 100, 1, []float64{1, 1, 1})
	// DDD：费率近零→net-negative（无收租面）。
	seedCensusInst(t, pool, "DDD-USDT-SWAP", "DDD-USDT", 0.0000001, 100, 1, []float64{1, 1, 1})
	// EEE：负费率+可借但借率年化~175%吞光→net-negative。
	seedCensusInst(t, pool, "EEE-USDT-SWAP", "", -0.002, 100, 1, []float64{1, 1, 1})
	// FFF：费率事实缺（无 md_funding 行——md_ticks 单边入宇宙）→funding unjudgeable/dead。
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask,recv_ts)
		 VALUES(now(),'okx','FFF-USDT-SWAP',100,99.9,100.1,now())`); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeOppCensus(ctx); err != nil {
		t.Fatal(err)
	}
	if v, net := oppMapVerdict(t, pool, "AAA-USDT-SWAP", "funding"); v != "executable" || net == nil || *net <= 500 {
		t.Fatalf("AAA funding verdict=%s net=%v 期 executable>500", v, net)
	}
	if v, _ := oppMapVerdict(t, pool, "BBB-USDT-SWAP", "funding"); v != "structurally-dead" {
		t.Fatalf("BBB funding verdict=%s 期 structurally-dead", v)
	}
	if v, _ := oppMapVerdict(t, pool, "CCC-USDT-SWAP", "funding"); v != "structurally-dead" {
		t.Fatalf("CCC funding verdict=%s 期 structurally-dead", v)
	}
	if v, _ := oppMapVerdict(t, pool, "DDD-USDT-SWAP", "funding"); v != "net-negative" {
		t.Fatalf("DDD funding verdict=%s 期 net-negative", v)
	}
	if v, _ := oppMapVerdict(t, pool, "EEE-USDT-SWAP", "funding"); v != "net-negative" {
		t.Fatalf("EEE funding verdict=%s 期 net-negative", v)
	}
	// 升件：AAA 族格 executable∧net>阈∧eval 未评→升（SWAP 行+现货行同机会
	// 双 scope 如实各升——scope=inst:mechanism 键面）。
	var esc int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='升件' AND esc_class=$1`,
		censusEscClass).Scan(&esc); err != nil {
		t.Fatal(err)
	}
	if esc < 1 {
		t.Fatal("AAA 可执行格应升件")
	}
	// FFF：funding 集无此 perp→perp_exists=false→structurally-dead。
	if v, _ := oppMapVerdict(t, pool, "FFF-USDT-SWAP", "funding"); v != "structurally-dead" {
		t.Fatalf("FFF funding verdict=%s 期 structurally-dead", v)
	}
}

func TestCensusSpreadDiscriminates(t *testing.T) {
	s, pool := censusFixture(t)
	ctx := context.Background()
	// HHH：窄 spread 1bp+费率 0.00002/8h（apy≈219bps）→executable（净估>0
	// 但<500 阈——判定与升件闸分层验）。
	seedCensusInst(t, pool, "HHH-USDT-SWAP", "HHH-USDT", 0.00002, 100, 1, []float64{1, 1, 1})
	// GGG：同费率同 venue 但 spread 200bp×2 腿≈400bps>apy219→net-negative。
	seedCensusInst(t, pool, "GGG-USDT-SWAP", "GGG-USDT", 0.00002, 100, 200, []float64{200, 200, 200})
	if err := s.ProbeOppCensus(ctx); err != nil {
		t.Fatal(err)
	}
	if v, _ := oppMapVerdict(t, pool, "HHH-USDT-SWAP", "funding"); v != "executable" {
		t.Fatalf("HHH funding verdict=%s 期 executable（窄 spread）", v)
	}
	if v, _ := oppMapVerdict(t, pool, "GGG-USDT-SWAP", "funding"); v != "net-negative" {
		t.Fatalf("GGG funding verdict=%s 期 net-negative（宽 spread 吞光）", v)
	}
}

func TestCensusSpreadUnjudgeable(t *testing.T) {
	s, pool := censusFixture(t)
	ctx := context.Background()
	// III：高费率但零 spread 样本→unjudgeable（不足 N 不虚构）。
	seedCensusInst(t, pool, "III-USDT-SWAP", "III-USDT", 0.002, 100, 1, nil)
	if err := s.ProbeOppCensus(ctx); err != nil {
		t.Fatal(err)
	}
	if v, _ := oppMapVerdict(t, pool, "III-USDT-SWAP", "funding"); v != "unjudgeable" {
		t.Fatalf("III funding verdict=%s 期 unjudgeable", v)
	}
}

func TestCensusEvalSuppressed(t *testing.T) {
	s, pool := censusFixture(t)
	ctx := context.Background()
	seedCensusInst(t, pool, "JJJ-USDT-SWAP", "JJJ-USDT", 0.002, 100, 1, []float64{1, 1, 1})
	// eval 臂已评（funding-hedge 模板命中）→不升件。
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		 VALUES('JJJ-USDT-SWAP','funding-hedge','short_perp_long_spot','rejected','{}','{}')`); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeOppCensus(ctx); err != nil {
		t.Fatal(err)
	}
	if v, _ := oppMapVerdict(t, pool, "JJJ-USDT-SWAP", "funding"); v != "executable" {
		t.Fatalf("JJJ funding verdict=%s 期 executable（判定不受 eval 抑）", v)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='升件' AND esc_class=$1
		 AND esc_scope LIKE 'JJJ%'`, censusEscClass).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("eval 已评格不应升件 n=%d", n)
	}
}

func TestCensusBasisExecutable(t *testing.T) {
	s, pool := censusFixture(t)
	ctx := context.Background()
	// JKL：perp 100 + FUT 105（到期 ~30d）→基差 500bps 年化~6083bps→executable。
	seedCensusInst(t, pool, "JKL-USDT-SWAP", "JKL-USDT", 0.0001, 100, 1, []float64{1, 1, 1})
	fut := "JKL-USD-261225"
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask,recv_ts)
		 VALUES(now(),'okx',$1,105,104.9,105.1,now())`, fut); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_opp_map(inst_id,mechanism,legs,verdict,evidence)
		 VALUES($1,'basis','{}','unjudgeable',
		   jsonb_build_object('spread_samples','[1,1,1]'::jsonb))`, fut); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeOppCensus(ctx); err != nil {
		t.Fatal(err)
	}
	v, net := oppMapVerdict(t, pool, "JKL-USDT-SWAP", "basis")
	if v != "executable" || net == nil || *net <= 500 {
		t.Fatalf("JKL basis verdict=%s net=%v 期 executable>500", v, net)
	}
}

func TestCensusIdempotentAndGate(t *testing.T) {
	s, pool := censusFixture(t)
	ctx := context.Background()
	seedCensusInst(t, pool, "AAA-USDT-SWAP", "AAA-USDT", 0.002, 100, 1, []float64{1, 1, 1})
	if err := s.ProbeOppCensus(ctx); err != nil {
		t.Fatal(err)
	}
	var n1 int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_map`).Scan(&n1); err != nil {
		t.Fatal(err)
	}
	// 窗内重拍=Quiet（拍距门——不重跑不写）。
	if err := s.ProbeOppCensus(ctx); err != nil {
		t.Fatal(err)
	}
	var marks int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event=$1`, censusEventMarker).Scan(&marks); err != nil {
		t.Fatal(err)
	}
	if marks != 1 {
		t.Fatalf("拍距门失效——标记行=%d 期 1", marks)
	}
	// 越窗重拍：行数不增（幂等 upsert）+升件抑复不重复。
	s.now = func() time.Time { return time.Now().Add(7 * time.Hour) }
	if err := s.ProbeOppCensus(ctx); err != nil {
		t.Fatal(err)
	}
	var n2 int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_map`).Scan(&n2); err != nil {
		t.Fatal(err)
	}
	if n1 != n2 {
		t.Fatalf("幂等 upsert 败——行数 %d→%d", n1, n2)
	}
	var esc int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='升件' AND esc_class=$1`,
		censusEscClass).Scan(&esc); err != nil {
		t.Fatal(err)
	}
	if esc != 2 { // SWAP+现货两 scope 首拍各一；越窗重拍 dedupH=24h 内抑复=仍 2
		t.Fatalf("升件数=%d 期 2（抑复窗内不重复）", esc)
	}
}

func TestCensusCfgFailClosed(t *testing.T) {
	s, pool := censusFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`DELETE FROM app_config WHERE key='explore.census_ivl_h'`); err != nil {
		t.Fatal(err)
	}
	if err := s.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeOppCensus(ctx); err == nil {
		t.Fatal("拍距键缺应 fail-closed 报错")
	}
}
