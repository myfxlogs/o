// subscribe.go —— 订阅面装配件（T-0122 返修② 拆出——棘轮行数控制件）：
// 四频道 args 构建+≤100 args/帧分帧发送（OKX WS 单帧硬限——全宇宙 ~500
// arg 单帧被拍死；帧间 50ms 节流防限流）。
package okxws

import (
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// subscribeArgs 四频道订阅面（funding-rate/mark-price 仅 SWAP instId 合法；
// index-tickers 挂 spot 形 instId；tickers 两形通用——books5 已切除
// T-0117 返修③：D-0018 停采零读者面订阅负荷主动降）。
func subscribeArgs(syms []string) []map[string]string {
	args := make([]map[string]string, 0, len(syms)*4)
	for _, s := range syms {
		args = append(args, map[string]string{"channel": "tickers", "instId": s})
		if strings.HasSuffix(s, "-SWAP") {
			args = append(args,
				map[string]string{"channel": "funding-rate", "instId": s},
				map[string]string{"channel": "mark-price", "instId": s})
		} else if !strings.Contains(s, "-2") { // 现货形=index-tickers；
			// 季度期货（-USDT-YYMMDD 尾）指数面不适用——tickers 唯订
			args = append(args, map[string]string{"channel": "index-tickers", "instId": s})
		}
	}
	return args
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
