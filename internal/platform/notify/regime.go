// Regime 判词推送件（T-0312 O 群推送——[Regime] 前缀与 [财经日历]/[经营事件]
// 分区；读 SG 镜像 md_regime_fact JOIN md_regime_memo(fact_id)，md_regime_push
// 幂等锚重启/重拍不重推；ev_regime 帧触发非轮询；败=计数+log 不炸）。
package notify

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Regime 判词推送件（sender 复用同装配面——不新造栈）。
type Regime struct {
	pool   *pgxpool.Pool
	cli    Sender
	chatID func() string
	on     func() bool
}

// NewRegime enabled/chatID 缺席=fail-closed 静默（on=nil/false=只 log 不发）。
func NewRegime(pool *pgxpool.Pool, cli Sender, chatID func() string, on func() bool) *Regime {
	return &Regime{pool: pool, cli: cli, chatID: chatID, on: on}
}

// Scan 一拍（ev_regime 帧调用点）：查未推判词批组一条推——零新行零推送。
func (g *Regime) Scan(ctx context.Context) error {
	rows, err := g.pool.Query(ctx, `SELECT f.id,f.dim,f.stance,f.confidence,f.created_ts,f.valid_until,
		(SELECT string_agg(memo,' / ') FROM md_regime_memo m WHERE m.fact_id=f.id)
		FROM md_regime_fact f
		WHERE NOT EXISTS (SELECT 1 FROM md_regime_push p WHERE p.fact_id=f.id)
		ORDER BY f.created_ts`)
	if err != nil {
		return err // 表缺/读败=计数上抛不炸
	}
	defer rows.Close()
	type row struct {
		id     int64
		dim    string
		stance string
		conf   *float64
		cts    *time.Time
		vu     *time.Time
		memo   *string
	}
	var out []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.dim, &r.stance, &r.conf, &r.cts, &r.vu, &r.memo); err != nil {
			return err
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("[市场判词] " + fmt.Sprint(len(out)) + " 条\n")
	for _, r := range out {
		conf := "—"
		if r.conf != nil {
			conf = fmt.Sprintf("%.2f", *r.conf)
		}
		b.WriteString(fmt.Sprintf("· %s：%s(%s)", r.dim, regimeStance(r.stance), conf))
		if r.memo != nil && *r.memo != "" {
			b.WriteString("——" + *r.memo)
		}
		b.WriteString("\n")
	}
	if g.on == nil || !g.on() {
		log.Printf("[regime] 推送关——%d 判词待推（不发只锚）", len(out))
	} else if _, err := g.cli.SendText(ctx, g.chatID(), b.String()); err != nil {
		return err // 推送败=上抛下拍重试（push 锚未落不重推不误）
	}
	// 幂等锚同拍落（发/不发同锚——关态下拍不重扫；发败=上抛已 return 不落锚）。
	for _, r := range out {
		if _, err := g.pool.Exec(ctx, `INSERT INTO md_regime_push(fact_id) VALUES($1)
			ON CONFLICT (fact_id) DO NOTHING`, r.id); err != nil {
			return err
		}
	}
	return nil
}

// regimeStance 判词立场机读枚举人话（risk_on/off/neutral/uncertain——
// T-0441 白话令；未知值原样透出不吞）。
func regimeStance(s string) string {
	switch s {
	case "risk_on":
		return "风险偏好上行"
	case "risk_off":
		return "风险偏好下行"
	case "neutral":
		return "中性"
	case "uncertain":
		return "不确定"
	}
	return s
}
