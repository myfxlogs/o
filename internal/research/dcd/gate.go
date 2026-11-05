// gate.go —— 业主资产域闸+复役条件书机判（T-0353 件③④）：
//
//	件④ 锁仓币≠USDT=业主资产域——dcd_call 锁基币（BTC/ETH…）、非 USDT
//	     计价 put 同律，proposed 分流 fin_condition_gap(kind=holding) 请求
//	     行「业主批了才 arm」，不进自动提案链；USDT 腿（dcd_put 计价币）
//	     =基金钱常径不受闸。
//	件③  enum 臂连续读败/错件 >N 拍未自愈=自停+升件（ops_event 升件行+
//	     fin_condition_gap(kind=permission) 复役请托+业主通道 Alert）；
//	     config 翻位（research.dcd.enabled=false）留 SG 正本人工艺——
//	     research_rw 无 app_config 写权（fin_0086 授权面即全表，越权不授），
//	     机判=检测+自停+升件三件齐，复役动作=升件行处置项。
package dcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/rejects"
)

// OwnerDomain 业主资产域判定（判据=产品锁仓币——lockedCcyOf 现算非 Eval
// 构造快照，strangle 复合键等未全量构造 Eval 的落卡径同样可靠）。
func OwnerDomain(e Eval) bool { return lockedCcyOf(e.Product) != "USDT" }

// RequestOwnerDomain 分流落盒公共径（dcd persistEval/文法 persistLegEval
// 两落卡点共用）：fin_condition_gap(kind=holding) pending 去重（venue+
// surface+kind）——同域未决不重行，payload 随最新候选刷新；fin=nil=写径缺
// 显形 fail-closed（提案不落=红线保住，缺口经 rejects 计数可见）。
func RequestOwnerDomain(ctx context.Context, fin *pgxpool.Pool, e Eval,
	rej *rejects.Door, logPrefix string) error {
	locked := lockedCcyOf(e.Product)
	if fin == nil {
		rej.Inc("gap_writer_missing")
		log.Printf("WARN [%s] %s 业主资产域分流写径缺（fin 池未挂——提案不落待装配补件）",
			logPrefix, e.Product.ProductID)
		return fmt.Errorf("SG fin writer missing; owner-domain proposal withheld")
	}
	payload, _ := json.Marshal(map[string]any{
		"ask":        "业主资产域动用批核——锁仓币非基金 USDT 域（T-0353 件④，批了才 arm）",
		"productId":  e.Product.ProductID,
		"uly":        e.Product.Uly,
		"template":   e.Template,
		"locked_ccy": locked,
		"apr":        e.APR,
	})
	_, err := fin.Exec(ctx, `INSERT INTO fin_condition_gap(kind,venue,surface,payload)
		VALUES('holding','okx','dcd',$1::jsonb)
		ON CONFLICT (venue,surface,kind) WHERE state='pending'
		DO UPDATE SET requested_at=now(),payload=EXCLUDED.payload`, string(payload))
	if err != nil {
		rej.Inc("gap_write_err")
		return err
	}
	rej.Inc("owner_domain_gap")
	log.Printf("[%s] %s 业主资产域分流——条件请求落盒不进提案链（locked=%s apr=%.4f）",
		logPrefix, e.Product.ProductID, locked, e.APR)
	return nil
}

// autoRetired 从 fin 正本读取最近一次机器退役请求：pending/rejected/dismissed
// 都保持停扫；仅业主明确 fulfilled 才解除。条件请求盒是跨进程重启的持久闸。
func autoRetired(ctx context.Context, fin *pgxpool.Pool) (bool, error) {
	if fin == nil {
		return true, fmt.Errorf("SG fin writer missing")
	}
	var state string
	err := fin.QueryRow(ctx, `SELECT state FROM fin_condition_gap
		WHERE kind='permission' AND venue='okx' AND surface='dcd-fail-streak'
		ORDER BY requested_at DESC,id DESC LIMIT 1`).Scan(&state)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return true, err
	}
	return state != "fulfilled", nil
}

// scanTracked 连败计数壳（件③机判——真实成功扫描拍清零；节流空拍、缺读件
// 不算自愈；触发严格为 >N）。
func (r *Runner) scanTracked(ctx context.Context) error {
	if enabled, err := r.cfg.GetBool("research.dcd.enabled"); err == nil && !enabled {
		r.failStreak = 0
		r.retired = false
		return nil
	}
	stopped, err := autoRetired(ctx, r.fin)
	if err != nil {
		r.rej.Inc("dcd_retire_state_read_err")
		return fmt.Errorf("dcd 复役状态读败 fail-closed: %w", err)
	}
	if stopped {
		r.retired = true
		return nil
	}
	if r.retired {
		// 条件请求已由业主 fulfilled：新一轮从零计败，旧窗口不带入复役后。
		r.retired = false
		r.failStreak = 0
	}
	before := r.lastOK
	err = r.scan(ctx)
	if err == nil && r.lastOK.After(before) {
		r.failStreak = 0
		return nil
	}
	if err == nil {
		return nil // 节流空拍/缺读件不算自愈成功，不清 streak
	}
	r.failStreak++
	if retireErr := r.maybeAutoRetire(ctx, err); retireErr != nil {
		return fmt.Errorf("dcd scan failed (%v); auto-retire persistence failed: %w", err, retireErr)
	}
	return err
}

// maybeAutoRetire 复役条件书触发器：N=research.dcd.fail_streak_n（缺键=
// 机判不启用 fail-closed——判据缺位禁虚构）；触发条件严格为「>N 拍」。
// 条件请求+升件事件同事务落 SG，pending 请求是跨重启的持久停扫闸；业主
// fulfilled 才复扫。不授 research_rw app_config 写权、不自动改正本配置。
func (r *Runner) maybeAutoRetire(ctx context.Context, last error) error {
	if r.retired {
		return nil
	}
	var n float64
	if v, err := r.cfg.GetFloat("research.dcd.fail_streak_n"); err == nil {
		n = v
	}
	if n <= 0 || n != float64(int(n)) || float64(r.failStreak) <= n {
		return nil
	}
	if r.fin == nil {
		r.rej.Inc("dcd_retire_writer_missing")
		return fmt.Errorf("SG fin writer missing; cannot persist dcd retirement")
	}
	detail := fmt.Sprintf("dcd 枚举臂连败 %d 拍自停（>N=%.0f——复役条件书触发）最后错: %v",
		r.failStreak, n, last)
	log.Printf("WARN [research/dcd] %s", detail)
	payload, _ := json.Marshal(map[string]any{
		"auto_retire": "research.dcd.fail_streak_n",
		"ask":         "dcd 枚举臂连败自停——核因后由业主明确复役（SG fin 正本；fulfilled 后方恢复扫描）",
		"streak":      r.failStreak,
		"n":           n,
		"last_err":    last.Error(),
	})
	tx, err := r.fin.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('T-0353:dcd-fail-streak'))`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO fin_condition_gap(kind,venue,surface,payload)
		VALUES('permission','okx','dcd-fail-streak',$1::jsonb)
		ON CONFLICT (venue,surface,kind) WHERE state='pending'
		DO UPDATE SET requested_at=now(),payload=EXCLUDED.payload`, string(payload)); err != nil {
		return fmt.Errorf("dcd retire condition request: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('research/dcd','升件','T-0353',$1)`, detail); err != nil {
		return fmt.Errorf("dcd retire escalation event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	r.retired = true
	r.rej.Inc("dcd_fail_streak_retire")
	if r.alert != nil {
		r.alert.Alert(ctx, "dcd_auto_retire", detail)
	}
	return nil
}
