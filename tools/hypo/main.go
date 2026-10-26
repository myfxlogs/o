// hypo —— 研究员席窄门写径（T-0121/D-0016 Ⅲ）：短会话起草席唯一可执行面。
// 两子命令——card: INSERT fin_opp_hypothesis（UNIQUE(venue,surface) 幂等，
// 重复面不重生）；metrics: 周卡数/采纳率 upsert fin_employee_registry
// 「研究员」行（产出度量机读+last_beat 在岗锚——research_arm_silent 看门锚源）。
// 零越权证载体：本文件=研究员全部 DB 写面，仅此两条写语句——venue 写径/
// fin_opp_template/fin_proposal 零触（grep 证锚）。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fatal(err)
	}
	defer pool.Close()
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "card":
		card(ctx, pool, os.Args[2:])
	case "metrics":
		metrics(ctx, pool)
	default:
		usage()
	}
}

func card(ctx context.Context, pool *pgxpool.Pool, args []string) {
	fs := flag.NewFlagSet("card", flag.ExitOnError)
	venue := fs.String("venue", "", "venue（okx/mt5/cmc/trx…）")
	surface := fs.String("surface", "", "能力面名（fin_capability_surface.surface 同名）")
	family := fs.String("family", "", "结构族（carry/hedge/basis/vol-sell 或新族名）")
	legs := fs.String("legs", "", "腿原语逗号分隔（文法闭集优先，超集须 evidence 说明）")
	req := fs.String("req", "{}", "需求清单 JSON（evaluator/risk 阈/数据面）")
	ev := fs.String("ev", "{}", "证据 JSON（出处链接；provenance 工具强制注入）")
	fs.Parse(args)
	if *venue == "" || *surface == "" || *family == "" {
		fatal(fmt.Errorf("card: venue/surface/family 必填"))
	}
	if err := jsonValid(*req, "req"); err != nil {
		fatal(err)
	}
	if err := jsonValid(*ev, "ev"); err != nil {
		fatal(err)
	}
	evm := map[string]any{}
	if err := json.Unmarshal([]byte(*ev), &evm); err != nil {
		fatal(err)
	}
	evm["provenance"] = "researcher" // 溯源强制——窄门盖章非会话自报
	evb, _ := json.Marshal(evm)
	legb, _ := json.Marshal(splitCSV(*legs)) // 空 legs=[]（非 null——列 NOT NULL）
	tag, err := pool.Exec(ctx, `INSERT INTO fin_opp_hypothesis(venue,surface,family,leg_primitives,requirements,evidence)
		VALUES($1,$2,$3,$4::jsonb,$5::jsonb,$6::jsonb)
		ON CONFLICT (venue,surface) DO NOTHING`,
		*venue, *surface, *family, string(legb), *req, string(evb))
	if err != nil {
		fatal(err)
	}
	if tag.RowsAffected() == 0 {
		fmt.Printf("exists（重复面不重生——%s/%s 卡已在册，state 见表）\n", *venue, *surface)
		return
	}
	fmt.Printf("inserted（proposed 待审——%s/%s family=%s legs=%s）\n", *venue, *surface, *family, *legs)
}

func metrics(ctx context.Context, pool *pgxpool.Pool) {
	var weekly, adopted, total int
	// 席 KPI=席卡——provenance=researcher 收窄（grammar/假想官卡不入席度量；
	// wk_all=7/wk_seat=0 锚污染实案 2026-10-26）
	if err := pool.QueryRow(ctx, `SELECT
			COUNT(*) FILTER (WHERE ts > now()-interval '7 days'),
			COUNT(*) FILTER (WHERE state='adopted'),
			COUNT(*)
		FROM fin_opp_hypothesis WHERE evidence->>'provenance'='researcher'`).Scan(&weekly, &adopted, &total); err != nil {
		fatal(err)
	}
	rate := 0.0
	if total > 0 {
		rate = float64(adopted) / float64(total)
	}
	meta, _ := json.Marshal(map[string]any{
		"weekly_cards": weekly, "adopted": adopted,
		"adopt_rate": rate, "total_cards": total, "window": time.Now().UTC().Format("2006-01"),
	})
	if _, err := pool.Exec(ctx, `INSERT INTO fin_employee_registry(name,dept,title,meta,last_beat)
		VALUES('研究员','research','外部进水口起草席（D-0016 Ⅲ——文法空格×venue 变更×经济经验库→假想卡）',$1::jsonb,now())
		ON CONFLICT(name) DO UPDATE SET dept=EXCLUDED.dept, title=EXCLUDED.title,
			meta=EXCLUDED.meta, status='active', last_beat=now()`, string(meta)); err != nil {
		fatal(err)
	}
	fmt.Printf("metrics 刷讫——周卡=%d 采纳=%d/%d(%.0f%%) registry 研究员行 Beat=now\n",
		weekly, adopted, total, rate*100)
}

func splitCSV(s string) []string {
	out := []string{}
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '，' }) {
		out = append(out, strings.TrimSpace(f))
	}
	return out
}

func jsonValid(s, name string) error {
	if !json.Valid([]byte(s)) {
		return fmt.Errorf("card: -%s 非合法 JSON: %s", name, s)
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, `用法:
  hypo card -venue okx -surface spread-trading -family basis -legs spot_long,perp_short [-req '{}'] [-ev '{"src":"…"}']
  hypo metrics`)
	os.Exit(2)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
