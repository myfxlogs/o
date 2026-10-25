// subscribe.go —— 订阅面装配件（T-0122 返修② 拆出——棘轮行数控制件；
// T-0132 分级：宇宙=低频三面 args，tickers 退出全宇宙改活跃集动态增删）：
// 宇宙 args 构建+≤100 args/帧分帧发送（OKX WS 单帧硬限——全宇宙 ~500
// arg 单帧被拍死；帧间 50ms 节流防限流）+活跃集增删订阅帧。
package okxws

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// datedFutRe 季度期货形（'-' 尾 6 位到期日——mark-price 适用无 funding）。
var datedFutRe = regexp.MustCompile(`-[0-9]{6}$`)

// subscribeArgs 宇宙低频订阅面（T-0132 分级——全宇宙=费率/标记价/指数价三
// 事实层零损发现面；tickers 退出宇宙订活跃集动态补：funding-rate/mark-price
// 仅 SWAP instId 合法；index-tickers 挂 spot 形 instId；季度期货订 mark-price）。
func subscribeArgs(syms []string) []map[string]string {
	args := make([]map[string]string, 0, len(syms)*3)
	for _, s := range syms {
		switch {
		case strings.HasSuffix(s, "-SWAP"):
			args = append(args,
				map[string]string{"channel": "funding-rate", "instId": s},
				map[string]string{"channel": "mark-price", "instId": s})
		case datedFutRe.MatchString(s):
			args = append(args, map[string]string{"channel": "mark-price", "instId": s})
		default:
			args = append(args, map[string]string{"channel": "index-tickers", "instId": s})
		}
	}
	return args
}

// subArgs tickers 增删订阅帧（op=subscribe/unsubscribe——活跃集动态重订不重建连接）。
func subArgs(op, inst string) map[string]any {
	return map[string]any{"op": op,
		"args": []map[string]string{{"channel": "tickers", "instId": inst}}}
}

// subscribeAll ≤100 args/帧分帧订阅（帧间节流 50ms）。
func subscribeAll(conn *websocket.Conn, args []map[string]string) error {
	for i := 0; i < len(args); i += 100 {
		end := i + 100
		if end > len(args) {
			end = len(args)
		}
		if err := conn.WriteJSON(map[string]any{
			"op": "subscribe", "args": args[i:end]}); err != nil {
			return fmt.Errorf("subscribe 帧 %d-%d: %w", i, end, err)
		}
		if end < len(args) {
			time.Sleep(50 * time.Millisecond) // sched-exempt-ok：订阅节流
		}
	}
	return nil
}

// activeSet 活跃集标的（T-0132 分级——在仓+在途提案+非退役策略 inst_id
// 才配 tickers 级实时性；其余宇宙标的三低频面即足）。
func (c *Collector) activeSet(ctx context.Context) (map[string]bool, error) {
	rows, err := c.pool.Query(ctx, `SELECT DISTINCT inst_id FROM (
		SELECT inst_id FROM fin_position WHERE closed_at IS NULL
		UNION SELECT inst_id FROM fin_proposal WHERE state IN ('fin_proposed','fin_armed')
		UNION SELECT inst_id FROM fin_strategy WHERE state<>'retired') t`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	set := map[string]bool{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		set[s] = true
	}
	return set, rows.Err()
}
