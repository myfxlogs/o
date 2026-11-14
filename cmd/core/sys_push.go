// sys_push.go —— 系统频富卡推面（业主令 2026-11-15：飞书双频——钱径文本报
// 走 feishu.chat_id 零改；系统运行情况富卡片走 feishu.sys_chat_id）。
// 件①在办活卡：ev_headless 敲门（fin_0183 在办/唤起/收口/果败/叙事族行落
// 即醒）→重算在办面→PATCH 同一张卡——「我现在正在做什么」=常驻活卡
// 原地刷新非消息流；msg_id 存 fin_push_ledger(sys_status/doing_card)——
// 重启续 PATCH 不重发新卡；PATCH 败（卡被删）=新发换 mid 自愈。
package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/notify"
)

type sysPush struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	cli  *notify.FeishuClient
}

// startSysPush 装配（cli nil=凭据缺席整件降级不炸——同 alerter fail-closed 律；
// ev_headless 事件脊钦定径非轮询）。
func startSysPush(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client, cli *notify.FeishuClient) {
	if cli == nil {
		return
	}
	sp := &sysPush{pool: pool, cfg: cfg, cli: cli}
	go events.ListenLoop(ctx, pool, "sys_push", []string{"ev_headless"}, sp.push)
}

// push 一敲一刷。fail-closed 两闸：enabled 关 / sys_chat_id 空=不推不炸。
func (s *sysPush) push(ctx context.Context) error {
	if en, _ := s.cfg.GetBool("notify.feishu.sys.enabled"); !en {
		return nil
	}
	chat, _ := s.cfg.GetString("feishu.sys_chat_id")
	if chat == "" {
		return nil
	}
	rows, err := queryDoingRows(ctx, s.pool)
	if err != nil {
		return err
	}
	card := notify.StatusCard(s.cardLines(ctx, rows), anyFresh(rows))
	if mid := s.cardMID(ctx); mid != "" {
		if err := s.cli.PatchCard(ctx, mid, card); err == nil {
			return nil
		} // PATCH 败=卡已删/旧mid失效→下面新发换锚
	}
	mid, err := s.cli.CreateInteractive(ctx, chat, card)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO fin_push_ledger(scope,key,msg_id)
		VALUES('sys_status','doing_card',$1)
		ON CONFLICT (scope,key) DO UPDATE SET msg_id=EXCLUDED.msg_id,pushed_at=now()`, mid)
	return err
}

// cardMID 活卡锚（fin_push_ledger 幂等锚面复用——重启首拍续 PATCH 不重发）。
func (s *sysPush) cardMID(ctx context.Context) string {
	var mid string
	_ = s.pool.QueryRow(ctx,
		`SELECT msg_id FROM fin_push_ledger WHERE scope='sys_status' AND key='doing_card'`).Scan(&mid)
	return mid
}

func anyFresh(rs []doingRowT) bool {
	for _, r := range rs {
		if r.fresh {
			return true
		}
	}
	return false
}

// ———— 去黑话渲染层（业主令 2026-11-15③：卡体去黑话——k=v 机产字段中化、
// unit 机产ID 短名、沪时更戳；会话自撰正文留原声仅做标点/常见词清理）————

// cardLines 卡体行：在办明细（态/短名/声明/龄）+分隔+最近一场+自述+更时。
func (s *sysPush) cardLines(ctx context.Context, rs []doingRowT) []string {
	var ls []string
	if len(rs) == 0 {
		ls = append(ls, "**歇着——没有在办的活**")
	}
	for _, r := range rs {
		tag := "在干"
		if !r.fresh {
			tag = "挂起没动"
		}
		ls = append(ls, fmt.Sprintf("**%s** %s\n%s（%s）",
			tag, shortUnit(r.unit), r.note, agoS(time.Since(r.ts))))
	}
	ls = append(ls, "────────")
	if t := s.lastNote(ctx, `('会话收口','唤起果','唤起败')`); t != "" {
		ls = append(ls, "**最近一场** "+humanizeKV(t))
	}
	if t := s.lastNote(ctx, `('叙事')`); t != "" {
		ls = append(ls, "**自述** "+humanizeNarr(t))
	}
	ls = append(ls, fmt.Sprintf("_更新 %s（沪时）_", time.Now().In(shTZ()).Format("15:04")))
	return ls
}

// lastNote 事件族末行 note（截 120 字防卡体臃肿）。败=空串静默不炸卡。
func (s *sysPush) lastNote(ctx context.Context, events string) string {
	var n string
	_ = s.pool.QueryRow(ctx, fmt.Sprintf(
		`SELECT left(note,120) FROM ops_event WHERE event IN %s
		 ORDER BY id DESC LIMIT 1`, events)).Scan(&n)
	return strings.TrimSpace(n)
}

// agoS 龄读法（分内=刚刚）。
func agoS(d time.Duration) string {
	m := int(d.Minutes())
	if m < 1 {
		return "刚刚"
	}
	if m < 60 {
		return fmt.Sprintf("%d 分钟前", m)
	}
	return fmt.Sprintf("%d 小时 %d 分前", m/60, m%60)
}

// shTZ 业主时区（Asia/Shanghai——同 trx_vote_rebalance 惯例；败兜 CST+8）。
func shTZ() *time.Location {
	if l, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return l
	}
	return time.FixedZone("CST", 8*3600)
}

// shortUnit 机产 unit ID→短名（o-cognition-NNNN→唤知·尾NNNN；
// o-cognition-monologue→唤知·独白；o-cog 截尾→唤知；其他=o-前缀剥）。
func shortUnit(u string) string {
	if u == "o-cog" {
		return "唤知"
	}
	s := strings.TrimPrefix(u, "o-cognition-")
	if s == u {
		return strings.TrimPrefix(u, "o-")
	}
	if s == "monologue" {
		return "唤知·独白"
	}
	if len(s) > 4 && isDigits(s) {
		return "唤知·尾" + s[len(s)-4:]
	}
	return "唤知·" + s
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

var leadKV = regexp.MustCompile(`^(\w+)=(\S+)`)

// humanizeKV 末场行头段 k=v 机产字段中化（unit→短名/host→@机/rc=0 省非零示/
// reason→中义/dur→读法）；头段后正文留原声走 narrTidy。无 k=v=原形留。
func humanizeKV(note string) string {
	rest := note
	var tags []string
	for {
		m := leadKV.FindStringSubmatch(rest)
		if m == nil {
			break
		}
		switch m[1] {
		case "unit":
			tags = append(tags, shortUnit(m[2]))
		case "host":
			tags = append(tags, "@"+m[2])
		case "rc":
			if m[2] != "0" {
				tags = append(tags, "退出码"+m[2])
			}
		case "reason":
			tags = append(tags, reasonCN(m[2]))
		case "dur":
			tags = append(tags, durCN(m[2]))
		default:
			tags = append(tags, m[1]+"="+m[2])
		}
		rest = strings.TrimLeft(rest[len(m[0]):], " ")
	}
	if len(tags) == 0 {
		return narrTidy(note)
	}
	out := strings.Join(tags, " · ")
	if rest != "" {
		out += "｜" + narrTidy(rest)
	}
	return out
}

// reasonCN 收口因码中义（生码留原）。
func reasonCN(r string) string {
	switch r {
	case "session_incomplete":
		return "会话未完整"
	case "ok", "done":
		return "正常收口"
	case "timeout":
		return "超时"
	case "error":
		return "出错"
	case "killed":
		return "被杀"
	}
	return r
}

// durCN 时长读法（ParseDuration 生码留原）。
func durCN(d string) string {
	t, err := time.ParseDuration(d)
	if err != nil {
		return d
	}
	switch {
	case t < time.Minute:
		return fmt.Sprintf("%d 秒", int(t.Seconds()))
	case t < time.Hour:
		return fmt.Sprintf("%d 分 %d 秒", int(t.Minutes()), int(t.Seconds())%60)
	default:
		return fmt.Sprintf("%d 小时 %d 分", int(t.Hours()), int(t.Minutes())%60)
	}
}

var unitTok = regexp.MustCompile(`unit=o-cognition-[0-9A-Za-z_-]+|o-cognition-[0-9A-Za-z_-]+|o-cog\b`)

// narrTidy 正文常见机产词清理（unit 令牌短名化+常见 k=v/缩词替换）。
func narrTidy(s string) string {
	s = unitTok.ReplaceAllStringFunc(s, func(m string) string {
		return shortUnit(strings.TrimPrefix(m, "unit="))
	})
	return strings.NewReplacer(
		"用帧=无", "未用帧",
		"自问=无", "无自问",
		"deploy=master ", "部署 ",
		"NAV", "净值",
	).Replace(s)
}

var (
	narrDay  = regexp.MustCompile(`第(\d+)日`)
	narrCnt  = regexp.MustCompile(`唤(\d+)场/收(\d+)/败(\d+)`)
	narrDeal = regexp.MustCompile(`处置(\d+)件`)
	narrTk   = regexp.MustCompile(`工单(\d+)项`)
	narrKV   = regexp.MustCompile(`\|([^|=\s]+)=`)
)

// humanizeNarr 自述行整理：日号/场计三连/处置工单计数展开+k=v 转「名：值」。
func humanizeNarr(s string) string {
	s = narrTidy(s)
	s = narrDay.ReplaceAllString(s, "第 $1 日")
	s = narrCnt.ReplaceAllString(s, "唤起 $1·收口 $2·败 $3")
	s = narrDeal.ReplaceAllString(s, "处置 $1")
	s = narrTk.ReplaceAllString(s, "工单 $1")
	s = narrKV.ReplaceAllString(s, "｜$1：")
	return strings.ReplaceAll(s, "; ", "｜")
}
