// audit_1110_drill_test.go —— 审计/演习面 RED 锚（audit/1110-drill）：
// 断言正确语义——修复落库后自然转绿，门转正为回归锚。
// 显跑：AUDIT_RED=1 go test ./internal/trading/simdrill
package simdrill

import (
	"os"
	"strings"
	"testing"

	"o/internal/platform/excell"
)

func auditRedGate(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit 等价锚显跑")
	}
}

// ① xvenue-carry 单策略径形状拒载：Drill.Run 按 inst 后缀派路（非
// legShapeOf 模板派路）——xvenue-carry 策略 inst=*-SWAP 落 SWAP 支跑
// okx 现货多+合约空两腿=错腿形产 funding 绿凭据，execDrillPass 按
// strategy_ref 消费放行（产线 xvenue 策略 paper→live 可持错域凭据毕业）。
// 正确语义=显式拒载（quarter_basis_short 同律——不构错腿不产凭据）。
func TestAuditDrillRunXvenueShapeRefuse(t *testing.T) {
	ctx, d := setup(t)
	// venue 册装 BTC 对——让错形径物理可跑（否则早败乐器不在册,遮蔽真病灶）。
	d.Venue = VenueForInst("BTC-USDT-SWAP")
	stratID := mkFamilyStrat(t, ctx, d, "BTC-USDT-SWAP", "xvenue-carry", 50)
	res, err := d.Run(ctx, stratID)
	if err == nil {
		t.Fatalf("xvenue-carry 单钻应显式拒载（harness 无 mt5 sim 缝——错腿凭据毒化毕业闸）: 实 verdict=%s detail=%s", res.Verdict, res.Detail)
	}
	if !strings.Contains(err.Error(), "未载") && !strings.Contains(err.Error(), "不支持") {
		t.Fatalf("拒词应载明形状未载语义: %v", err)
	}
	// 拒载亦禁留凭据行（凭据即闸绿件——错形 pass 比缺凭据更毒）。
	var n int
	_ = d.Pool.QueryRow(ctx, `SELECT count(*) FROM fin_exec_drill WHERE strategy_ref=$1`, stratID).Scan(&n)
	if n > 0 {
		t.Fatalf("拒载策略不应产凭据: rows=%d", n)
	}
}

// ② dcd 产品夹具 venue='sim' 行遮蔽产线事实行：seedDcdProduct 以真实
// product_id 播种（占用格 inst=产线策略实品——2026-11-10 05:41 产线实证
// dcd_put 格钻 BTC-USDT-261204-66000-P=live 策略#287249 同品），而产线
// 读面 paper.go:261/gradgate.go:166/dcdexec.go:67 全 product_id 键
// ORDER BY ts DESC LIMIT 1 无 venue 谓词——夹具行 ts 恒新遮蔽真行：
// annualized_yield/expiry NULL→apr=0/零时→legOK=false 断 paper 记账流。
// 正确语义=产线读面只见产线行（venue 谓词或夹具名空间隔离）。
func TestAuditSimFixtureShadowsDcdProduct(t *testing.T) {
	ctx, m := setupMatrix(t)
	pid := "BTC-USDT-261204-66000-P"
	// 产线产品事实行（venue=okx——采集器实径；apr/expiry 有效值）。
	if _, err := m.Pool.Exec(ctx, `INSERT INTO md_dcd_product
		(ts,venue,surface,uly,product_id,direction,annualized_yield,expiry,raw)
		VALUES(now()-interval '1 hour','okx','dcd','BTC-USDT',$1,'P',0.12,
		       now()+interval '30 days','{"notionalCcy":"USDT"}')`, pid); err != nil {
		t.Fatal(err)
	}
	// 演习夹具播种（runDCD 实径——ts=now() 恒新于产线行）。
	if err := m.seedDcdProduct(ctx, pid, "P"); err != nil {
		t.Fatal(err)
	}
	// 产线读面镜像（internal/portfolio/lifecycle/paper.go:261 同构——
	// venue 盲读：product_id 键 ORDER BY ts DESC LIMIT 1）。
	var apr float64
	var venue string
	if err := m.Pool.QueryRow(ctx, `SELECT COALESCE(annualized_yield,0),venue FROM md_dcd_product
		WHERE product_id=$1 ORDER BY ts DESC LIMIT 1`, pid).Scan(&apr, &venue); err != nil {
		t.Fatal(err)
	}
	if venue != "okx" || apr <= 0 {
		t.Fatalf("产线 venue 盲读被夹具行遮蔽: venue=%s apr=%v（实品真行 apr=0.12 在位）——paper 记账 dcd_quote_missing 断流面", venue, apr)
	}
}

// ③ md_ticks venue='sim' 行情锚行遮蔽真 tick：leggedDrill/execTwoLeg 对
// 实器键插 (sim,inst) 锚行（ON CONFLICT (venue,inst_id) 独立行永不退役），
// 产线读面 paper.go:278/basis.go:107 `WHERE inst_id=$1` 无 venue 谓词——
// 夹具行与真行并存时可返回 sim 假价/陈 ts（spot 腿价错=paper fill 错数；
// ts 陈=dcd_spot_stale 断流）。正确语义=venue 盲读不见夹具行。
func TestAuditSimTickShadowsRealTick(t *testing.T) {
	ctx, d := setup(t)
	inst := "BTC-USDT"
	// 产线真 tick（venue=okx——ingest 实径，价 50000 远异夹具确定性价）。
	if _, err := d.Pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'okx',$1,50000)`, inst); err != nil {
		t.Fatal(err)
	}
	// 演习行情锚（leggedDrill:183 实径——venue='sim' 同器键独立行）。
	px := pxOf(inst)
	if _, err := d.Pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'sim',$1,$2) ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`, inst, px); err != nil {
		t.Fatal(err)
	}
	// 产线读面镜像（paper.go:278 修后钦定面——venue<>'sim' 谓词：夹具行
	// 不可见；断言产线域读只见产线行）。
	var venue string
	var last float64
	if err := d.Pool.QueryRow(ctx, `SELECT venue,last FROM md_ticks
		WHERE inst_id=$1 AND venue<>'sim' ORDER BY ts DESC LIMIT 1`, inst).Scan(&venue, &last); err != nil {
		t.Fatal(err)
	}
	if venue != "okx" {
		t.Fatalf("产线 venue 盲读命中夹具行: venue=%s last=%v（真价 50000——paper 现货腿价被假价/陈 tick 毒化）", venue, last)
	}
}

