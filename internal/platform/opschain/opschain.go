// Package opschain —— ops_event 防篡改链验件（T-0386 自我迭代加固族①）：
// 库内单源 ops_chain_verify() 全链重算→创世锚/伪戳/断链/裸行四类断检出
// →`账链断`+`升件`自报双行。验链重算=纯读零写，检出落行=唯一写径。
// 挂点=lifecycle 饥饿同槽慢拍+tools/hungertick 同径（票面钦定慢拍位）。
package opschain

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// 事件行席别/常设环票号（DM 执行环=决策席、例行行携环票号——hunger 同例）。
const (
	seat   = "决策席"
	ticket = "T-0386"
)

// Finding 链断点——kind 四类：创世锚|伪戳|断链|裸行（ops_chain_verify 输出口径）。
type Finding struct {
	ID     int64
	Kind   string
	Detail string
}

// Report 验链结果——零 Findings=链完。
type Report struct {
	Findings []Finding
}

// OK 链完判（nil 报告亦完——调用面容错）。
func (r *Report) OK() bool { return r == nil || len(r.Findings) == 0 }

// Verifier 验链器（fin 池——ops_event 同源池）。
type Verifier struct{ pool *pgxpool.Pool }

// New 装配。
func New(pool *pgxpool.Pool) *Verifier { return &Verifier{pool: pool} }

// Verify 全链重算（纯读零写——ops_chain_verify() 库内单源，口径与
// check_ops_chain.sh LIVE 段一式）。
func (v *Verifier) Verify(ctx context.Context) (*Report, error) {
	rows, err := v.pool.Query(ctx,
		`SELECT id,kind,detail FROM ops_chain_verify() ORDER BY id,kind`)
	if err != nil {
		return nil, fmt.Errorf("ops_chain_verify: %w", err)
	}
	defer rows.Close()
	rep := &Report{}
	for rows.Next() {
		var f Finding
		if err := rows.Scan(&f.ID, &f.Kind, &f.Detail); err != nil {
			return nil, fmt.Errorf("ops_chain_verify scan: %w", err)
		}
		rep.Findings = append(rep.Findings, f)
	}
	return rep, rows.Err()
}

// ReportBreak 断链自报双行：ops_event `账链断`行（检出证据——首处锚）+
// `升件`行（账链断类/ops_event 域——判类册登记词）。同窗 24h 至多一报
// （hungerEscalate 同构——断链未修复持续检出防每拍爆量；升件不丢=持久在账）。
func (v *Verifier) ReportBreak(ctx context.Context, rep *Report) error {
	if rep.OK() {
		return nil
	}
	f := rep.Findings[0]
	note := fmt.Sprintf("检出=%d 首处=%s@#%d %s", len(rep.Findings), f.Kind, f.ID, f.Detail)
	tag, err := v.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		SELECT $1,'账链断',$2,$3 WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='账链断' AND ts>now()-interval '24 hours')`,
		seat, ticket, note)
	if err != nil {
		return fmt.Errorf("账链断行: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil // 同窗已报——24h 内至多一报
	}
	if _, err := v.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,'账链断','ops_event')`,
		seat, ticket, "ops_event 账链断——"+note); err != nil {
		return fmt.Errorf("账链断升件行: %w", err)
	}
	return nil
}
