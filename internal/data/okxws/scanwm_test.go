// scanwm_test.go —— T-0440③a 接线桩测（真实 DB 径）：水位闸随装配自动挂线
// （期望集=注册面派生零写死）+族陈水（recv_ts 龄超阈）升件行落 ops_event
// （谱域缺行 scope=okx:spot）+1/族/日 NOT EXISTS 去重（复拍不重升）。
package okxws

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/scanwm"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// famCols 注册面族标列集（闸期望集派生形——装配同函数）。
func famCols(fams []famReg) []string {
	cols := make([]string, len(fams))
	for i, f := range fams {
		cols[i] = f.col
	}
	return cols
}

// escCount 指定 scope 的谱域缺行升件行数。
func escCount(t *testing.T, pool *pgxpool.Pool, scope string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2`,
		scanwm.ClassMissing, scope).Scan(&n); err != nil {
		t.Fatalf("升件行计数: %v", err)
	}
	return n
}

// escClass 指定 scope 的升件类（存在断言在 escCount 后调用）。
func escClass(t *testing.T, pool *pgxpool.Pool, scope string) string {
	t.Helper()
	var cls string
	if err := pool.QueryRow(context.Background(), `SELECT esc_class FROM ops_event
		WHERE event='升件' AND esc_scope=$1 ORDER BY id DESC LIMIT 1`, scope).Scan(&cls); err != nil {
		t.Fatalf("升件行读: %v", err)
	}
	return cls
}

func TestScanWatermarkWiring(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	p := NewScanPoller(pool, nil, tickersStub(map[string]json.RawMessage{
		"SWAP":    json.RawMessage(`[{"instId":"ZZZ-USDT-SWAP","last":"3","bidPx":"2.9","askPx":"3.1","ts":"1700000000000"}]`),
		"SPOT":    json.RawMessage(`[{"instId":"ZZZ-USDT","last":"2.9","bidPx":"2.8","askPx":"3.0","ts":"1700000001000"}]`),
		"FUTURES": json.RawMessage(`[{"instId":"ZZZ-USD-260327","last":"3.1","bidPx":"3.0","askPx":"3.2","ts":"1700000002000"}]`),
	}))
	// 接线锚：闸随装配自动挂线+期望集=注册面派生（swap/spot/futures——
	// 非闸内写死；新族注册即自动入闸域）。
	if p.WM == nil {
		t.Fatal("水位闸应随装配自动挂线")
	}
	if got := famCols(p.fams); len(got) != 3 || got[0] != "swap" || got[1] != "spot" || got[2] != "futures" {
		t.Fatalf("期望集应=注册面派生: %v", got)
	}
	// 首拍：三族鲜落——零升件。
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := escCount(t, pool, "okx:spot"); n != 0 {
		t.Fatalf("鲜落拍不应升件: %d", n)
	}
	// SPOT 族陈水化（recv_ts 回拨 25min——ts 守卫使后续拍不刷新，病灶形态
	// =族停落水位陈化）。
	if _, err := pool.Exec(ctx, `UPDATE md_tick_scan SET recv_ts=now()-interval '25 min'
		WHERE venue='okx' AND inst_type='spot'`); err != nil {
		t.Fatal(err)
	}
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := escCount(t, pool, "okx:spot"); n != 1 {
		t.Fatalf("族陈水跨阈应升件 1 行: %d", n)
	}
	// 复拍：1/族/日 NOT EXISTS 去重——不重升。
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := escCount(t, pool, "okx:spot"); n != 1 {
		t.Fatalf("复拍应去重不重升: %d", n)
	}
	if cls := escClass(t, pool, "okx:spot"); cls != scanwm.ClassMissing {
		t.Fatalf("升件类应=谱域缺行: %q", cls)
	}
}
