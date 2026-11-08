// econ.go —— T-0314 经济案 digest O 群推（日拍首版不 LISTEN——fin_econ_lesson
// pending 计数批组一条推【经济案】；发败=上抛不落锚不重推、关态=只 log；
// venue_req 同款 sender/chatID/on 装配）。
package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Econ 经济案 digest 件（SG o-core 读本地 fin_econ_lesson——跨写零径只读+推）。
type Econ struct {
	pool   *pgxpool.Pool
	cli    Sender
	chatID func() string
	on     func() bool
	now    func() time.Time
}

func NewEcon(pool *pgxpool.Pool, cli Sender, chatID func() string, on func() bool) *Econ {
	return &Econ{pool: pool, cli: cli, chatID: chatID, on: on, now: time.Now}
}

// Scan 一拍（日拍 ticker 装配——pending>0 推待裁统计行；零 pending 零推）。
func (e *Econ) Scan(ctx context.Context) error {
	rows, err := e.pool.Query(ctx, `SELECT kind,count(*) FROM fin_econ_lesson
		WHERE status='pending' GROUP BY kind`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var parts []string
	var total int
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return err
		}
		parts = append(parts, fmt.Sprintf("%s=%d", econKind(k), n))
		total += n
	}
	if total == 0 {
		return nil
	}
	msg := fmt.Sprintf("【经济案】%d 件待裁——%s（经济案台账——裁定走决策席）",
		total, strings.Join(parts, " "))
	if e.on == nil || !e.on() {
		return nil
	}
	_, err = e.cli.SendText(ctx, e.chatID(), msg)
	return err
}

// econKind 经济案类机读键人话（T-0441——未知类原样透出不吞）。
func econKind(k string) string {
	switch k {
	case "rootcause_diag":
		return "根因诊断"
	case "trade_postmortem":
		return "交易复盘"
	case "settled_loss":
		return "结算亏损"
	case "falsified_verdict":
		return "证伪判定"
	case "cost_diverge":
		return "成本偏离"
	case "semantic_caliber":
		return "口径定义"
	}
	return k
}
