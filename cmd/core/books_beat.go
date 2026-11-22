// books_beat.go —— 账房日对账量表探（业主令 11-12：账房器官立件）：
// ΔNAV 三桶归因——外部净流 + 盈亏 + 未释残差。
//
//	净流=fin_venue_bill deposit/withdraw（外部进出,内部 type=1 划转不入）;
//	盈亏=fin_cashflow 六族（同 profitDailyBeat 钦定口径,Earn 息不入账已剔）;
//	残差=Δ-净流-盈亏——|残差|>阈=账面闹鬼→escEdgeUV 升件 class='对账'
//	scope='nav_residual'（残差值为指纹:同值已阅不火,异值=新鬼）。
//
// mt5 侧无 bill 源在案——其进出落残差桶,note 诚实注记不虚构。
// 分腿归因 v1 略：equity 腿标 ccy 非 venue（report_daily 按币簿记）,
// 腿差读件缺=诚实略不硬拆。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// booksEvery 对账日帧（量表族同拍——指纹异即重报,同纹 20h 续）。
const booksEvery = 20 * time.Hour

// booksGhostUSD 未释残差硬阈——|残差|超此=账面闹鬼升件（实测底噪
// Earn/估值漂<$50,阈放宽十倍防误报）。
const booksGhostUSD = 200.0

func (c coreService) booksBeat(ctx context.Context, deadSince map[string]time.Time) {
	// ① 锚点：NAV 今帧+≥20h 前末帧（ccy='NAV' 聚合行——读件缺/陈
	//   归 navStale 探,本件守「有账才对」不判断供）。
	var navNow, navPrev float64
	var tsNow, tsPrev time.Time
	if err := c.pool.QueryRow(ctx, `SELECT equity,ts FROM md_equity_curve
		WHERE ccy='NAV' ORDER BY ts DESC LIMIT 1`).Scan(&navNow, &tsNow); err != nil {
		return
	}
	if err := c.pool.QueryRow(ctx, `SELECT equity,ts FROM md_equity_curve
		WHERE ccy='NAV' AND ts<=$1 ORDER BY ts DESC LIMIT 1`,
		tsNow.Add(-booksEvery)).Scan(&navPrev, &tsPrev); err != nil {
		return // 无前帧=窗不成立（首拍/断供窗让位）
	}
	// ② 窗内三桶。
	var flow, earned float64
	_ = c.pool.QueryRow(ctx, `SELECT coalesce(sum(bal_chg),0) FROM fin_venue_bill
		WHERE type IN ('deposit','withdraw') AND ts>$1 AND ts<=$2`,
		tsPrev, tsNow).Scan(&flow)
	_ = c.pool.QueryRow(ctx, `SELECT coalesce(sum(amount),0) FROM fin_cashflow
		WHERE venue<>'sim' AND occurred_at>$1 AND occurred_at<=$2
		AND kind IN ('fee','funding_settle','trade_pnl','rollback_cost','spot_buy','spot_sell')`,
		tsPrev, tsNow).Scan(&earned)
	delta := navNow - navPrev
	resid := delta - flow - earned
	fp := fmt.Sprintf("%.0f|%.0f", delta, resid)
	if !gaugeDue(ctx, c, "books_daily", fp, booksEvery) {
		return
	}
	// ③ 鬼账升件（残差超阈——探针只述实数,定性归唤知会话）。
	//    残差透镜（工单O-27665 取舍b）：阈内不扰，阈外 note 附未载类
	//    明细+venue 腿跨窗差——「残差为何残」可读非黑盒，口径不动。
	ghost := resid > booksGhostUSD || resid < -booksGhostUSD
	lens := ""
	if ghost {
		lens = c.residLens(ctx, tsPrev, tsNow)
	}
	note := booksNote(fp, delta, flow, earned, resid, tsPrev, tsNow, lens)
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','books_daily',$1)`,
		note); err != nil {
		return
	}
	c.escEdgeUV(ctx, deadSince, "对账", "nav_residual", fmt.Sprintf("%.0f", resid), ghost, func() string {
		return fmt.Sprintf("账面闹鬼：ΔNAV%+.2f 净流%+.2f 盈亏%+.2f 未释%+.2f——净值跳变拆三桶后残差超阈（定性归唤知会话;可能因=快照缺腿/mt5离线进出/未载费类）%s",
			delta, flow, earned, resid, lens)
	})
}

