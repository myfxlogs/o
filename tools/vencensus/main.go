// tools/vencensus —— venue 能力面普查薄件（工单#25843，研究员席配尺）。
// 扫 venue API 面 × 对拍 canon 已登记能力（skills/* 能力账 + 码面调用点）
// → 未用能力差集。产出：docs/vencensus/census_gap.md 幂等快照（重跑同
// 文件 diff 出新增/消失项）；-post 时按族落 ops_event 记事行（seat='研究'，
// 键 vencensus:<venue>:<family> 抑重——重跑不重报）。
//
// 源面（禁凭记忆列——可复拉）：
//
//	OKX    = docs/vencensus/okx_v5_endpoints.tsv（-pull-okx 重拉官方
//	         docs-v5/zh 全量页重建，端点从页体 /api/v5/* 字面量抽取）
//	MT4/5  = mtapi proto rpc 面（-mt4proto/-mt5proto；仓内 mt5 副本与权威源
//	         /opt/grpc-proto/mt5/protos/mt5.proto 对拍纪律 D-604 在案）
//	binance= 码面已接端点（fapi/sapi 字面量——无官方镜像源，已接面内
//	         未登记同律）
//
// 登记基线 = skills/*.md canon 全文 + internal/cmd/tools 非 gen 码面
// 调用点字面量（internal/gen/web gen 剔除——桩名≠用）。
//
// 用法：
//
//	go run ./tools/vencensus                      # 普查→快照（幂等，默认）
//	go run ./tools/vencensus -pull-okx            # 重拉官方页重建端点册
//	DB_DSN=postgres://... go run ./tools/vencensus -post   # 差集族行落 ops_event
//
// 选型：薄工具人拉（hunger 拍挂载非本票域——禁线禁建常驻件，人拉/挂拍
// 归研究员席消费节奏）。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type item struct {
	Face string // 端点路径或 RPC 名
	Fam  string // 族（路径第二段 / rpc 全名）
}

var okxUse = map[string]string{
	"tradingBot":  "venue 原生 bot 容器（网格/DCD/信号/定投/搬砖）——省自建撮合与格账（工单#25840 网格适配器在案）",
	"finance":     "理财/借贷/链上赚币停泊面——savings 已接一脉，余族未探",
	"copytrading": "跟单/带单面——信号分发 venue 代管径",
	"rfq":         "大宗询价面——多腿组合 RFQ 价优径",
	"sprd":        "价差组合原子下单面——多腿同撮防瘸腿",
	"rubik":       "官方统计/多空比/期权与大数据面——ctx 源补",
	"users":       "子账户族管理面",
	"affiliate":   "代理返佣面（弱相关）",
	"fiat":        "法币出入金面（红线外）",
	"support":     "客服/公告面",
	"system":      "系统状态面",
	"account":     "账户配置/仓位/费级面——已接部分，余未探",
	"asset":       "资金户/划转/账单面——已接部分，余未探",
	"trade":       "交易单面——已接部分（algo/precheck 等未探）",
	"public":      "公开品种/标记价/资金费面——已接部分，余未探",
	"market":      "行情面——已接部分（books-full/option 等未探）",
}

func use(venue, fam string) string {
	if venue == "okx" {
		if u, ok := okxUse[fam]; ok {
			return u
		}
	}
	return "未评估——研究员席判"
}

// ---- 源面装载 ----

func loadTSV(path string) ([]item, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []item
	for _, ln := range strings.Split(string(b), "\n") {
		f := strings.Split(strings.TrimSpace(ln), "\t")
		if len(f) >= 2 && f[0] != "" {
			out = append(out, item{Face: f[0], Fam: f[1]})
		}
	}
	return out, nil
}

var rpcRe = regexp.MustCompile(`(?m)^\s*rpc\s+(\w+)\s*\(`)

func rpcItems(path string) ([]item, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []item
	for _, m := range rpcRe.FindAllSubmatch(b, -1) {
		out = append(out, item{Face: string(m[1]), Fam: "mtapi-rpc"})
	}
	return out, nil
}

// ---- 登记语料（skills canon + 非 gen 码面）----

func corpus(root string) (map[string][]byte, error) {
	canon := map[string][]byte{} // "canon"=skills/*.md ；"code"=go 源
	for _, spec := range []struct {
		dir, key string
		ext      string
	}{
		{"skills", "canon", ".md"},
		{"internal", "code", ".go"},
		{"cmd", "code", ".go"},
		{"tools", "code", ".go"},
	} {
		base := filepath.Join(root, spec.dir)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, spec.ext) {
				return err
			}
			// gen 桩名≠用——生成层剔语料
			if strings.Contains(p, string(filepath.Separator)+"gen"+string(filepath.Separator)) {
				return nil
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return nil // 单件读败不炸谱
			}
			canon[spec.key] = append(canon[spec.key], b...)
			canon[spec.key] = append(canon[spec.key], '\n')
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return canon, nil
}

var binanceRe = regexp.MustCompile(`/(fapi|sapi)/[a-zA-Z0-9/_-]*[a-zA-Z0-9]`)

