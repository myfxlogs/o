// ticketdraft —— 候选票草案物料化径（T-0393 范围⑤配套）：fin_ticket_draft
// pending→docs/tasks/<TID>-<slug>.md 候选态票件。确定性模板填充
// （docs/templates/ticket-draft.md 槽位←payload 机读件——LLM 不入安全径）；
// 取号唯一径 scripts/next_ticket.sh（领号台账行自带）；落件即 git commit
// （-o -- 单件面禁裸 -A）+git_push.sh——写径只落候选草案态零自派
// （批准权=COS/DM，派发权=DM，权序红线 T-0376）。败=ops_event 升件自报
// （写败/META-DRAFT——类册面不入起草拍签名域防自起草回环）。
//
// 用法（dm_tick.sh 零 LLM 尾段同拍；手动同径）：
//
//	FIN_PG_DSN=… ticketdraft [-repo /opt/o] [-dry-run]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ticketSeat 物料化落账席别（DM 执行环=决策席同 drafterSeat 口径）。
const ticketSeat = "决策席"

func main() {
	repo := flag.String("repo", "/opt/o", "物料化落件仓根（master 检出面）")
	dry := flag.Bool("dry-run", false, "干跑——只列 pending 不写件/不落账（锚③ 生产谱面 dry 对拍面）")
	flag.Parse()
	dsn := os.Getenv("FIN_PG_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "FIN_PG_DSN 缺——钱径池不可达 fail-closed")
		os.Exit(2)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ticketdraft: 池建败: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := run(ctx, pool, *repo, *dry); err != nil {
		esc(ctx, pool, err)
		fmt.Fprintf(os.Stderr, "ticketdraft: %v\n", err)
		os.Exit(1)
	}
}

// payload 草案载荷（与 internal/portfolio/hunger draftPayload 同构——
// 跨包不引：生产侧件私有，物料化面自读 jsonb 字段名等值契约）。
type payload struct {
	Slug          string  `json:"slug"`
	Title         string  `json:"title"`
	Signature     string  `json:"signature"`
	Kind          string  `json:"kind"`
	LesionKey     string  `json:"lesion_key"`
	EvidenceIDs   []int64 `json:"evidence_ids"`
	EvidenceNote  string  `json:"evidence_note"`
	EvidenceQuery string  `json:"evidence_query"`
	DomainHint    string  `json:"domain_hint"`
}

func run(ctx context.Context, pool *pgxpool.Pool, repo string, dry bool) error {
	rows, err := pool.Query(ctx, `SELECT id,payload FROM fin_ticket_draft
		WHERE status='pending' ORDER BY id`)
	if err != nil {
		return fmt.Errorf("pending 读败: %w", err)
	}
	defer rows.Close()
	type draft struct {
		id int64
		p  payload
	}
	var ds []draft
	for rows.Next() {
		var d draft
		var raw []byte
		if err := rows.Scan(&d.id, &raw); err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &d.p); err != nil {
			return fmt.Errorf("draft#%d 载荷解析: %w", d.id, err)
		}
		ds = append(ds, d)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if dry {
		for _, d := range ds {
			fmt.Printf("[dry] draft#%d signature=%s slug=%s\n", d.id, d.p.Signature, d.p.Slug)
		}
		return nil
	}
	tpl, err := readTemplate(repo)
	if err != nil {
		return fmt.Errorf("模板件读败: %w", err)
	}
	for _, d := range ds {
		if err := materialize(ctx, pool, repo, string(tpl), d.id, d.p); err != nil {
			return fmt.Errorf("draft#%d %s: %w", d.id, d.p.Signature, err)
		}
		fmt.Printf("[ticketdraft] draft#%d 物料化落件——signature=%s\n", d.id, d.p.Signature)
	}
	return nil
}

// readTemplate 模板件读径（repo 缺省 /opt/o——桩测面仓根相对径回落
// 兼容 go test 工作目录=包目录）。
func readTemplate(repo ...string) ([]byte, error) {
	base := "/opt/o"
	if len(repo) > 0 {
		base = repo[0]
	}
	for _, p := range []string{
		filepath.Join(base, "docs/templates/ticket-draft.md"),
		"../../docs/templates/ticket-draft.md",
	} {
		if b, err := os.ReadFile(p); err == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("ticket-draft.md 两径皆缺（%s + 包相对径）", base)
}

// render 模板槽填充（{{KEY}} 字面替换——确定性零求值）。
func render(tpl string, p payload, tid, draftID string) string {
	ids := make([]string, len(p.EvidenceIDs))
	for i, id := range p.EvidenceIDs {
		ids[i] = fmt.Sprintf("#%d", id)
	}
	reps := map[string]string{
		"{{TID}}":            tid,
		"{{DRAFT_ID}}":       draftID,
		"{{TITLE}}":          p.Title,
		"{{SIGNATURE}}":      p.Signature,
		"{{KIND}}":           p.Kind,
		"{{LESION_KEY}}":     p.LesionKey,
		"{{EVIDENCE_IDS}}":   strings.Join(ids, ","),
		"{{EVIDENCE_NOTE}}":  p.EvidenceNote,
		"{{EVIDENCE_QUERY}}": p.EvidenceQuery,
		"{{DOMAIN_HINT}}":    p.DomainHint,
	}
	out := tpl
	for k, v := range reps {
		out = strings.ReplaceAll(out, k, v)
	}
	return out
}

// materialize 单件物料化：取号→模板填充→落件→commit+push→行翻
// materialized+起草行携新票号（串链——draft#↔TID 双向可溯）。
func materialize(ctx context.Context, pool *pgxpool.Pool, repo, tpl string, id int64, p payload) error {
	tidb, err := exec.Command("bash", filepath.Join(repo, "scripts/next_ticket.sh"), "起草拍").Output()
	if err != nil {
		return fmt.Errorf("取号败: %w", err)
	}
	tid := strings.TrimSpace(string(tidb))
	rel := fmt.Sprintf("docs/tasks/%s-%s.md", tid, p.Slug)
	abs := filepath.Join(repo, rel)
	if _, err := os.Stat(abs); err == nil {
		return fmt.Errorf("票件已存在（撞号面——取号闸漏）: %s", rel)
	}
	body := render(tpl, p, tid, fmt.Sprintf("%d", id))
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		return fmt.Errorf("票件写败: %w", err)
	}
	if out, err := exec.Command("git", "-C", repo, "add", "--", rel).CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}
	msg := fmt.Sprintf("[角色:决策][%s] 探针升件自动起草——%s（签名 %s，draft#%d，T-0393 起草拍件）",
		tid, p.Title, p.Signature, id)
	if out, err := exec.Command("git", "-C", repo, "commit", "-o", "--", rel, "-m", msg).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	if out, err := exec.Command("bash", filepath.Join(repo, "scripts/git_push.sh")).CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %w: %s", err, out)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_ticket_draft
		SET status='materialized',ticket=$2,materialized_at=now() WHERE id=$1`,
		id, tid); err != nil {
		return fmt.Errorf("行翻 materialized: %w", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'起草',$2,$3)`,
		ticketSeat, tid,
		fmt.Sprintf("自动起草落件 %s——签名 %s（draft#%d 物料化，候选态零自派，批准权=COS/DM）",
			rel, p.Signature, id)); err != nil {
		return fmt.Errorf("起草行: %w", err)
	}
	return nil
}

// esc 物料化径败自报升件（写败/META-DRAFT——note 前缀同窗 24h 至多一行；
// 自报行自身写败=静默退，结构性限度同 hungerEscalate 在案）。
func esc(ctx context.Context, pool *pgxpool.Pool, cause error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	prefix := "ticketdraft 物料化败"
	_, _ = pool.Exec(cctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件','T-0393',$2,'写败','META-DRAFT'
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='升件' AND note LIKE $3 AND ts>now()-interval '24 hours')`,
		ticketSeat, fmt.Sprintf("%s: %v", prefix, cause), prefix+"%")
}
