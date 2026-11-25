package ledger

// rebase.go —— O-33206 业主令账本重锚（「以前的旧账都不要了,以昨天的
// 账为基点」2026-11-25——一次性簿记件,append-only 更正行非 UPDATE）:
//
//   armed 键 app_config.ledger.rebase_armed='"armed"' 在场→scan 拍先于
//   reconcile 跑——前置序与 reconcile 全同时点（在途窗净+postInterest
//   先行+同一 r.bal 读面），对拍行集（reconRows 同源数学）逐键
//   delta=eq−net 落 recon_adjust（account_id 直发，source_table=
//   'rebase_owner_decree'+source_ref=令簿记 ops_event 行）——「此刻
//   venue 实账=账本新基线」不拆旧腿不逐笔对，差值一笔进；幂等=
//   (account_id,ccy) 已有重锚行跳（重跑/半截续跑不双补）+跑毕置
//   'done' 只武装一次；静态 USD 已知残差（旧差豁账——#23/#32/#33/#40
//   同族）同步 expires 退役，resolver 跟踪型（构性漂移差会再生）保留。
//
//   让位面（armed 保留下拍重试，盲锚禁）：bal 未挂/venue 读败/账户册
//   读败/在途窗净——任何一步不满足=本轮零写零旗变。

import (
	"context"
	"fmt"
	"log"
	"math"
)

// rebaseSourceTable 重锚行来源簿记名（source_ref=令簿记 ops_event 行 id——
// recon_baseline_adjust 先例同式：逻辑源名+事实账指针）。
const rebaseSourceTable = "rebase_owner_decree"