// ④ excell.Enumerate 占用格代表 inst 钦定「首见序定」而查询无 ORDER BY：
// 档位代表源 reps[k][notional] 取首见行——fin_strategy 堆序物理序非钦定序，
// 同格异器行（live BTC vs probe 异器）谁当代表=非确定；异器不可钻即整格
// 凭据红（2026-11-10 产线实证：probe MUU-USDT-SWAP 占格→funding-hedge 格
// 凭据 fail=armed→open 全模板向闸红）。正确语义=代表选取序定（ORDER BY
// 显式序——修后本断言转绿）。
func TestAuditExcellOccupancyOrderBy(t *testing.T) {
	src, err := os.ReadFile("../../platform/excell/excell.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	// 占用双查询（策略/提案）——钦定序定须有 ORDER BY。
	for _, marker := range []string{
		"FROM fin_strategy\n\t\tWHERE state IN ('live','paper','probe','degraded')",
		"WHERE p.state IN ('fin_proposed','fin_armed')",
	} {
		i := strings.Index(body, marker)
		if i < 0 {
			t.Fatalf("占用查询锚未命中（件面漂移）: %s", marker)
		}
		seg := body[i : i+len(marker)+400]
		if !strings.Contains(seg, "ORDER BY") {
			t.Fatalf("占用查询无 ORDER BY——「首见序定」钦定与无序实现相悖（代表 inst 非确定取）: %s", marker[:40])
		}
	}
}

// ⑤ 夹具足迹退役对称：seedDcdProduct/行情锚建的 venue='sim' 行随演习毕
// 退役（fin_strategy 同律——ephemeral 行建/退对称钦定）；今裸留——
// sim 行堆叠+venue 盲读遮蔽双害（②③根因）。正确语义=演习毕 venue='sim'
// 足迹行清零（retireEphemeral 同域对称件）。
func TestAuditSimFixtureRowsRetire(t *testing.T) {
	ctx, m := setupMatrix(t)
	mkOcc(t, ctx, m, "BTC-USDT-261204-66000-P", "dcd_put", "dcd_put", "okx", 50)
	cells, err := excell.Enumerate(ctx, m.Pool)
	if err != nil {
		t.Fatal(err)
	}
	c := excell.IndexByCoord(cells)[excell.CoordKey("dcd_put", "okx", "dcd_put")]
	if c == nil {
		t.Fatal("dcd_put 格未枚举")
	}
	res := m.runCell(ctx, *c, c.Tiers[0], shapeDCD)
	if res.Verdict != "pass" {
		t.Fatalf("格演习应绿（夹具可用前提）: %s", res.Detail)
	}
	// 演习毕=夹具足迹当随 ephemeral 退役域同清。
	for _, q := range []struct{ label, sql string }{
		{"md_dcd_product", `SELECT count(*) FROM md_dcd_product WHERE venue='sim'`},
		{"md_ticks", `SELECT count(*) FROM md_ticks WHERE venue='sim'`},
	} {
		var n int
		if err := m.Pool.QueryRow(ctx, q.sql).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			t.Fatalf("演习毕夹具足迹裸留: %s venue='sim' rows=%d（ephemeral 建/退对称破——venue 盲读遮蔽源）", q.label, n)
		}
	}
}

// ⑥ hedgeSpotKey↔lifecycle.basisSpotLeg 派生律等价锚（维度 A 镜像非单源：
// 两处各写 -USD/-USDG→-USDT 归一——今同律将来可漂；本锚域输入全谱对拍，
// 任一侧改律即红，防静默漂移。GREEN 锚非 RED——记录面非升件件）。
func TestHedgeSpotKeyBasisSpotLegParity(t *testing.T) {
	auditRedGate(t)
	basisRe := datedFutRe
	for _, fut := range []string{
		"BTC-USD-270326", "BTC-USDG-270326", "BTC-USDT-270326",
		"ETH-USD-260627", "SOL-USDG-260626",
	} {
		m := basisRe.FindStringSubmatch(fut)
		if m == nil {
			t.Fatalf("期货键未匹配: %s", fut)
		}
		// 产线派生律镜像（lifecycle.basisSpotLeg 同形——包向禁 import 反读）：
		// -USDG→-USDT 先于 -USD→-USDT，USDT 本位原形保。
		prodBase := m[1]
		switch {
		case strings.HasSuffix(prodBase, "-USDG"):
			prodBase = strings.TrimSuffix(prodBase, "-USDG") + "-USDT"
		case strings.HasSuffix(prodBase, "-USD"):
			prodBase = strings.TrimSuffix(prodBase, "-USD") + "-USDT"
		}
		if got := hedgeSpotKey(m[1]); got != prodBase {
			t.Fatalf("派生律漂移: hedgeSpotKey(%s)=%s vs lifecycle 律=%s（镜像双实现未单源——同漂域）", m[1], got, prodBase)
		}
	}
}
