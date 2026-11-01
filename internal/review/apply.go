// apply.go —— 判决/状态应用分离（T-0232 乙案，D-0036 §12.2 裁4）：OCI 评史官
// 只写判决事实（fin_review/fin_review_verdict 追加——review_rw 收窄面），
// 本件=SG 侧判决消费者：轮询 fin_review 增量，falsified 判决应用钱径态迁
// （fin_proposal 同键退役+fin_strategy degraded+transition 留痕）。
// 过渡件纪律（票面钦定）：持久化游标 fin_verdict_cursor + 幂等键
// （verdict_id=fin_review.id 去重）——outbox 判决道（T-0231）切换后本件
// 废弃（票面验收锚：grep 零引用）。
package review

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/rejects"
	"o/internal/platform/roster"
)

const verdictPollEvery = 30 * time.Second

// Applier 判行官岗——SG 侧判决应用消费者（钱径态迁唯一写径；OCI 侧产判决事实）。
type Applier struct {
	pool *pgxpool.Pool
	rej  *rejects.Door
}

func NewApplier(pool *pgxpool.Pool) *Applier {
	return &Applier{pool: pool, rej: rejects.New("verdict_apply", "lookup_err", "write_err")}
}

func (a *Applier) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, a.pool, "判行官", "review",
		"falsified 判决状态应用官（T-0232 乙案——判决事实 SG 消费面）", nil); err != nil {
		errc <- fmt.Errorf("判行官 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, a.pool, "判行官", 30*time.Second, errc)
	if err := a.applyDue(ctx); err != nil { // boot 首拍先对齐——起步态一次追平
		log.Printf("[verdict-apply] boot 首拍: %v", err)
	}
	t := time.NewTicker(verdictPollEvery) // sched-exempt-ok：T-0232 钦定过渡轮询件（fin_review 增量消费——持久游标+幂等键；outbox 判决道切换即废，票面废弃锚钉死）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := a.applyDue(ctx); err != nil {
				a.rej.Inc("write_err")
				log.Printf("[verdict-apply] 增量拍败: %v", err)
			}
		}
	}
}

// applyDue 消费 fin_review 增量（游标后 falsified 行逐行同事务应用+游标推进；
// 非 falsified 行由尾步游标过顶跳过——幂等键=fin_review.id）。
func (a *Applier) applyDue(ctx context.Context) error {
	rows, err := a.pool.Query(ctx, `SELECT id,position_ref,attribution FROM fin_review
		WHERE verdict='falsified' AND id>COALESCE(
			(SELECT applied_upto FROM fin_verdict_cursor WHERE k=1),0)
		ORDER BY id`)
	if err != nil {
		return err
	}
	type vrow struct {
		id, pos int64
		attrib  string
	}
	var vs []vrow
	for rows.Next() {
		var v vrow
		if err := rows.Scan(&v.id, &v.pos, &v.attrib); err != nil {
			rows.Close()
			return err
		}
		vs = append(vs, v)
	}
	rows.Close()
	for _, v := range vs {
		if err := a.applyFalsified(ctx, v.id, v.pos, v.attrib); err != nil {
			return err // 本行未收口=游标不过——下拍重试（幂等重放安全）
		}
	}
	// 游标过顶：非 falsified 行+已应用行一井跳（max(id) 水位）。
	_, err = a.pool.Exec(ctx, `UPDATE fin_verdict_cursor SET
		applied_upto=GREATEST(applied_upto,(SELECT COALESCE(max(id),0) FROM fin_review)),
		updated_at=now() WHERE k=1`)
	return err
}

// applyFalsified 单行应用（自 run.go:200-216 原样搬迁，D-514 内联注释随迁）：
// 查链（position→execution→strategy）失败=行不可解——计数门+ALERT 留痕后游标
// 照推（毒丸不堵队：判决事实行在 fin_review 永久在案，应用缺位非丢）。
func (a *Applier) applyFalsified(ctx context.Context, rid, posID int64, attrib string) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var execID, sid int64
	var inst, dir string
	skip := func() error {
		a.rej.Inc("lookup_err")
		if _, err := tx.Exec(ctx,
			"UPDATE fin_verdict_cursor SET applied_upto=GREATEST(applied_upto,$1),updated_at=now() WHERE k=1",
			rid); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if err := tx.QueryRow(ctx,
		"SELECT execution_id FROM fin_position WHERE id=$1", posID).Scan(&execID); err != nil {
		log.Printf("ALERT [verdict-apply] 判决 #%d 仓行查败（pos=%d）——跳过应用（行留案）", rid, posID)
		return skip()
	}
	if err := tx.QueryRow(ctx, `SELECT s.id,s.inst_id,s.direction FROM fin_strategy s
		JOIN fin_execution f ON f.strategy_ref=s.id WHERE f.id=$1`,
		execID).Scan(&sid, &inst, &dir); err != nil {
		log.Printf("ALERT [verdict-apply] 判决 #%d 回炉策略行查败（exec=%d）——跳过应用", rid, execID)
		return skip()
	}
	if tag, err := tx.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired',retire_reason='falsified_回炉',retired_at=now()
		WHERE inst_id=$1 AND direction=$2 AND state='fin_armed'`, inst, dir); err == nil && tag.RowsAffected() > 0 {
		log.Printf("[verdict-apply] falsified 回炉——同键 armed 提案退役 %d 行（%s %s）", tag.RowsAffected(), inst, dir)
	}
	// live→degraded 合法径内联迁（review→lifecycle 跨包向禁——D-514）：
	// state='live' 谓词=只迁 live 行，非 live 行零动不误迁。
	tag, err := tx.Exec(ctx, `UPDATE fin_strategy SET state='degraded',entered_at=now()
		WHERE id=$1 AND state='live'`, sid)
	if err != nil {
		a.rej.Inc("write_err")
		return fmt.Errorf("策略 #%d falsified 回炉降级败: %w", sid, err)
	}
	if tag.RowsAffected() > 0 {
		if _, err := tx.Exec(ctx, `INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
			VALUES($1,'live','degraded',$2)`, sid,
			fmt.Sprintf("falsified 判决回炉（归因=%s）——降级待复盘非原地待命", attrib)); err != nil {
			a.rej.Inc("write_err")
			return fmt.Errorf("策略 #%d transition 留痕败: %w", sid, err)
		}
		log.Printf("[verdict-apply] 策略 #%d %s falsified（归因=%s）→degraded 回炉", sid, inst, attrib)
	}
	if _, err := tx.Exec(ctx,
		"UPDATE fin_verdict_cursor SET applied_upto=$1,updated_at=now() WHERE k=1", rid); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