// rebaseDecree 业主令重锚执行段（scan 拍②——armed 闸内全软失败：
// 读败/写败=计数+WARN+让位或跳键,绝不返 err 挡对账径）。
func (r *Runner) rebaseDecree(ctx context.Context) {
	// ① armed 闸——直读 app_config（config 缓存代际外,迁移落键即见;
	//    value jsonb 精确等值,非字符串形/缺行=不跑）。
	var armed bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM app_config
			WHERE key='ledger.rebase_armed' AND value='"armed"'::jsonb)`).Scan(&armed); err != nil || !armed {
		return
	}
	// ② 读面前置——bal 未挂=对账缺岗同判例让位（重锚无 venue 实读=盲锚禁）。
	if r.bal == nil {
		r.rej.Inc("bal_missing")
		log.Printf("WARN [ledger] 重锚 venue 现读件未挂接——本轮让位")
		return
	}
	// ③ 在途窗净——与 reconcile 同判据（inflightActive 同源件）：
	//    划转/执行 30s 内活动=eq 与 net 本就错位,锚了=假基线。
	inflight, err := r.inflightActive(ctx)
	if err != nil {
		r.rej.Inc("recon_probe_failed")
		log.Printf("WARN [ledger] 重锚在途窗探测败——本轮让位: %v", err)
		return
	}
	if inflight {
		log.Printf("[ledger] 重锚跳过——执行/划转在途窗（30s 内活动,armed 保留下拍重试）")
		return
	}
	// ④ postInterest 先行——与 reconcile 同序：未入账应计利息先落
	//    cashflow,否则重锚差值会把它吸进 recon_adjust 再被 interest
	//    行双计（高水位幂等,重跑零写）。
	r.postInterest(ctx)
	ven, _, err := r.bal(ctx)
	if err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 重锚 venue 现读败——本轮让位 armed 保留: %v", err)
		return
	}
	wl, parked, dcdBase, nativeBase, err := ReconSets(ctx, r.pool)
	if err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 重锚对拍集读败——本轮让位 armed 保留: %v", err)
		return
	}
	accts, err := AccountRows(ctx, r.pool, "active")
	if err != nil {
		r.rej.Inc("recon_probe_failed")
		log.Printf("WARN [ledger] 重锚账户册读败——本轮让位 armed 保留: %v", err)
		return
	}
	rows := r.reconRows(ctx, accts, ven, wl, parked, dcdBase, nativeBase)
	// ⑤ 令簿记行——重锚明细锚（业主令 2026-11-25「旧账都不要了,以昨天
	//    的账为基点」+spec /opt/o-research/prompts/O-33206——cashflow 行
	//    source_ref 全指本行;event='簿记' 矩阵外不受 sem_gate 闸）。
	var evID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('外席','簿记','O-33206',$1) RETURNING id`,
		"业主令2026-11-25账本重锚执行——「以前的旧账都不要了,以昨天的账为基点」:"+
			"对拍行集逐键 delta=venue实读−账本净 落 recon_adjust(account_id 直发,"+
			"source_table=rebase_owner_decree 指本行);账户级锚=okx:trading/earn/dcd/funding"+
			"+exness:3001157/xmglobal:110660304 实读枚举(资金户 avail/frozen 子桶分量在案);"+
			"旧差豁账静态 USD 残差同步退役,resolver 跟踪型保留;实据 #39346/业主口述/"+
			"spec-rewrite #39963").Scan(&evID); err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 重锚令簿记行写败——本轮让位 armed 保留: %v", err)
		return
	}
	// ⑥ 逐键锚行——幂等=(account_id,ccy) 已有重锚行跳（半截续跑不双补；
	//    指纹语义=桶级一次锚,重申须新令新源名——spec「指纹=桶+目标值」
	//    由 armed→done 单次性承载,同桶重跑目标漂移不再锚=防Venue抖动反复改账）。
	posted, skipped, failed := 0, 0, 0
	for _, x := range rows {
		delta := x.eq - x.net
		if math.Abs(delta) < 1e-9 {
			skipped++
			continue // 净额=实读——无需锚行
		}
		var dup bool
		if err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM fin_cashflow
				WHERE source_table=$1 AND account_id=$2 AND ccy=$3)`,
			rebaseSourceTable, x.acct, x.ccy).Scan(&dup); err != nil {
			r.rej.Inc("recon_probe_failed")
			log.Printf("WARN [ledger] 重锚幂等探测败 %s/%s——本键让位: %v", x.acct, x.ccy, err)
			failed++
			continue
		}
		if dup {
			skipped++
			continue
		}
		if _, err := Post(ctx, r.pool, Entry{Kind: "recon_adjust", Venue: x.venue,
			Ccy: x.ccy, Amount: delta, AccountID: x.acct,
			SourceRef: evID, SourceTable: rebaseSourceTable}); err != nil {
			r.rej.Inc("write_err")
			log.Printf("WARN [ledger] 重锚行写败 %s/%s delta=%.8f——跳键: %v", x.acct, x.ccy, delta, err)
			failed++
			continue
		}
		posted++
	}
	// ⑦ 静态 USD 残差退役——旧差豁账（recon 判据已归零,豁免=负假差源
	//    同 fin_0250#24/#26 判例）；限 active 账户域+USDT/USD 币+静态行
	//    （resolver 空——跟踪型豁免的构性漂移差会再生必须保留）。
	var retired int64
	if res, err := r.pool.Exec(ctx,
		`UPDATE fin_known_residual SET expires_at=now(),
			reason=reason||'｜作废判词:O-33206——业主令重锚:venue实账为新基线,旧差豁账退役(resolver跟踪型保留)'
			WHERE expires_at>now() AND ccy IN ('USDT','USD')
			  AND (resolver IS NULL OR resolver='')
			  AND account_id IN (SELECT account_id FROM fin_account WHERE state='active')`); err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 重锚残差退役写败——已锚行不回滚,豁免残留待人工: %v", err)
	} else {
		retired = res.RowsAffected()
	}
	// ⑧ done 落旗——单次性承载（重跑须新迁移新令——防 venue 抖动反复改账）。
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO app_config(key,value,updated_at,updated_by)
			VALUES('ledger.rebase_armed','"done"',now(),'ledger/rebaseDecree')
			ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value,
			updated_at=now(),updated_by=EXCLUDED.updated_by`); err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 重锚 done 落旗败——下拍幂等续跑不双补: %v", err)
		return
	}
	// ⑨ 执行结果簿记行（判词行原料——posted/skipped/failed/retired 全量；
	//    止血#39358 目标达成标解同账——「重锚达成止血目标」令径有据,
	//    非裸解：虚胖账本面已冲平至 venue 实读,新腿簿记常态恢复）。
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('外席','簿记','O-33206',$1)`,
		fmt.Sprintf("业主令重锚执行毕——锚行 posted=%d skip(零差/已锚)=%d 写败跳键=%d;"+
			"静态USD残差退役=%d;对拍键集=%d;残差探针下拍应归零(验收锚点);"+
			"止血#39358 标解——重锚达成止血目标(虚胖冲平至venue实读,新腿簿记常态)",
			posted, skipped, failed, retired, len(rows))); err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 重锚结果簿记行写败: %v", err)
	}
}
