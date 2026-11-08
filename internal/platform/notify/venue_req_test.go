package notify

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 锚：recommended 未推→组文+发+锚；重扫零推；关态落锚不发；boards 空→保底文。
func TestVenueReqPush(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 生产真形 fixture（scout board 六字段 json tag 逐字段对拍——margin 不塞
	// 即「无 margin 字段的 metrics 也出富文」锚直接覆盖）。
	mets := `{"boards":[{"family":"carry","net_apr_mid":0.12,"persist":0.8,"baseline":0.06,"baseline_source":"earn","top_n":5}]}`
	if _, err := pool.Exec(ctx, `UPDATE fin_venue_candidate SET state='recommended',
		last_request_at=now(),metrics=$1::jsonb WHERE venue='mexc'`, mets); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	v := NewVenueReq(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := v.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "请注册 mexc") || !strings.Contains(fs.sent[0], "单所息差") {
		t.Fatalf("组文不符: %+v", fs.sent)
	}
	if !strings.Contains(fs.sent[0], "12") || !strings.Contains(fs.sent[0], "6") {
		t.Fatalf("富文应含实数百分比: %s", fs.sent[0])
	}
	if err := v.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 {
		t.Fatalf("重扫不应重推: %d", len(fs.sent))
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_venue_req_push`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("锚应=1: %d", n)
	}
}

// 保底文：metrics 解析败/空→兜底推送不丢。
func TestVenueReqFallback(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_venue_candidate SET state='recommended',
		last_request_at=now(),metrics='{}'::jsonb WHERE venue='gate'`); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	v := NewVenueReq(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := v.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "请注册 gate") {
		t.Fatalf("保底文未发: %+v", fs.sent)
	}
}
