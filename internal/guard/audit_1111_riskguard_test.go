// audit_1111_riskguard_test.go —— 审计/风控哨戒 RED 锚（audit/1111-riskguard）：
// 断言正确语义——修复落库后自然转绿，门转正为回归锚。
// 显跑：AUDIT_RED=1 go test ./internal/guard
package guard

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func auditRedGate(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit RED 锚——修复期显跑: AUDIT_RED=1 go test -run " + t.Name() + " ./internal/guard；修好后去门转正为回归锚")
	}
}

// ③ 哨戒事实面 venue 盲读：notionalNow 与 bagUSD 的 md_ticks 分支
// WHERE inst_id=$1 无 venue 谓词——演习夹具 venue='sim' 行以真实 inst_id
// 播种（产线 2026-11-10 05:41 实发同款），新陈期内假价进漂移判官→
// 假失衡误杀真仓；bagUSD 同径把假价喂尘估值。risk.dcdExposureUSD 已显滤
// venue<>'sim' 而对拍件未滤=跨包口径漂移。正确语义=产线事实读面显式拒
// 'sim' 域（或 sim 行不与生线 inst_id 共享键空）。
func TestAuditNotionalNowSimVenueBlind(t *testing.T) {
	auditRedGate(t)
	ctx, w, _ := setup(t)
	// ctVal 假件直挂（不引 seedFacts 的 okx tick——保 sim 行为唯一价源）。
	w.AttachCtVal(func(_ctx context.Context, _, _ string) (float64, bool) { return 0.01, true })
	// 仅夹具域有该 inst 的新 tick（无产线行）——守卫不应读到任何价。
	for _, inst := range []string{"BTC-USDT-SWAP", "BTC-USDT"} {
		if _, err := w.pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
			VALUES(now(),'sim',$1,6468)
			ON CONFLICT(venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`, inst); err != nil {
			t.Fatal(err)
		}
	}
	n, ok := w.notionalNow(ctx, leg{inst: "BTC-USDT-SWAP", venue: "okx", sz: 1, szUnit: "contract"})
	if ok {
		t.Fatalf("notionalNow 消费 venue='sim' 夹具价 6468→名义 %v——夹具污染判官径（当 ok=false 无价 fail-closed）", n)
	}
	if v := w.bagUSD(ctx, "BTC", 1); v != 0 {
		t.Fatalf("bagUSD 消费 venue='sim' 夹具价→袋估值 %v——当 0（估值缺席跳尘）", v)
	}
}

// ④ equity-caliber-multi-consumer-drift 判例回潮（KB 钦定）：T-0264 要求
// dd 硬线与 riskGate 共享单源三分量权益闭包（trading+funding+earn 停泊）。
// 现接线 wd.AttachEquity(liveEq) 挂交易户单口径闭包，riskGate.WithEquity
// 三分量——停泊态 trading≈0→dd_no_base 空转；户间划转=假回撤→误触
// 全平+kill_switch（钱径级）。正确语义=两消费点同口径同闭包。
// 免疫面在 cmd/core 接线层，源码级断言两挂点同闭包。
func TestAuditDdEquityCaliberParity(t *testing.T) {
	auditRedGate(t)
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	src, err := os.ReadFile(filepath.Join(root, "cmd/core/venue_readers.go"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	// 单源判据：AttachEquity 挂点必须共享含 fundSaveBalances 的三分量闭包——
	// 钦定修法=okxEq 单闭包两挂点（AttachEquity(okxEq)+WithEquity(okxEq)）。
	attach := strings.Index(s, "wd.AttachEquity(")
	if attach < 0 {
		t.Fatal("AttachEquity 挂点缺席")
	}
	// 挂点闭包名回查其定义体是否含三分量聚合（fundSaveBalances）。
	seg := s[attach:]
	end := strings.Index(seg, ")")
	name := strings.TrimSpace(seg[len("wd.AttachEquity("):end])
	var def string
	for _, mark := range []string{name + " := func", "var " + name + " = func"} {
		if i := strings.Index(s, mark); i >= 0 {
			body := s[i:]
			if j := strings.Index(body, "\n\t}"); j >= 0 { // 闭包收束于一缩进 `}`——禁窗口越界误吃后续 riskGate 闭包
				def = body[:j]
			}
			break
		}
	}
	if !strings.Contains(def, "fundSaveBalances") {
		t.Fatalf("AttachEquity 挂 %s=交易户单口径（无 fundSaveBalances 三分量）——dd 硬线与 riskGate 权益口径漂移（停泊态假回撤/假死基线）", name)
	}
}
