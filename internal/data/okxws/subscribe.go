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

// subscribeArgs 宇宙低频订阅面（T-0132 分级；tickers 退出宇宙订活跃集
// 动态补：funding-rate 仅 SWAP instId 合法）。
// T-0171 快照价格全面退役：T-0134/T-0135 的 REST 轮询替（spotpoll）已撤——
// SG 瘦身面 funding-rate 之外全赖活跃集 tickers 动态订（含 dcd 现货腿）。
func subscribeArgs(syms []string) []map[string]string {
	args := make([]map[string]string, 0, len(syms)*3)
	for _, s := range syms {
		switch {
		case strings.HasSuffix(s, "-SWAP"):
			args = append(args, map[string]string{"channel": "funding-rate", "instId": s})
		case datedFutRe.MatchString(s): // 季度期货=SG 瘦身面零订（桥面 mark-price 供）
			continue
		default: // 现货形=宇宙内零订（活跃集 tickers 动态补——T-0171 唯一现货价源）
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

// activeSet 活跃集标的（T-0132 分级——在仓+在途提案+live/degraded 策略 inst_id
// 才配 tickers 级实时性；其余宇宙标的三低频面即足）。
// T-0171 价格源律：dcd 策略现货腿 uly 一并入集（spotpoll REST 退役后现货腿
// 真价唯一来源=本地 tickers WS 行——md_ticks 供 dcdPaperAccounting 记账价）；
// uly 方言归一（OKX DCD uly=标的指数名 BTC-USD/BTC-USDG→现货对 BTC-USDT，
// 与 research/dcd spot() 候选链同语义）。
// T-0216 返修：basis 两族 paper/live 策略现货腿并入集（基差期货键剥 -YYMMDD
// 尾→剥 -USD 尾→-USDT 现货对，同 dcd 归一链）——basisPaperAccounting 记账
// 现货腿价源（期货腿价=md_mark_index mark-price 桥面行，不占本地订）。
func (c *Collector) activeSet(ctx context.Context) (map[string]bool, error) {
	// T-0187：fin_position 项删——PG-md 无此表（fin 域禁引；持仓标的
	// 由 live/degraded 策略项覆盖=冗余项）。md 侧查询只许引 domain.tsv
	// 标 md|bridge 表（fin 域表禁引——relation does not exist 病灶根修）。
	rows, err := c.pool.Query(ctx, `SELECT DISTINCT inst_id FROM (
		SELECT inst_id FROM fin_proposal WHERE state IN ('fin_proposed','fin_armed')
		UNION SELECT inst_id FROM fin_strategy WHERE state IN ('live','degraded')
		UNION SELECT regexp_replace(d.uly, '-USD(G)?$', '-USDT') FROM fin_strategy s
			JOIN md_dcd_product d ON d.product_id=s.inst_id
			WHERE s.state<>'retired' AND s.template_ref LIKE 'dcd%'
		UNION SELECT regexp_replace(regexp_replace(s.inst_id,'-[0-9]{6}$',''),'-USD$','-USDT')
			FROM fin_strategy s WHERE s.state IN ('paper','live','degraded')
			AND s.template_ref IN ('quarter_basis_long','perp_quarter_spread')) t`)
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

// syncActiveSet 活跃集动态重订（T-0132 面；T-0133 三修——①幽灵标的滤除：
// activeSet 提案/策略/在仓 inst_id 含 DCD 产品名等假想链产物非真实行情
// 标的，∩订阅宇宙 syms 才发 tickers 帧；②加订分帧节流≤100 args/帧
// ——60014 限频实证；③60018 拒收集 refused 本连接内不再重发）。
func (c *Collector) syncActiveSet(ctx context.Context, conn *websocket.Conn,
	syms []string, subTick, refused map[string]bool) {
	valid := map[string]bool{}
	for _, s := range syms {
		valid[s] = true
	}
	set, err := c.activeSet(ctx)
	if err != nil {
		c.rej.Inc("activeset_read_failed")
		return
	}
	var adds []map[string]string
	for inst := range set {
		if subTick[inst] || !isUSDTInst(inst) || !valid[inst] || refused[inst] {
			continue
		}
		adds = append(adds, map[string]string{"channel": "tickers", "instId": inst})
	}
	if err := subscribeAll(conn, adds); err != nil {
		return
	}
	for _, a := range adds {
		subTick[a["instId"]] = true
	}
	for inst := range subTick {
		if !set[inst] {
			if err := conn.WriteJSON(subArgs("unsubscribe", inst)); err != nil {
				return
			}
			delete(subTick, inst)
		}
	}
}

// subscribeArgsFull 桥面全量订阅面（T-0137：dev 蒸馏=index-tickers 现货全量+
// mark-price 全量+funding-rate——不受 SG 瘦身口径；tickers 仍活跃集动态补）。
func subscribeArgsFull(syms []string) []map[string]string {
	args := make([]map[string]string, 0, len(syms)*4)
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

// subscribeSet 订阅面分派（桥=全量含 index-tickers/mark-price；SG=瘦身面）。
func (c *Collector) subscribeSet(syms []string) []map[string]string {
	if c.FullSub {
		return subscribeArgsFull(syms)
	}
	return subscribeArgs(syms)
}
