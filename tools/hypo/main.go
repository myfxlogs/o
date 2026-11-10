// hypo —— 研究员席窄门写径（T-0121/D-0016 Ⅲ）：短会话起草席唯一可执行面。
// 两子命令——card: INSERT fin_opp_hypothesis（UNIQUE(venue,surface) 幂等，
// 重复面不重生）；metrics: 周卡数/采纳率 upsert fin_employee_registry
// 「研究员」行（产出度量机读+last_beat 在岗锚——research_arm_silent 看门锚源）。
// decide: 假想卡 state 迁移窄门（execfam 骨检+ops_event 审计行；
// OPS_DB_DSN 置=审计跨库两序写——md 正本与 fin 正本裂面实据在案）。
// 零越权证载体：本文件=研究员全部 DB 写面——venue 写径/
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

	"o/internal/platform/execfam"
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
	case "decide":
		decide(ctx, pool, os.Args[2:])
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

// decide 卡裁决窄门（T-0522 骨架批——决策席/唤知会话卡生命周期写径）：
// proposed|parked → adopted|parked|dismissed 单事务两件：state 迁移+
// ops_event 处置审计行（旧态/新态/裁词必携——append-only 证据轨）。
// adopted 骨检=execfam 册（五件骨架的执行族锚——未注册拒收养,防「收养了
// 但跑不起来」的假骨架）；模板名=surface 剥 _template 尾（grammar 卡构键律）
// 或 -template 显式传（研究员卡面）。
func decide(ctx context.Context, pool *pgxpool.Pool, args []string) {
	fs := flag.NewFlagSet("decide", flag.ExitOnError)
	id := fs.Int64("id", 0, "假想卡 id")
	state := fs.String("state", "", "目标态 adopted|parked|dismissed")
	reason := fs.String("reason", "", "裁词（必填——入审计行证据面）")
	seat := fs.String("seat", "处置席", "裁席名（ops_event.seat——fin_0154 矩阵 处置∈{处置席,复审席}）")
	tmpl := fs.String("template", "", "模板名（缺省=surface 剥 _template 尾）")
	fs.Parse(args)
	if *id <= 0 || *reason == "" {
		fatal(fmt.Errorf("decide: -id 与 -reason 必填"))
	}
	switch *state {
	case "adopted", "parked", "dismissed":
	default:
		fatal(fmt.Errorf("decide: -state 闭集 adopted|parked|dismissed（%q）", *state))
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var old, surface string
	if err := tx.QueryRow(ctx,
		`SELECT state,surface FROM fin_opp_hypothesis WHERE id=$1 FOR UPDATE`, *id).
		Scan(&old, &surface); err != nil {
		fatal(fmt.Errorf("decide: 卡 #%d 读败: %w", *id, err))
	}
	if old != "proposed" && old != "parked" {
		fatal(fmt.Errorf("decide: 卡 #%d state=%s 不可迁（闭集源态 proposed|parked）", *id, old))
	}
	t := *tmpl
	if t == "" {
		t = strings.TrimSuffix(surface, "_template")
	}
	if *state == "adopted" && !execfam.Has(t) {
		fatal(fmt.Errorf("decide: 收养拒——execfam 未注册 %s（骨架未齐先落执行族册）", t))
	}
	// state 迁+evidence.last_reviewed 复核戳合写（T-0523——parked 再议
	// 节律键：orphan_hypothesis_parked 探针以此戳判挂期,decide=复核即落戳；
	// re-park 续挂 30d 不扰面,超戳=需求黑洞如实浮面）。
	if _, err := tx.Exec(ctx,
		`UPDATE fin_opp_hypothesis SET state=$1,
			evidence = evidence || jsonb_build_object('last_reviewed',
				to_char(now() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'))
			WHERE id=$2`, *state, *id); err != nil {
		fatal(fmt.Errorf("decide: state 迁败: %w", err))
	}
	note := fmt.Sprintf("假想卡#%d %s: %s→%s 裁词=%s", *id, surface, old, *state, *reason)
	// 审计面跨库裂（KB hypothesis-cards-adopt-via-hypo-tool——fin_opp_hypothesis
	// 正本在 OCI PG-md，ops_event 正本在 SG fin 库）：OPS_DB_DSN 置=两序写
	// （态迁先落审计后补，审计落败=态已迁+显式重放声，零静默）；
	// 缺省=同库同事务（测试/同域面原语义）。
	if opsDSN := os.Getenv("OPS_DB_DSN"); opsDSN != "" {
		if err := tx.Commit(ctx); err != nil {
			fatal(err)
		}
		opool, err := pgxpool.New(ctx, opsDSN)
		if err != nil {
			fatal(fmt.Errorf("decide: 态已迁 #%d %s→%s——审计面连落败（OPS_DB_DSN），手工补 INSERT INTO ops_event(seat,event,ticket,note) VALUES('%s','处置','hypo','%s'): %w", *id, old, *state, *seat, note, err))
		}
		defer opool.Close()
		if _, err := opool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'处置','hypo',$2)`,
			*seat, note); err != nil {
			fatal(fmt.Errorf("decide: 态已迁 #%d %s→%s——审计行落败，手工补 INSERT INTO ops_event(seat,event,ticket,note) VALUES('%s','处置','hypo','%s'): %w", *id, old, *state, *seat, note, err))
		}
		fmt.Printf("decide #%d %s→%s（template=%s 审计行落账·跨库 OPS_DB_DSN）\n", *id, old, *state, t)
		return
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'处置','hypo',$2)`,
		*seat, note); err != nil {
		fatal(fmt.Errorf("decide: 审计行落败: %w", err))
	}
	if err := tx.Commit(ctx); err != nil {
		fatal(err)
	}
	fmt.Printf("decide #%d %s→%s（template=%s 审计行落账）\n", *id, old, *state, t)
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
  hypo metrics
  hypo decide -id <N> -state adopted|parked|dismissed -reason "裁词" [-seat 决策席] [-template 名]`)
	os.Exit(2)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