// residLens 残差透镜——阈外才取（O-27665）：
// ① fin_cashflow 未入桶类行明细（kind/venue/amt——六族盈亏桶+bill净流
//
//	桶之外的外部进出/息/调整类，owner_in±10k 类即此面）；
//
// ② venue 腿跨窗集差（md_equity_curve venue/ccy 腿：窗内首见=+新腿、
//
//	窗前有账窗内断=−退腿——腿集差=残差候选因）。
//
// 读败=空串（透镜件不阻主径——fail-open 同本文件惯例）。
func (c coreService) residLens(ctx context.Context, tsPrev, tsNow time.Time) string {
	var unloaded, legDiff string
	rows, err := c.pool.Query(ctx, `SELECT kind||'/'||venue||' '||round(sum(amount),2)||'x'||count(*)
		FROM fin_cashflow
		WHERE venue<>'sim' AND occurred_at>$1 AND occurred_at<=$2
		  AND kind NOT IN ('fee','funding_settle','trade_pnl','rollback_cost','spot_buy','spot_sell')
		GROUP BY kind,venue ORDER BY abs(sum(amount)) DESC LIMIT 5`, tsPrev, tsNow)
	if err == nil {
		var segs []string
		for rows.Next() {
			var s string
			if rows.Scan(&s) == nil {
				segs = append(segs, s)
			}
		}
		rows.Close()
		if len(segs) > 0 {
			unloaded = strings.Join(segs, ";")
		}
	}
	_ = c.pool.QueryRow(ctx, `WITH w AS (
			SELECT DISTINCT venue||'/'||ccy AS leg FROM md_equity_curve
			WHERE ts>$1 AND ts<=$2 AND ccy<>'NAV'),
		p AS (SELECT DISTINCT venue||'/'||ccy AS leg FROM md_equity_curve
			WHERE ts<=$1 AND ccy<>'NAV')
		SELECT coalesce((SELECT string_agg('+'||leg,',') FROM (SELECT leg FROM w EXCEPT SELECT leg FROM p) n),'')
			||coalesce((SELECT string_agg('-'||leg,',') FROM (SELECT leg FROM p EXCEPT SELECT leg FROM w) o),'')`,
		tsPrev, tsNow).Scan(&legDiff)
	return booksLensText(unloaded, legDiff)
}

// booksLensText 透镜文纯函数——两段有则拼，无则空串（供 shape 测）。
func booksLensText(unloaded, legDiff string) string {
	var seg []string
	if unloaded != "" {
		seg = append(seg, "未载类["+unloaded+"]")
	}
	if legDiff != "" {
		seg = append(seg, "腿差["+legDiff+"]")
	}
	if len(seg) == 0 {
		return ""
	}
	return "——残差透镜:" + strings.Join(seg, ";")
}

// booksNote 对账账页文——三桶式+窗+阈注+残差透镜尾段（lens 空=原样，纯函数供 shape 测）。
func booksNote(fp string, delta, flow, earned, resid float64, tsPrev, tsNow time.Time, lens string) string {
	return fmt.Sprintf("v=%s 账房对账: ΔNAV%+.2f = 净流%+.2f + 盈亏%+.2f + 未释%+.2f（窗 %s→%s %.0fh）——|未释|>$%.0f=账面闹鬼升件对账/nav_residual;mt5无bill源其进出落残差桶%s",
		fp, delta, flow, earned, resid,
		tsPrev.UTC().Format("01-02 15:04"), tsNow.UTC().Format("01-02 15:04"),
		tsNow.Sub(tsPrev).Hours(), booksGhostUSD, lens)
}