// binanceItems 已接面宇宙——码面 binance host 字面量（无官方镜像源）。
func binanceItems(code []byte) []item {
	seen := map[string]bool{}
	var out []item
	for _, m := range binanceRe.FindAll(code, -1) {
		p := strings.TrimSuffix(string(m), "/")
		if seen[p] {
			continue
		}
		seen[p] = true
		fam := strings.Split(strings.TrimPrefix(p, "/"), "/")[0]
		out = append(out, item{Face: p, Fam: fam})
	}
	return out
}

// ---- 差集 ----

func gap(venue string, items []item, canon, code []byte) map[string][]string {
	out := map[string][]string{}
	for _, it := range items {
		if bytes(canon, it.Face) || bytes(code, it.Face) {
			continue
		}
		out[it.Fam] = append(out[it.Fam], it.Face)
	}
	_ = venue
	return out
}

func bytes(hay []byte, needle string) bool {
	return len(needle) > 0 && strings.Contains(string(hay), needle)
}

// ---- ops_event 落账 ----

func postGaps(ctx context.Context, dsn string, gaps map[string]map[string][]string) (int, error) {
	// 迟绑 pgx——无 -post 不引库依赖导入成本（薄件人拉常态零 DB）。
	return postGapsPG(ctx, dsn, gaps)
}

func main() {
	var (
		root     = flag.String("root", ".", "仓根")
		pullOKX  = flag.Bool("pull-okx", false, "重拉官方 docs-v5/zh 页重建 okx 端点册")
		docsURL  = flag.String("docs-url", "https://www.okx.com/docs-v5/zh/", "OKX 官方文档页")
		mt4proto = flag.String("mt4proto", "/opt/grpc-proto/mt4/protos/mt4.proto", "mtapi mt4 权威 proto")
		mt5proto = flag.String("mt5proto", "proto/mtapi/mt5/mt5.proto", "仓内 mt5 proto 副本")
		snap     = flag.String("out", "docs/vencensus/census_gap.md", "快照落点")
		post     = flag.Bool("post", false, "差集族行落 ops_event（DB_DSN|FIN_PG_DSN）")
	)
	flag.Parse()

	okxTSV := filepath.Join(*root, "docs/vencensus/okx_v5_endpoints.tsv")
	if *pullOKX {
		n, err := pullOKXEndpoints(*docsURL, okxTSV)
		if err != nil {
			fmt.Fprintln(os.Stderr, "vencensus pull-okx:", err)
			os.Exit(1)
		}
		fmt.Printf("pull-okx: %d 端点重建 %s\n", n, okxTSV)
		return
	}

	c, err := corpus(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vencensus corpus:", err)
		os.Exit(1)
	}
	canon, code := c["canon"], c["code"]

	gaps := map[string]map[string][]string{}
	src := []string{}

	if items, err := loadTSV(okxTSV); err == nil {
		gaps["okx"] = gap("okx", items, canon, code)
		src = append(src, fmt.Sprintf("okx_v5_endpoints.tsv(%d)", len(items)))
	} else {
		fmt.Fprintln(os.Stderr, "vencensus: okx 册缺——先 -pull-okx:", err)
	}
	for _, sp := range []struct {
		venue, path string
	}{
		{"mt4", *mt4proto}, {"mt5", *mt5proto},
	} {
		p := sp.path
		if !filepath.IsAbs(p) {
			p = filepath.Join(*root, p)
		}
		if items, err := rpcItems(p); err == nil {
			gaps[sp.venue] = gap(sp.venue, items, canon, code)
			src = append(src, fmt.Sprintf("%s(%drpc)", filepath.Base(sp.path), len(items)))
		} else {
			fmt.Fprintf(os.Stderr, "vencensus: %s proto 缺（跳过非炸）: %v\n", sp.venue, err)
		}
	}
	bi := binanceItems(code)
	gaps["binance"] = gap("binance", bi, canon, code)
	src = append(src, fmt.Sprintf("binance-code(%d)", len(bi)))

	out := *snap
	if !filepath.IsAbs(out) {
		out = filepath.Join(*root, out)
	}
	if err := writeSnapshot(out, strings.Join(src, " "), gaps); err != nil {
		fmt.Fprintln(os.Stderr, "vencensus snapshot:", err)
		os.Exit(1)
	}
	for _, v := range []string{"okx", "mt4", "mt5", "binance"} {
		fams, n := 0, 0
		for _, eps := range gaps[v] {
			fams++
			n += len(eps)
		}
		fmt.Printf("%-8s %d 族 %d 面未登记\n", v, fams, n)
	}
	fmt.Println("snapshot:", out)

	if *post {
		dsn := os.Getenv("DB_DSN")
		if dsn == "" {
			dsn = os.Getenv("FIN_PG_DSN")
		}
		if dsn == "" {
			fmt.Fprintln(os.Stderr, "vencensus -post: DB_DSN/FIN_PG_DSN 空——fail-closed")
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		n, err := postGaps(ctx, dsn, gaps)
		if err != nil {
			fmt.Fprintln(os.Stderr, "vencensus post:", err)
			os.Exit(1)
		}
		fmt.Printf("post: %d 族行落 ops_event（抑重过）\n", n)
	}
}
