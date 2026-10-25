// 推径：报表行落库后→飞书摘要 SendText。fail-closed 降级——凭据/chat_id 缺=不推
// +降级计数（投递失败不抹报表行不回滚）。token 缓存在 FeishuClient 内。
package report

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
)

// Pusher 推送岗（cli=Sender 接口注入——凭据缺席=nil 降级不推不炸；接口面防
// typed-nil 同病灶：装配只装非 nil 具体件，测试注假件）。
type Pusher struct {
	cfg  *config.Client
	pool *pgxpool.Pool
	cli  notify.Sender
	rej  *rejects.Door
}

// NewPusher 装配：凭据自 app_secrets 密文解出（mk=master key）；任何一环缺=cli=nil。
func NewPusher(pool *pgxpool.Pool, cfg *config.Client, mk []byte) *Pusher {
	p := &Pusher{cfg: cfg, pool: pool,
		rej: rejects.New("report", "push_degraded", "push_err")}
	var idCt, secCt []byte
	if err := pool.QueryRow(context.Background(),
		"SELECT ciphertext FROM app_secrets WHERE key='feishu.app_id'").Scan(&idCt); err != nil {
		p.rej.Inc("push_degraded")
		return p
	}
	if err := pool.QueryRow(context.Background(),
		"SELECT ciphertext FROM app_secrets WHERE key='feishu.app_secret'").Scan(&secCt); err != nil {
		p.rej.Inc("push_degraded")
		return p
	}
	id, err1 := cryptoutil.Open(mk, idCt)
	sec, err2 := cryptoutil.Open(mk, secCt)
	if err1 != nil || err2 != nil {
		p.rej.Inc("push_degraded")
		return p
	}
	p.cli = notify.NewFeishu(id, sec)
	return p
}

// MaybePush Gen 落行后即时推（notify.feishu.enabled 热启；chat_id 缺=fail-closed
// 降级计数非炸——票据语义：无 chat_id=不推不炸）。
func (p *Pusher) MaybePush(ctx context.Context, period string, m map[string]any, note string) {
	if p.cli == nil {
		p.rej.Inc("push_degraded")
		return
	}
	en, err := p.cfg.GetBool("notify.feishu.enabled")
	if err != nil || !en {
		p.rej.Inc("push_degraded")
		return
	}
	chatID, err := p.cfg.GetString("feishu.chat_id")
	if err != nil || chatID == "" {
		p.rej.Inc("push_degraded")
		return
	}
	text := fmt.Sprintf("【O %s 报】%s\n净额 %s / 累计净值 %s\n开 %s 平 %s / 评 %s 过 %s 拒\nEarn 停泊 %s\n%s",
		map[string]string{"daily": "日", "weekly": "周", "monthly": "月"}[period],
		boundsLabel(period),
		fmt.Sprint(m["net"]), fmt.Sprint(m["cum_nav"]),
		fmt.Sprint(m["opened"]), fmt.Sprint(m["closed"]),
		fmt.Sprint(m["proposed"]), fmt.Sprint(m["rejected"]),
		fmt.Sprint(m["earn_balance"]), note)
	mid, err := p.cli.SendText(ctx, chatID, text)
	if err != nil {
		p.rej.Inc("push_err")
		log.Printf("[report] feishu push err: %v", err)
		return
	}
	log.Printf("[report] feishu pushed %s msg=%s", period, mid)
}

func boundsLabel(period string) string {
	return BoundsOf(period, time.Now()).Start.Format("2006-01-02")
}

// PushText 通用文本推送（日历日推径——enabled/chat_id 缺=降级计数同 MaybePush）。
func (p *Pusher) PushText(ctx context.Context, text string) {
	if p.cli == nil {
		p.rej.Inc("push_degraded")
		return
	}
	en, err := p.cfg.GetBool("notify.feishu.enabled")
	if err != nil || !en {
		p.rej.Inc("push_degraded")
		return
	}
	chatID, err := p.cfg.GetString("feishu.chat_id")
	if err != nil || chatID == "" {
		p.rej.Inc("push_degraded")
		return
	}
	if _, err := p.cli.SendText(ctx, chatID, text); err != nil {
		p.rej.Inc("push_err")
		log.Printf("[report] feishu push err: %v", err)
	}
}
