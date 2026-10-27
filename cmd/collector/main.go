// collector —— 采集面外置开发机蒸馏件（采集器机）（T-0137/D-0020）：dev 跑全宇宙 WS
// （index-tickers/mark-price/funding-rate 全量），蒸馏态 upsert 直写 SG md_* 表
// （SSH 隧道径——写权 collector_rw 收窄件）。键域契约：同键 upsert 最新 ts 赢。
// T-0171 快照价格全面退役：spot_last/mark_price REST 轮询兜底件全撤——
// mark/index 行唯一来源=WS 全量订阅（REST 只准条款/目录注册面）。
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/data/okxws"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/hostmetrics"
	"o/internal/platform/pgguard"
	"o/internal/platform/roster"
	"o/internal/research/sigdist"
	"o/internal/trading/okxexec"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("SG_PG_DSN") // env 白名单件（隧道终点 SG PG——开发机出口）
	if dsn == "" {
		log.Fatal("SG_PG_DSN 缺——桥写径无目的地 fail-closed")
	}
	pcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("SG PG DSN 解析败: %v", err)
	}
	pgguard.GuardPool(pcfg) // T-0154 事务残留守卫（同源普查——自建池同守）
	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		log.Fatalf("SG PG 连接败: %v", err)
	}
	defer pool.Close()
	// T-0169 分域：席位台账归 SG 册（fin_employee_registry）——ROSTER_PG_DSN 钉
	// 隧道径池（D-0030 §五直 beat）；业务写面=本池（PG-md 正本）。
	if rp := os.Getenv("ROSTER_PG_DSN"); rp != "" {
		rpool, err := pgxpool.New(ctx, rp)
		if err != nil {
			log.Fatalf("roster 池连接败: %v", err)
		}
		defer rpool.Close()
		// T-0173 主机资源自报（与席位 beat 同频同径——30s 拍 upsert dev 行）。
		go hostmetrics.Run(ctx, rpool, "dev", "/", 30*time.Second)
		roster.SetRegistryPool(rpool)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		log.Fatalf("config load: %v", err)
	}
	cli := okxexec.NewClient(okxexec.Credentials{}, false) // 公开面零凭据
	c := okxws.NewCollector(pool, cfg)
	c.FullSub = true             // 桥面=全量订阅（不受 SG 瘦身口径）
	c.AlertKind = "bridge_stale" // 面死告警 kind 覆写（T-0137 设计钉）
	c.ExpandSymbols = func(ctx context.Context) ([]string, error) {
		return universe(ctx, cli) // 宇宙扩件同 SG universeExpander 径
	}
	c.AttachCTV(ctvFeed(cli)) // T-0139 depth5 USD 额换算（instruments SWAP ctVal 实读）
	errc := make(chan error, 4)
	go c.Run(ctx, errc)
	sd := sigdist.New(pool)
	sd.AttachConfig(cfg)
	go sd.Run(ctx, errc) // 桥二期信号蒸馏拍（basis_ann/funding_view/xvenue→md_signal）
	// T-0141 MT5 quote-only 桥腿（env 凭据本地注入；缺件=让位 WARN 不拖 OKX 面）
	if bc, err := mt5.NewBridgeCollector(pool, cfg); err != nil {
		log.Printf("[collector] MT5 桥腿缺席（凭据 env 未注入——跨所族停摆非全桥）: %v", err)
	} else {
		go bc.Run(ctx, errc)
	}
	log.Printf("[collector] 采集桥起——全宇宙蒸馏写 SG（共写语义：同键最新 ts 赢）")
	for {
		select {
		case err := <-errc:
			if err != nil {
				log.Fatalf("[collector] 岗终: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// universe 全宇宙标的（instruments 实读——SWAP live+季度期货+基币现货对，
// 与 cmd/core/universe.go 同语义桥面复制件）。
func universe(ctx context.Context, cli *okxexec.Client) ([]string, error) {
	raw, err := cli.PublicGet(ctx, "/api/v5/public/instruments?instType=SWAP")
	if err != nil {
		return nil, err
	}
	var swap []struct {
		InstID string `json:"instId"`
		State  string `json:"state"`
	}
	if err := json.Unmarshal(raw, &swap); err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]bool{}
	for _, i := range swap {
		if i.State == "live" && strings.HasSuffix(i.InstID, "-USDT-SWAP") {
			out = append(out, i.InstID)
			base := strings.TrimSuffix(i.InstID, "-USDT-SWAP") + "-USDT"
			if !seen[base] {
				seen[base] = true
				out = append(out, base)
			}
		}
	}
	raw, err = cli.PublicGet(ctx, "/api/v5/public/instruments?instType=FUTURES")
	if err != nil {
		return out, nil
	}
	var fut []struct {
		InstID string `json:"instId"`
		Alias  string `json:"alias"`
		Uly    string `json:"uly"`
		State  string `json:"state"`
	}
	if err := json.Unmarshal(raw, &fut); err == nil {
		datedRe := regexp.MustCompile(`-\d{6}$`)
		for _, i := range fut {
			// T-0189 返修：quarter 到期期货全收（-\d{6}$ 尾判——OKX 实挂
			// 全 -USD- 币本位，-USDT- 保证金判=空集误诊）；uly 剥离键
			//（BTC-USD-261225→BTC-USD）并入集——index-tickers USD 指数行
			// 覆盖（basisAnn JOIN 键）。
			if i.State == "live" && i.Alias == "quarter" && datedRe.MatchString(i.InstID) {
				out = append(out, i.InstID)
				out = append(out, i.Uly) // T-0189 返修③：uly 载荷字段=钦定指数键（_UM 变体剥串错源根治）
			}
		}
	}
	return out, nil
}

// ctvFeed SWAP 合约面值实读（instruments SWAP——ctVal 币数/张；depth5 USD
// 额=px×sz×ctVal 换算源）。
func ctvFeed(cli *okxexec.Client) func(ctx context.Context) (map[string]float64, error) {
	return func(ctx context.Context) (map[string]float64, error) {
		raw, err := cli.PublicGet(ctx, "/api/v5/public/instruments?instType=SWAP")
		if err != nil {
			return nil, err
		}
		var rows []struct {
			InstID string `json:"instId"`
			CtVal  string `json:"ctVal"`
		}
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, err
		}
		m := map[string]float64{}
		for _, r := range rows {
			v, err := strconv.ParseFloat(r.CtVal, 64)
			if err == nil && v > 0 {
				m[r.InstID] = v
			}
		}
		return m, nil
	}
}
