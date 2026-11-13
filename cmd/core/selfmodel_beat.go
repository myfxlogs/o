// selfmodel_beat.go —— 自我模型器官帧（自我意识的「能力账」翼，工单#13540 同源）。
// self_aware 帧答「我现在动不动/卡不卡」=状态层;本帧答「我擅长什么/哪类活
// 总砸」=能力层——人格=对自己能力边界的稳定认识,元认知底座。
//
// 域键=工单 scope= 源升件域,无 scope 兜 seat（业主令票/外席票自带域义）。
// 域面三数(30d窗):票=立票量,逾=开逾期+销逾期,燃=stuck_esc 销案后同域
// 复燃次数(判词不实/修错界的自证)。尾行=会话果败比+求助值守数——
// 「我干活行不行」直接事实。
// 量表件只述事实不定性;值指纹闸 20h——域集/数变=新现实重报,同纹抑。

package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const selfModelEvery = 20 * time.Hour

// selfModelBeat 自模账帧——30d 域面成败三数 + 会话果败尾,日行照。
func (c coreService) selfModelBeat(ctx context.Context) {
	// 域面成败——票量/逾期(开逾∨销逾)/销票,top6 按票量序。
	type domRow struct {
		dom   string
		n, od int
	}
	var doms []domRow
	rows, err := c.pool.Query(ctx, `WITH tk AS (
		SELECT t.id, t.ts, COALESCE(substring(t.note from 'scope=([^ ,，]+)'), t.seat) dom,
			t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' ddl
		FROM ops_event t WHERE t.event='工单' AND t.ts>now()-interval '30 days'),
	cl AS (SELECT t.id, min(x.ts) cts FROM tk t
		JOIN ops_event x ON x.event='处置' AND x.ts>t.ts
			AND x.note ~ ('工单#'||t.id::text||'([^0-9]|$)')
			AND x.note NOT LIKE '%挂起条件=%'
			AND x.note !~ ('工单#'||t.id::text||' ?接续')
		GROUP BY t.id)
		SELECT t.dom, count(*),
			count(*) FILTER (WHERE (c.cts IS NULL AND t.ddl<now()) OR c.cts>t.ddl),
			count(*) FILTER (WHERE c.cts IS NOT NULL)
		FROM tk t LEFT JOIN cl c ON c.id=t.id
		GROUP BY t.dom ORDER BY 2 DESC LIMIT 6`)
	if err == nil {
		for rows.Next() {
			var d domRow
			var cl int
			if rows.Scan(&d.dom, &d.n, &d.od, &cl) == nil {
				doms = append(doms, d)
			}
		}
		rows.Close()
	}
	// 域面复燃——stuck_esc note v= 前缀内 scope#id 对(销案后再火=自证不实)。
	refire := map[string]int{}
	r2, err := c.pool.Query(ctx, `SELECT m[1], count(DISTINCT e.id) FROM ops_event e,
		LATERAL regexp_matches(e.note,'([a-zA-Z0-9_-]+)#[0-9]+','g') m
		WHERE e.event='升件' AND e.esc_scope='stuck_esc' AND e.ts>now()-interval '30 days'
		GROUP BY 1`)
	if err == nil {
		for r2.Next() {
			var k string
			var v int
			if r2.Scan(&k, &v) == nil {
				refire[k] = v
			}
		}
		r2.Close()
	}
	// 会话果败尾——唤知环自身健康(果行 rc!=0=干活干败)+求助值守数。
	var good, bad, help int
	_ = c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE event='唤起果' AND note ~ 'rc=0([^0-9]|$)'),
		count(*) FILTER (WHERE event IN ('唤起果','唤起败')
			AND (event='唤起败' OR note !~ 'rc=0([^0-9]|$)')),
		(SELECT count(*) FROM ops_event WHERE event='求助值守'
			AND ts>now()-interval '30 days')
		FROM ops_event WHERE event IN ('唤起果','唤起败')
		AND ts>now()-interval '30 days'`).Scan(&good, &bad, &help)

	var seg, fpIn []string
	if len(doms) == 0 {
		seg = append(seg, "域=?")
	} else {
		for _, d := range doms {
			s := fmt.Sprintf("%s 票%d逾%d燃%d", d.dom, d.n, d.od, refire[d.dom])
			seg = append(seg, s)
			fpIn = append(fpIn, s)
		}
	}
	fpIn = append(fpIn, fmt.Sprintf("会话%d果%d败", good, bad), fmt.Sprintf("求%d", help))
	fp := fpHash(fpIn)
	if !gaugeDue(ctx, c, "self_model", fp, selfModelEvery) {
		return
	}
	note := fmt.Sprintf("v=%s 自模账: %s | 会话=%d果%d败 求=%d——自我模型器官帧(能力账翼;30d窗域面成败,只述事实不定性)",
		fp, strings.Join(seg, " "), good, bad, help)
	_, _ = c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','self_model',$1)`, note)
}
