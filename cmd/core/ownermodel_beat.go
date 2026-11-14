// ownermodel_beat.go —— 懂你模型帧（棒4「懂你」器官）:owner_intent
// 从静态文本升机产模型——请托结局谱(fin_condition_gap per-kind
// 成/否/销/pending+中位解期)∪点破域谱(业主升件 esc_class 30d 谱)
// 聚合成帧,量表/owner_model 指纹拍(谱变=新现实)。消费面:condreq
// 起草前问模型(少空转)、独白读件、会话懂你判词材料。
// 副臂探件 reask_denied:pending 请托与≥2次已否同键史=重复问业主已否
// 件——升件提示取同键否案证改口径∨销,欲再请须载新事证。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (c coreService) ownerModelBeat(ctx context.Context, deadSince map[string]time.Time) {
	// 副臂探件每拍——重复问业主已否件
	var reask int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM fin_condition_gap p
		WHERE p.state='pending' AND (
			SELECT count(*) FROM fin_condition_gap o
			WHERE o.kind=p.kind AND o.venue IS NOT DISTINCT FROM p.venue
				AND o.surface IS NOT DISTINCT FROM p.surface
				AND o.state IN ('rejected','dismissed'))>=2`).Scan(&reask); err == nil {
		c.escEdgeUV(ctx, deadSince, "懂你", "reask_denied", "@", reask > 0, func() string {
			return fmt.Sprintf("懂你模型实证——%d 件 pending 请托与≥2次已否同键史=重复问业主已否件(取同键否案证改口径∨销;欲再请须载新事证非原样重递)", reask)
		})
	}

	// 帧件:请托结局谱+点破域谱指纹拍
	type kindRow struct {
		kind            string
		n, ok, rej, dis int
		pend            int
		latH            float64
	}
	var kinds []kindRow
	rows, err := c.pool.Query(ctx, `SELECT kind, count(*),
		count(*) FILTER (WHERE state='fulfilled'),
		count(*) FILTER (WHERE state='rejected'),
		count(*) FILTER (WHERE state='dismissed'),
		count(*) FILTER (WHERE state='pending'),
		COALESCE(round(percentile_cont(0.5) WITHIN GROUP
			(ORDER BY EXTRACT(EPOCH FROM resolved_at-requested_at)/3600.0)::numeric,1),0)
		FROM fin_condition_gap GROUP BY kind ORDER BY count(*) DESC`)
	if err == nil {
		for rows.Next() {
			var k kindRow
			if rows.Scan(&k.kind, &k.n, &k.ok, &k.rej, &k.dis, &k.pend, &k.latH) == nil {
				kinds = append(kinds, k)
			}
		}
		rows.Close()
	}
	var ask []string
	for _, k := range kinds {
		ask = append(ask, fmt.Sprintf("%s n=%d 成%d/否%d/销%d 解中位%.1fh pend%d",
			k.kind, k.n, k.ok, k.rej, k.dis, k.latH, k.pend))
	}
	var poke []string
	rows, err = c.pool.Query(ctx, `SELECT COALESCE(esc_class,'-'), count(*) FROM ops_event
		WHERE seat='业主' AND event='升件' AND ts>now()-interval '30 days'
		GROUP BY 1 ORDER BY 2 DESC LIMIT 5`)
	if err == nil {
		for rows.Next() {
			var cls string
			var n int
			if rows.Scan(&cls, &n) == nil {
				poke = append(poke, fmt.Sprintf("%s×%d", cls, n))
			}
		}
		rows.Close()
	}
	askS, pokeS := strings.Join(ask, "|"), strings.Join(poke, "|")
	fp := fpHash([]string{askS, pokeS})
	if !gaugeDue(ctx, c, "owner_model", fp, 20*time.Hour) {
		return
	}
	note := fmt.Sprintf("v=%s 懂你: 请托[%s] 点破域30d[%s]",
		fp, orNone(askS), orNone(pokeS))
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('探测席','量表','owner_model',$1)`, note); err != nil {
		return
	}
}

func orNone(s string) string {
	if s == "" {
		return "无"
	}
	return s
}
