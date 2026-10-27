// ops_face_test.go —— T-0170 桩锚：健康板判态（新鲜=绿/陈=黄红/语义空态）+
// ListReports 直出（metrics 键值对齐）+EquityCurve 日桶取末+ListCashflows
// 筛/页。全走 testutil 独立 schema——不假数据不打生产。
package main

import (
	"context"
	"testing"

	"o/internal/gen/dashboardv1"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func opsSetup(t *testing.T) coreService {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(context.Background(), pool, "md"); err != nil {
		t.Fatal(err)
	}
	return coreService{pool: pool}
}

// 锚①：健康板判态三态——鲜行绿、超阈黄（2min 陈）→红；零行语义空态（
// order_profit 空仓=绿+旁注，basis_ann 零行=黄+立案注）。16 件全集在册。
func TestSystemHealthStates(t *testing.T) {
	c := opsSetup(t)
	ctx := context.Background()
	_, _ = c.pool.Exec(ctx, `TRUNCATE md_mark_index, md_signal, md_account_events`)
	if _, err := c.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,inst_id,channel,px)
		VALUES (now(),'BTC-USDT-SWAP','mark-price',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
		VALUES ('depth5','X','1','{}',now()-interval '70 seconds')`); err != nil {
		t.Fatal(err)
	}
	r, err := c.GetSystemHealth(ctx, &dashboardv1.GetSystemHealthRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Components) != 16 {
		t.Fatalf("组件全集应 16 件: %d", len(r.Components))
	}
	byName := map[string]*dashboardv1.SystemComponent{}
	for _, comp := range r.Components {
		byName[comp.Component] = comp
	}
	if got := byName["okxws WS 行情"]; got.State != 0 {
		t.Fatalf("鲜行应绿: %+v", got)
	}
	if got := byName["买五档 depth5"]; got.State != 1 {
		t.Fatalf("5min 陈 depth5 应黄（阈 30s）: %+v", got)
	}
	if got := byName["蒸馏 funding_view"]; got.State != 2 || got.Note == "" {
		t.Fatalf("零行非语义面应红+注: %+v", got)
	}
	if got := byName["MT5 利润流"]; got.State != 0 || got.Note == "" {
		t.Fatalf("order_profit 零行=空仓语义应绿+注: %+v", got)
	}
	if got := byName["蒸馏 basis_ann"]; got.State != 1 || got.Note == "" {
		t.Fatalf("basis_ann 零行=已知缺席应黄+注: %+v", got)
	}
}

// 锚②：ListReports metrics jsonb→键值对（键序稳定）+period 缺省/n 界。
func TestListReports(t *testing.T) {
	c := opsSetup(t)
	ctx := context.Background()
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_report(period,period_start,period_end,metrics,note,generated_at)
		VALUES ('daily',date_trunc('day',now()),now(),
		'{"net":-1.5,"cum_nav":100.25,"z_last":7}'::jsonb,'试',now())`); err != nil {
		t.Fatal(err)
	}
	r, err := c.ListReports(ctx, &dashboardv1.ListReportsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Reports) != 1 {
		t.Fatalf("应 1 行: %d", len(r.Reports))
	}
	row := r.Reports[0]
	if len(row.Metrics) != 3 || row.Metrics[0].K != "cum_nav" || row.Metrics[2].K != "z_last" {
		t.Fatalf("metrics 键序/值不对: %+v", row.Metrics)
	}
	if row.Metrics[0].V != 100.25 || row.Note != "试" {
		t.Fatalf("metric 值/注不对: %+v note=%s", row.Metrics[0], row.Note)
	}
	// 未知 period=空集如实
	r2, err := c.ListReports(ctx, &dashboardv1.ListReportsRequest{Period: "weekly"})
	if err != nil || len(r2.Reports) != 0 {
		t.Fatalf("weekly 应空集: %d %v", len(r2.Reports), err)
	}
}

// 锚③：EquityCurve 日桶取末——同日两拍取末值，异日分行；days 缺省。
func TestEquityCurveBuckets(t *testing.T) {
	c := opsSetup(t)
	ctx := context.Background()
	_, _ = c.pool.Exec(ctx, `DELETE FROM md_equity_curve WHERE ccy='NAV'`)
	_, _ = c.pool.Exec(ctx, `INSERT INTO md_equity_curve(ts,venue,ccy,equity,src) VALUES
		(now()-interval '2 days','ledger','NAV',100,'report_daily'),
		(now()-interval '1 days','ledger','NAV',110,'report_daily'),
		(now()-interval '1 days','ledger','NAV',119,'report_daily'),
		(now(),'ledger','NAV',120,'report_daily')`)
	r, err := c.GetEquityCurve(ctx, &dashboardv1.GetEquityCurveRequest{Days: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Points) != 3 {
		t.Fatalf("应 3 日桶: %d", len(r.Points))
	}
	if r.Points[1].Nav != 119 || r.Points[2].Nav != 120 {
		t.Fatalf("日桶应取末值: %+v", r.Points)
	}
}

// 锚④：ListCashflows 筛+分页+total 下推。
func TestListCashflows(t *testing.T) {
	c := opsSetup(t)
	ctx := context.Background()
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,ccy,amount,venue,occurred_at,source_ref,source_table) VALUES
		('interest','USDT',1,'okx',now(),1,'test'),('fee','USDT',-0.5,'okx',now(),2,'test'),
		('interest','USDT',2,'okx',now()-interval '1 hour',3,'test')`); err != nil {
		t.Fatal(err)
	}
	r, err := c.ListCashflows(ctx, &dashboardv1.ListCashflowsRequest{Kind: "interest", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r.Total != 2 || len(r.Rows) != 1 {
		t.Fatalf("筛 interest 应 total=2 页=1: %+v", r)
	}
	if r.Rows[0].Amount != 2 { // 最新优先
		t.Fatalf("应最新优先: %+v", r.Rows[0])
	}
	ra, _ := c.ListCashflows(ctx, &dashboardv1.ListCashflowsRequest{Limit: 10})
	if ra.Total != 3 || len(ra.Rows) != 3 {
		t.Fatalf("全量应 3: %+v", ra)
	}
}

// 锚⑤：主机资源——两行直出+stale 判（>2min 陈=stale 显形）。
func TestHostMetrics(t *testing.T) {
	c := opsSetup(t)
	ctx := context.Background()
	_, err := c.pool.Exec(ctx, `INSERT INTO fin_host_metrics
		(host,ts,cores,cpu_pct,mem_total_mb,mem_used_mb,disk_total_gb,disk_used_gb,net_rx_kbps,net_tx_kbps,uptime_s)
		VALUES ('dev',now(),12,33.5,32768,16384,500,120,800,400,86400),
		       ('sg',now()-interval '10 minutes',4,55.0,7680,6144,80,60,50,30,172800)`)
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.GetHostMetrics(ctx, &dashboardv1.GetHostMetricsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Hosts) != 2 {
		t.Fatalf("应 2 主机: %d", len(r.Hosts))
	}
	byHost := map[string]*dashboardv1.HostMetric{}
	for _, hm := range r.Hosts {
		byHost[hm.Host] = hm
	}
	dev := byHost["dev"]
	if dev.Stale || dev.CpuPct != 33.5 || dev.Cores != 12 {
		t.Fatalf("dev 行不对: %+v", dev)
	}
	sg := byHost["sg"]
	if !sg.Stale {
		t.Fatalf("10min 前的 sg 拍应 stale: %+v", sg)
	}
	if sg.UptimeHours != 48 {
		t.Fatalf("uptime 折时不符: %v", sg.UptimeHours)
	}
}
