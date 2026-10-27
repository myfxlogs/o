// reverse.go —— 反向桥面（T-0172/D-0030 §五延伸）：SG 正本 fin_strategy/
// fin_venue_untradeable → PG-md 只读副本。全表 diff 搬运（行量百级，仅差行过
// 隧道）；ev_strategy 帧（fin_0028 在册）即发唤醒+期界对账兜底。研究面只读
// （写权唯一=本臂）；正本无删径（state 表零删——retention sweep 白名单闭集），
// 副本 upsert-only。numeric 以文本搬运保精度（mirror.go 同律）。
package projector

import (
	"context"
	"fmt"
	"time"
)

const qStrategyRev = `SELECT id,inst_id,template_ref,proposal_ref,state,notional_usd::text,
	criteria::text,entered_at,created_at,venue,direction FROM fin_strategy`

const qUntradeableRev = `SELECT inst_id,code,msg,first_rejected_at FROM fin_venue_untradeable`

// 副本落地面（md 侧 upsert——id/inst_id 键全列覆写，SG 正本赢）。
const uStrategyMirror = `INSERT INTO fin_strategy(id,inst_id,template_ref,proposal_ref,state,
	notional_usd,entered_at,criteria,created_at,venue,direction)
	VALUES($1,$2,$3,$4,$5,$6::numeric,$7::timestamptz,$8::jsonb,$9::timestamptz,$10,$11)
	ON CONFLICT (id) DO UPDATE SET inst_id=EXCLUDED.inst_id,template_ref=EXCLUDED.template_ref,
		proposal_ref=EXCLUDED.proposal_ref,state=EXCLUDED.state,notional_usd=EXCLUDED.notional_usd,
		entered_at=EXCLUDED.entered_at,criteria=EXCLUDED.criteria,created_at=EXCLUDED.created_at,
		venue=EXCLUDED.venue,direction=EXCLUDED.direction`

const uUntradeableMirror = `INSERT INTO fin_venue_untradeable(inst_id,code,msg,first_rejected_at)
	VALUES($1,$2,$3,$4::timestamptz)
	ON CONFLICT (inst_id) DO UPDATE SET code=EXCLUDED.code,msg=EXCLUDED.msg,
		first_rejected_at=EXCLUDED.first_rejected_at`

type strategyRow struct {
	inst, tmpl, state string
	proposal          *int64
	notional, crit    *string
	entered, created  time.Time
	venue, dir        string
}

// reverseStrategies 策略正本→副本全表 diff（dedupe 闸读面——funding persist/
// dcd eval 提案闸 fin_strategy 计数不再 42P01）。
func (r *Runner) reverseStrategies(ctx context.Context) error {
	sgRows, err := r.sg.Query(ctx, qStrategyRev)
	if err != nil {
		return fmt.Errorf("strategy 正本读: %w", err)
	}
	want := map[int64]strategyRow{}
	defer sgRows.Close()
	for sgRows.Next() {
		var id int64
		var s strategyRow
		if err := sgRows.Scan(&id, &s.inst, &s.tmpl, &s.proposal, &s.state,
			&s.notional, &s.crit, &s.entered, &s.created, &s.venue, &s.dir); err != nil {
			return err
		}
		want[id] = s
	}
	if err := sgRows.Err(); err != nil {
		return err
	}
	have := map[int64]strategyRow{}
	mdRows, err := r.md.Query(ctx, qStrategyRev)
	if err != nil {
		return fmt.Errorf("strategy 副本读: %w", err)
	}
	defer mdRows.Close()
	for mdRows.Next() {
		var id int64
		var s strategyRow
		if err := mdRows.Scan(&id, &s.inst, &s.tmpl, &s.proposal, &s.state,
			&s.notional, &s.crit, &s.entered, &s.created, &s.venue, &s.dir); err != nil {
			return err
		}
		have[id] = s
	}
	if err := mdRows.Err(); err != nil {
		return err
	}
	var batch [][]any
	for id, s := range want {
		if h, ok := have[id]; !ok || h != s {
			batch = append(batch, []any{id, s.inst, s.tmpl, s.proposal, s.state,
				s.notional, s.entered, s.crit, s.created, s.venue, s.dir})
		}
	}
	return batchExec(ctx, r.md, uStrategyMirror, batch)
}

// reverseUntradeable 黑名单正本→副本全表 diff（起草闸读面——spotLegOK/
// gapscan classify/hypothesis xcarry 滤除恢复；正本学习入册低频，期界对账
// 兜底足够，无独立事件帧）。
func (r *Runner) reverseUntradeable(ctx context.Context) error {
	sgRows, err := r.sg.Query(ctx, qUntradeableRev)
	if err != nil {
		return fmt.Errorf("untradeable 正本读: %w", err)
	}
	type row struct {
		code, msg string
		at        time.Time
	}
	want := map[string]row{}
	defer sgRows.Close()
	for sgRows.Next() {
		var inst string
		var x row
		if err := sgRows.Scan(&inst, &x.code, &x.msg, &x.at); err != nil {
			return err
		}
		want[inst] = x
	}
	if err := sgRows.Err(); err != nil {
		return err
	}
	have := map[string]row{}
	mdRows, err := r.md.Query(ctx, qUntradeableRev)
	if err != nil {
		return fmt.Errorf("untradeable 副本读: %w", err)
	}
	defer mdRows.Close()
	for mdRows.Next() {
		var inst string
		var x row
		if err := mdRows.Scan(&inst, &x.code, &x.msg, &x.at); err != nil {
			return err
		}
		have[inst] = x
	}
	if err := mdRows.Err(); err != nil {
		return err
	}
	var batch [][]any
	for inst, x := range want {
		if h, ok := have[inst]; !ok || h != x {
			batch = append(batch, []any{inst, x.code, x.msg, x.at})
		}
	}
	return batchExec(ctx, r.md, uUntradeableMirror, batch)
}
