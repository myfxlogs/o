// owner_beat.go —— 业主令账+点破率量表探（escLoop 同拍载，业主令 2026-11-11）：
// 「业主令」一等工作面的账页——开账令件/点破率曲线日行照，进 relay digest
// （量表类自动面）。令件=seat='业主' event='升件' 行（#8957 先例钦定径：
// 业主口述令当场写升件进队,禁落处置note/draft孤儿——事实行不进工作队=
// 接令不成立,#9483 进水口触发档孤儿 20h 实证）。
// 点破率=自我意识收敛真度量：业主被迫发令移机的频率应单调下行,
// 曲线读法载 note 自释。
package main

import (
	"context"
	"fmt"
	"time"
)

const ownerBeatEvery = 20 * time.Hour

// ownerLedgerBeat 量表拍——业主令开账/点破率 7d×前7d→指纹闸→量表行。
func (c coreService) ownerLedgerBeat(ctx context.Context) {
	var openN, closed7d, spot7d, spotPrev7d int
	var oldest *time.Time
	// 开账令件=业主升件无引证行（处置/恢复 引用 升件#id——recoverNote 同键）。
	if err := c.pool.QueryRow(ctx, `SELECT count(*), min(e.ts) FROM ops_event e
		WHERE e.seat='业主' AND e.event='升件'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.event IN ('处置','恢复')
			AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)'))`).
		Scan(&openN, &oldest); err != nil {
		return // 读件败本拍让位
	}
	// 7d 落账=引证过的业主升件。
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e
		WHERE e.seat='业主' AND e.event='升件' AND e.ts>now()-interval '7 days'
		AND EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.event IN ('处置','恢复')
			AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)'))`).
		Scan(&closed7d); err != nil {
		return
	}
	// 点破率曲线——业主发令移机频（7d×前7d 对拍；下行=沉默面收敛）。
	if err := c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE ts>now()-interval '7 days'),
		count(*) FILTER (WHERE ts<=now()-interval '7 days')
		FROM ops_event WHERE seat='业主' AND event='升件'
		AND ts>now()-interval '14 days'`).Scan(&spot7d, &spotPrev7d); err != nil {
		return
	}
	fp := fmt.Sprintf("%d|%d", openN, spot7d)
	if !gaugeDue(ctx, c, "owner_ledger", fp, ownerBeatEvery) {
		return
	}
	old := "—"
	if oldest != nil {
		old = oldest.UTC().Format("01-02")
	}
	note := fmt.Sprintf("v=%s 业主令账: 开账%d件(最陈%s) 7d落%d件 | 点破率 7d=%d 前7d=%d——业主被迫发令移机频应单调下行(盲区收敛度;上行=沉默面新出须普查)",
		fp, openN, old, closed7d, spot7d, spotPrev7d)
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','owner_ledger',$1)`,
		note); err != nil {
		return
	}
}
