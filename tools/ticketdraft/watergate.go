package main

// watergate.go —— GW-001 段C3 批派水位机检/并票/派单闸升件面（自 main.go
// 拆分——check-lines 300 棘轮；本面=纯机检动作，物料化主径在 main.go）。

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 水位律「已批未派积压上限=5」——业主令 2026-11-04，超限 COS 停批新草案
// +草案照立不占位）。
func wipGateMax() int {
	if s := os.Getenv("O_TICKETDRAFT_WIP_MAX"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return 5
}

// wipBlocked 水位闸判（纯函数——已批未领+候选 ≥ 上限=停出票留 pending）。
func wipBlocked(wip, max int) bool { return wip >= max }

// wipCount 水位计数单源：dm_queue.sh approved（已批未领）+candidate（起草
// 未批）两面行数和——队列判据不在此复制（段B1 单源律：dm_queue 唯一推导）。
func wipCount(repo string) (int, error) {
	total := 0
	for _, mode := range []string{"approved", "candidate"} {
		out, err := exec.Command("bash", filepath.Join(repo, "scripts/meta/dm_queue.sh"), mode).Output()
		if err != nil {
			return 0, fmt.Errorf("dm_queue.sh %s: %v: %s", mode, err, tail(string(out), 300))
		}
		total += countLines(out)
	}
	return total, nil
}

func countLines(b []byte) int {
	n := 0
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(ln) != "" {
			n++
		}
	}
	return n
}

// absorbTicket 同签名族 24h 并票查：库内近 24h 同签名已物料化票号（最新
// 先到票为吸收票；无=空串）。fin_ticket_draft.signature 索引面（fin_0123）。
func absorbTicket(ctx context.Context, pool *pgxpool.Pool, sig string) (string, error) {
	var tid string
	err := pool.QueryRow(ctx, `SELECT ticket FROM fin_ticket_draft
		WHERE status='materialized' AND signature=$1
		  AND materialized_at>now()-interval '24 hours'
		ORDER BY materialized_at DESC,id DESC LIMIT 1`, sig).Scan(&tid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return tid, nil
}

// mergeDraft 并票落账：草案翻 merged 态（fin_0148 四态扩）+ticket 载吸收
// 票号+ops_event 并票行（并票=说明类不进相位——吸收票相位零扰动）。
func mergeDraft(ctx context.Context, pool *pgxpool.Pool, id int64, sig, absorbTID string) error {
	if _, err := pool.Exec(ctx, `UPDATE fin_ticket_draft
		SET status='merged',ticket=$2 WHERE id=$1 AND status='pending'`, id, absorbTID); err != nil {
		return fmt.Errorf("draft#%d 并票翻态: %w", id, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,'并票',$2,$3)`, ticketSeat, absorbTID,
		fmt.Sprintf("draft#%d 同签名族 %s 24h 并票——并入本票（GW-001 段C3 水位律：复发病灶一窗一票，后到草案不另占号）",
			id, sig)); err != nil {
		return fmt.Errorf("draft#%d 并票行: %w", id, err)
	}
	fmt.Printf("[ticketdraft] draft#%d 并票 %s——signature=%s（同签名族 24h 合并）\n", id, absorbTID, sig)
	return nil
}

// attachTKeyRE 附体判票号形（T-\d+——层扫对象真票键）。
var attachTKeyRE = regexp.MustCompile(`^T-[0-9]+$`)

// attachTicketOf 附体判：signature=「层扫:<类目>:<对象>」且对象=T-\d+
// →返回原票号（附体径）；否则空串=常径取号落件。
func attachTicketOf(p payload) string {
	if !strings.HasPrefix(p.Signature, "层扫:") {
		return ""
	}
	i := strings.Index(p.LesionKey, ":")
	if i <= 0 || i == len(p.LesionKey)-1 {
		return ""
	}
	obj := p.LesionKey[i+1:]
	if !attachTKeyRE.MatchString(obj) {
		return ""
	}
	return obj
}

// attachDraft 附体物料化：draft.ticket=原票号翻 materialized+处置注记行
// 附着原票链（非生命周期词零相位扰动——归并=诊断入原票史；不占新号
// 不立件不入 git）。seat=处置席=归领处置写席同径。
func attachDraft(ctx context.Context, pool *pgxpool.Pool, id int64, tid string, p payload) error {
	if _, err := pool.Exec(ctx, `UPDATE fin_ticket_draft
		SET status='materialized',ticket=$2,materialized_at=now() WHERE id=$1`,
		id, tid); err != nil {
		return fmt.Errorf("附体翻行: %w", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','处置',$1,$2)`, tid,
		fmt.Sprintf("层扫诊断归并%s——signature=%s（附体原票不占新号，draft#%d；T-0672）",
			tid, p.Signature, id)); err != nil {
		return fmt.Errorf("附体注记行: %w", err)
	}
	fmt.Printf("[ticketdraft] draft#%d 附体原票 %s——signature=%s\n", id, tid, p.Signature)
	return nil
}

// escWip 水位闸升件（饱和/不可判同键 24h 去重——esc_class 拥塞不入 hunger
// 探针类册=drafterProbeRules 无此键，不产自指草案；写败静默退结构性限度
// 同 esc）。
func escWip(ctx context.Context, pool *pgxpool.Pool, msg string) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, _ = pool.Exec(cctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件','T-0393',$2,'拥塞','TICKETDRAFT-WIP'
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='升件' AND note LIKE '水位闸:%' AND ts>now()-interval '24 hours')`,
		ticketSeat, fmt.Sprintf("水位闸:%s——草案留 pending 不占票号（GW-001 段C3 水位机检；队列消化后下拍自续）", msg))
}
