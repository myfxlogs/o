// basis.go —— basis_ann 季度基差年化（T-0139 起；T-0146 频道方言归并读）。
// T-0164 血统闸：行 ts=最旧上游观测（futures mark/spot_last 取旧）——上游陈→
// 行即陈→消费 Fresh 闸视缺席（D-0022「信号陈=不存在」钉到血统层）。
package sigdist

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// basisAnn 季度基差年化（bps）：mark-price（md_mark_index channel='mark_price'
// FUTURES inst）对同基币 spot_last——(mark-index)/index×365/到期日数×10⁴。
// 无到期解析的 inst 跳过如实。
func (r *Runner) basisAnn(ctx context.Context) error {
	// T-0146 频道方言归并：f 侧跨 'mark_price'/'mark-price' 双频道取最新
	// （桥写/SG 兜底双键域并存——freshness-priority 读）。
	rows, err := r.pool.Query(ctx, `WITH mk AS (
			SELECT inst_id, px, ts, row_number() OVER
				(PARTITION BY inst_id ORDER BY ts DESC) rn
			FROM md_mark_index
			WHERE channel IN ('mark_price','mark-price') AND inst_id ~ '-[0-9]{6}$'
		)
		SELECT mk.inst_id, mk.px, s.px, mk.ts, s.ts FROM mk
		JOIN md_mark_index s
		  ON s.channel='spot_last'
		 AND s.inst_id = regexp_replace(mk.inst_id, '-[0-9]{6}$', '')
		WHERE mk.rn=1`)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now()
	n := 0
	for rows.Next() {
		var instID string
		var mark, index float64
		var markTs, spotTs time.Time
		if err := rows.Scan(&instID, &mark, &index, &markTs, &spotTs); err != nil {
			return err
		}
		m := datedFutRe.FindStringSubmatch(instID)
		if m == nil || index == 0 {
			continue // 无到期解析/零指数——跳过如实
		}
		exp, err := time.Parse("060102", m[2])
		if err != nil {
			continue // 到期日坏形——跳过如实
		}
		days := exp.Sub(now).Hours() / 24
		if days <= 0 {
			continue // 到期邻近/已过——基差年化无意义
		}
		bps := (mark - index) / index * 365 / days * 10000
		meta, _ := json.Marshal(map[string]any{
			"expiry": exp.Format("2006-01-02"), "mark": mark, "index": index,
			"days": int(days) + 1})
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
			VALUES('basis_ann',$1,$2,$3::jsonb,$4)
			ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val,
			meta=EXCLUDED.meta, ts=EXCLUDED.ts`,
			instID, bps, string(meta), leastTs(markTs, spotTs)); err != nil {
			return err
		}
		n++
	}
	log.Printf("[sigdist] basis_ann 拍：%d 行", n)
	return nil
}
