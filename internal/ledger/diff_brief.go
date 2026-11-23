// diff_brief.go —— 账实差告警初诊装配（业主令 2026-11-10——「发现差额只报
// 发现不答钱从哪来」病灶：告警须载三事实——24h 出入金标识（外部移动钦定
// marker）、72h 簿记构成、venue-账方向。零出入金=簿记域机器定性倾向
// （残差/费用/尘归豁免+诊断径），在账=外部移动嫌疑明呈业主核验位——
// 告警即初诊非裸哨；定性/豁免/根修仍归唤知会话径不变。
package ledger

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ledgerDiffBrief(ctx context.Context, pool *pgxpool.Pool, venue, ccy string,
	mtmFn func(context.Context) (map[[2]string]float64, error)) string {
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
	var exN int
	var exSum float64
	var exWhy string
	_ = pool.QueryRow(ctx, `SELECT count(*),COALESCE(sum(amount_usd),0),
		COALESCE(string_agg(left(reason,28),'｜'),'')
		FROM fin_known_residual WHERE venue=$1 AND ccy=$2 AND expires_at>now()`,
		venue, ccy).Scan(&exN, &exSum, &exWhy)
	if exN > 0 {
		comp += fmt.Sprintf("；豁免在账$%.2f(%d行: %s)", exSum, exN, exWhy)
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
