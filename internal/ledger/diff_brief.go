// diff_brief.go —— 账实差告警初诊装配（业主令 2026-11-10——「发现差额只报
// 发现不答钱从哪来」病灶：告警须载三事实——24h 出入金标识（外部移动钦定
// marker）、72h 簿记构成、venue-账方向。零出入金=簿记域机器定性倾向
// （残差/费用/尘归豁免+诊断径），在账=外部移动嫌疑明呈业主核验位——
// 告警即初诊非裸哨；定性/豁免/根修仍归唤知会话径不变。
package ledger

import (
	"context"
	"fmt"
	"log"
)

func ledgerDiffBrief(ctx context.Context, r *Runner, venue, ccy string) string {
	pool := r.pool
	mtmFn := r.mtm
	var depo, wdr float64
	_ = pool.QueryRow(ctx, `SELECT COALESCE(sum(amount) FILTER (WHERE kind='deposit'),0),
		COALESCE(-sum(amount) FILTER (WHERE kind='withdraw'),0) FROM fin_cashflow
		WHERE venue=$1 AND ccy=$2 AND ts>now()-interval '24 hours'`,
		venue, ccy).Scan(&depo, &wdr)
	var legs string
	_ = pool.QueryRow(ctx, `SELECT COALESCE(string_agg(kind||' $'||to_char(amt,'FM999999990.00'),' / '),'')
		FROM (SELECT kind,sum(amount) amt FROM fin_cashflow
			WHERE venue=$1 AND ccy=$2 AND ts>now()-interval '72 hours'
			GROUP BY kind ORDER BY sum(abs(amount)) DESC LIMIT 3) t`,
		venue, ccy).Scan(&legs)
	// 工单#35226 验收④：复报 brief 须含全成分——净额构成五件（现金流/
	// 停泊/dcd基线/原生基线/mtm）+在账豁免逐件定性，差额可逐件归名非裸差。
	var cf float64
	_ = pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE venue=$1 AND ccy=$2 AND kind NOT IN ('earn_in','earn_out')`,
		venue, ccy).Scan(&cf)
	var mtm float64
	if mtmFn != nil {
		if mv, err := mtmFn(ctx); err == nil {
			mtm = mv[[2]string{venue, ccy}]
		}
	}
	comp := fmt.Sprintf("净额构成: 现金流%.4f", cf)
	if _, parked, dcdB, natB, err := ReconSets(ctx, pool); err == nil {
		k := [2]string{venue, ccy}
		comp += fmt.Sprintf("+停泊%.4f+dcd基线%.4f+原生基线%.4f+mtm%.4f",
			parked[k], dcdB[k], natB[k], mtm)
	} else {
		comp += "+构成读败(停泊/dcd/原生/mtm)"
	}
	// O-36327：豁免额=静态行+resolver 跟踪行现算——成分段显逐行
	// mode 定性（现算/锚/破带归零/回退），与 run.go 判据同源 trackedExempt。
	_, exBrief := r.trackedExemptBrief(ctx, venue, ccy)
	var exN int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_known_residual
		WHERE venue=$1 AND ccy=$2 AND expires_at>now()`, venue, ccy).Scan(&exN)
	if exN > 0 {
		comp += "；" + exBrief
	} else {
		comp += "；豁免在账=0"
	}
	return diffBriefText(depo, wdr, legs) + "；" + comp
}

func diffBriefText(depo, wdr float64, legs string) string {
	external := "24h无出入金记录→非外部移动（机器定性域：簿记残差/费用/尘）"
	if depo > 0 || wdr > 0 {
		external = fmt.Sprintf("24h出入金在账: 入+$%.2f 出-$%.2f→含外部移动嫌疑（业主可核）", depo, wdr)
	}
	if legs == "" {
		legs = "近72h无现金流行"
	}
	return fmt.Sprintf("%s；72h簿记构成: %s", external, legs)
}

// alertLedgerDiff 账实差告警+升件双写（工单O-34997 双载荷落地点：
// payload=机读诊断原文只喂唤知分诊径/指纹；plain=业主面人话摘要——
// 黑话词 venue读/账本净/容差/豁免/簿记构成/recon_adjust/spot_* 永不出
// 业主面，桩测词表断言在案）。over=USD折算后判定（run.go 判据不变）。
func (r *Runner) alertLedgerDiff(ctx context.Context, venue, ccy string,
	eq, net, tol, exempt, diffUSD float64, over bool, parts []BalPart) {
	if !over {
		return
	}
	r.rej.Inc("diff_alert")
	// 业主令 2026-11-10：告警载初诊（出入金标识+簿记构成——
	// 「只报发现不答钱从哪来」病灶，见 diff_brief.go）。
	brief := ledgerDiffBrief(ctx, r, venue, ccy)
	// O-36964：升件 note 头部白话账段（置于机读段前——业主三秒读懂
	// 差多少/差由哪几块组成/是否已豁免/要不要人动手；机读底稿段原样保留）。
	human := r.ledgerDiffHuman(ctx, venue, ccy, eq, net, diffUSD, tol, exempt, parts)
	raw := Diff(net, eq)
	dir := "少"
	if net > eq {
		dir = "多"
	}
	plain := fmt.Sprintf("账上记的钱比交易所实际显示%s $%.2f。初步判断：大概率是已知的计量口径不同，机器定性中——钱没丢，不用动。", dir, diffUSD)
	r.al.AlertPlain(ctx, "ledger_diff_alert",
		fmt.Sprintf("账实差 $%.2f（容差 $%.2f +已定性豁免 $%.2f；venue读 %.4f vs 账本净 %.4f）→ %s",
			diffUSD, tol, exempt, eq, net, brief), plain)
	// 机读进水口：账实差=对账类升件（原径仅 Feishu 人推——
	// 「探针只上 UI 不落 ops_event=病灶」同病）。dedup=未销才抑：
	// 同窗内有未销升件不暴,销案后差仍在/新差=新事证重升。
	scope := fmt.Sprintf("ledger_diff_%s_%s", venue, ccy)
	var open int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e
		WHERE e.event='升件' AND e.esc_class='对账' AND e.esc_scope=$1
		AND e.ts>now()-interval '24 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event x2 WHERE x2.ts>e.ts
			AND x2.note LIKE '%升件#%'
			AND (x2.note ~ ('升件#'||e.id::text||'([^0-9]|$)')
				OR EXISTS (SELECT 1 FROM regexp_matches(x2.note,'升件#(\d+)-(\d+)','g') AS m(mm)
					WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint)))`, scope).Scan(&open); err == nil && open == 0 {
		_, _ = r.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			VALUES('探测席','升件','',$1,'对账',$2)`,
			fmt.Sprintf("对账/%s: %s——账实差 $%.2f（容差 $%.2f+已定性豁免 $%.2f；venue 现读 %.4f vs 账本净 %.4f）→ %s——定性归唤知会话（runbook 对账档:tools/reconview 自助面→豁免行/根修）",
				scope, human, raw, tol, exempt, eq, net, brief), scope)
	}
	log.Printf("ALERT [ledger] 对账差 %s/%s venue=%.4f ledger=%.4f（容差 %.2f）", venue, ccy, eq, net, tol)
}
