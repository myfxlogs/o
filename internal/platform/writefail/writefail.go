// Package writefail —— 组件拍败/写败自报升件公共件（工单#22873）。
// 「事件拍 fn 败」「[arm] 拍败」日志行=静默面——o-research 42P10 类
// 写拒连刷 18h 升件零发实证（hunger probe 阈 48h/3d 够不着）。本件
// 同 hungerEscalate 写败类同构：ops_event 升件 esc_class='写败'，
// esc_scope=组件判例域 24h 同窗去重（单语句 INSERT...WHERE NOT
// EXISTS——族 norm 同根因不暴）；自报行自身写败=静默退（结构性限度
// 在案——升件不能再升）。写池由宿主进程 boot 装配（SetPool——研究
// 节点 ops_event 驻 fin 域=finPool；未装配=nil 守位不报，core 等
// 他进程天然免疫——池只此一份进程级单写者）。
package writefail

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Pool 升件写面最小缝（*pgxpool.Pool 同形——桩测可覆写）。
type Pool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

var pool Pool // 宿主装配池（SetPool——boot 置位后只读）

// SetPool 装配升件写池（cmd/research boot——finPool；nil=守位不报）。
func SetPool(p Pool) { pool = p }

// Report 写败自报一行：seat=上报组件身份，ticket=归属票，scope=判例
// 域单键（同窗去重锚——同 scope 24h 至多一报），detail=败因原文
// （note 载全——窗内首报因在案，后续因被去重不落，族 norm 同根因
// 不暴）。自报行自身写败=静默退（log 级留痕——结构性限度在案）。
func Report(ctx context.Context, seat, ticket, scope, detail string) {
	p := pool
	if p == nil {
		return // 守位——池未装配（非研究节点进程/测试件）
	}
	// 自报写自带 10s 期界——fin 库挂时 Exec 不得冻结宿主拍环
	// （事件脊 ctx=进程级无 deadline；hungerEscalate 同款裸 Exec
	// 怕点在 tick ctx 自带期界，本件宿主多为无界 ctx 故自设）。
	rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := p.Exec(rctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件',$2,$3,'写败',$4
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='升件' AND esc_class='写败' AND esc_scope=$4
			AND ts>now()-interval '24 hours')`,
		seat, ticket, fmt.Sprintf("%s 写败: %s", scope, detail), scope); err != nil {
		log.Printf("[writefail] 自报行落败（结构性限度静默退）: %v", err)
	}
}
