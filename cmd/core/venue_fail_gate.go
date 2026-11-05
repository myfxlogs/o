package main

// venueFailGate venue 读败连败升件闸（T-0352——gauges 覆盖补洞：venue 读面
// 连败只进 docker 日志=无人值守面事实失明，COS/业主零可见性）。
// 族通用件：venue 键参数化（okx/mt5/…——mt5=当前病灶非硬编码特例）；
// 只观测不改读径（fail-closed 原样上抛——读败根因修另案，红线）。
//
// 升件三件套对形（scripts/meta/gauges_tick.sh 同构）：
//   - 滞回：连败计数跨阈才升，读常一拍即复零（复常清零）；
//   - 日限：1/venue/日——DB 侧 NOT EXISTS 原子去重（进程重启不破）；
//   - 事件行：ops_event event='升件'（seat=o-core 既有径——trx.go 同型；
//     ticket='' 空挂不污染工单板；esc_class='读败'+esc_scope=<venue> 兼判例键）。
//
// 判据：连败 ≥venueFailThrBeats 拍（事件拍快面）或连败持续 ≥venueFailThrDur
// （慢拍面等价锚）——先到先升；升件行带 venue/败因摘要/连败时长。
// 计数粒度=venue 级跨面共享（对账/风控/守卫同 venue 同源可读性——任一面
// 读常即复零，任一面读败即累拍）。

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// 判据参量（var=测试面覆写锚——生产值钦定不动）。
var (
	venueFailThrBeats = 10              // 连败升件拍阈（事件拍≈5s 面≈50s 持续）
	venueFailThrDur   = 5 * time.Minute // 慢拍面持续阈（≥5min 连败先到先升）
	venueFailNoteMax  = 200             // 败因摘要截断（防长错文刷账）
)

type venueFailGate struct {
	pool *pgxpool.Pool
	mu   sync.Mutex
	st   map[string]*venueFailSt
}

type venueFailSt struct {
	n       int       // 本轮连败拍数
	since   time.Time // 本轮首败 ts（连败时长锚）
	lastErr string    // 最近败因摘要
}

func newVenueFailGate(pool *pgxpool.Pool) *venueFailGate {
	return &venueFailGate{pool: pool, st: map[string]*venueFailSt{}}
}

// fail 记一拍读败；跨阈且当日该 venue 未升→升件行落 ops_event。
// 观测面零侵入：本函数任何败（含写行败）都不上抛不拖读径。
func (g *venueFailGate) fail(ctx context.Context, venue string, err error) {
	if g == nil || err == nil || g.pool == nil {
		return
	}
	now := time.Now()
	g.mu.Lock()
	s := g.st[venue]
	if s == nil || s.n == 0 { // 新连败轮——since 锚本轮首败
		s = &venueFailSt{since: now}
		g.st[venue] = s
	}
	s.n++
	s.lastErr = venueFailOneLine(err.Error())
	n, since, why := s.n, s.since, s.lastErr
	g.mu.Unlock()
	if n < venueFailThrBeats && (n < 2 || now.Sub(since) < venueFailThrDur) {
		return // 未跨阈（拍数/持续双径，先到先升）
	}
	dur := now.Sub(since).Round(time.Second)
	note := fmt.Sprintf("venue 读败连败升件：%s 连败 %d 拍/%s——败因 %s（阈=%d拍或%s；读常清零；日限1）",
		venue, n, dur, why, venueFailThrBeats, venueFailThrDur)
	// 日限 1/venue/日：NOT EXISTS 原子去重——跨阈拍多写同幂等，重启复拍不重升。
	_, _ = g.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT 'o-core','升件','',$1,'读败',$2
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE seat='o-core' AND event='升件' AND esc_class='读败' AND esc_scope=$2
			  AND ts >= date_trunc('day', now()))`, note, venue)
}

// ok 读常一拍——滞回复常清零（票面钦定：复常清零）。
func (g *venueFailGate) ok(venue string) {
	if g == nil {
		return
	}
	g.mu.Lock()
	delete(g.st, venue)
	g.mu.Unlock()
}

// venueFailOneLine 败因摘要一行化+截断（升件行 note 卫生）。
func venueFailOneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > venueFailNoteMax {
		s = s[:venueFailNoteMax] + "…"
	}
	return s
}
