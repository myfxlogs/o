package grammar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// scanTracked 仅统计 vol-sell 自己的目录/数据/凭据读败；其他 grammar 族故障
// 不连坐。连败严格 >N 后以 SG 条件盒持久停扫，业主 fulfilled 才恢复。
func (r *Runner) scanTracked(ctx context.Context) error {
	r.volSellAttempted, r.volSellFailed, r.volSellSkip = false, false, false
	if r.volSellEnabled {
		stopped, err := r.volSellRetired(ctx)
		if err != nil {
			r.rej.Inc("volsell_retire_state_read_err")
			r.volSellSkip = true
			log.Printf("WARN [research/grammar] volsell 复役闸读取失败，fail-closed: %v", err)
		} else {
			r.volSellSkip = stopped
			if stopped {
				r.volSellWasRetired = true
			} else if r.volSellWasRetired {
				r.volSellWasRetired, r.volSellStreak = false, 0
			}
		}
	}
	err := r.scan(ctx)
	if !r.volSellEnabled || r.volSellSkip {
		return err
	}
	if r.volSellFailed {
		r.volSellStreak++
		if retireErr := r.maybeRetireVolSell(ctx); retireErr != nil {
			return fmt.Errorf("volsell 扫描败 (%v); 持久停扫写败: %w", err, retireErr)
		}
	} else if r.volSellAttempted {
		r.volSellStreak = 0
	}
	return err
}

func (r *Runner) markVolSellFailure(err error) {
	r.volSellFailed = true
	if err != nil {
		r.volSellLastErr = err.Error()
	}
}

func (r *Runner) volSellRetired(ctx context.Context) (bool, error) {
	if r.fin == nil {
		return true, fmt.Errorf("SG fin writer missing")
	}
	var state string
	err := r.fin.QueryRow(ctx, `SELECT state FROM fin_condition_gap
		WHERE kind='permission' AND venue='okx' AND surface='volsell-fail-streak'
		ORDER BY requested_at DESC,id DESC LIMIT 1`).Scan(&state)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return true, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return state != "fulfilled", nil
}

func (r *Runner) maybeRetireVolSell(ctx context.Context) error {
	if r.volSellStreak <= r.volSellFailN {
		return nil
	}
	if r.fin == nil {
		r.rej.Inc("volsell_retire_writer_missing")
		return fmt.Errorf("SG fin writer missing")
	}
	detail := fmt.Sprintf("vol_sell 连续读败/错件 %d 拍（>N=%d）——自停待业主复役；最后错=%s",
		r.volSellStreak, r.volSellFailN, r.volSellLastErr)
	payload, _ := json.Marshal(map[string]any{
		"auto_retire": "research.volsell.fail_streak_n",
		"ask":         "vol-sell 连败自停；核因后由业主明确 fulfilled 复役（SG fin 正本）",
		"streak":      r.volSellStreak, "n": r.volSellFailN, "last_err": r.volSellLastErr,
	})
	tx, err := r.fin.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('T-0356:volsell-fail-streak'))`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO fin_condition_gap(kind,venue,surface,payload)
		VALUES('permission','okx','volsell-fail-streak',$1::jsonb)
		ON CONFLICT (venue,surface,kind) WHERE state='pending'
		DO UPDATE SET requested_at=now(),payload=EXCLUDED.payload`, string(payload)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('research/grammar','升件','T-0356',$1)`, detail); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	r.rej.Inc("volsell_fail_streak_retire")
	r.volSellWasRetired, r.volSellSkip = true, true
	if r.alert != nil {
		r.alert.Alert(ctx, "volsell_auto_retire", detail)
	}
	return nil
}
