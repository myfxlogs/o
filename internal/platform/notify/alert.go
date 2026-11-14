// Alert 临界事件即时推送（T-0030 立、T-0032 根修）——复用 FeishuClient token 缓存。
// 三律（业主推送纪律 2026-10-24）：
//  1. 纯事件驱动——无事件零推送；
//  2. typed-nil 双防线：接口装类型化 nil 指针（(*FeishuClient)(nil)）时 Go 语义
//     a.cli==nil 为 false，SendText 调 nil receiver 即 panic（SG 崩环实证）——装配点
//     传 nil 接口为主修，本件 reflect 判为兜底防线（注入不炸锚双落）；
//  3. 持续告警退避：同指纹间隔 5→10→20→40→60min 顶（计数递进），首报即时——
//     kill_switch/dd 触线等瞬时事件首报不受退避；不同指纹各自计 ladder。
//
// 双频（业主令 2026-11-15）：chanMoney 集内 kinds=钱径面→主 chat_id 文本；
// 集外=系统运行情况→feishu.sys_chat_id 富卡片（CardSender 可升卡则卡，
// 否则文本；sys_chat_id 空=回落主频零漏件）。
package notify

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Sender 发送口（FeishuClient 满足；测试注假）。
type Sender interface {
	SendText(ctx context.Context, chatID, text string) (string, error)
}

const (
	backoffBase = 5 * time.Minute
	backoffCap  = 60 * time.Minute
)

// chanMoney 钱径 kinds（仓/金/账/风险/策略生灭/venue 注册——业主令双频分册；
// 集外一切=系统运行情况进系统频，未定性新 kind 默认落系统频护钱径净面）。
var chanMoney = map[string]bool{
	"ledger_diff_alert": true, "rollback_failed": true, "close_gave_up": true,
	"guard_close_loop": true, "dd_hardline_tripped": true, "dd_close_exhausted": true,
	"dir_close_exhausted": true, "kill_switch": true, "transfer_ambiguous": true,
	"bare_bag": true, "close_residual_perp": true, "residual_exposure": true,
	"residual_clear_failed": true, "dcd_auto_retire": true, "volsell_auto_retire": true,
	"bal_missing": true, "band_audit_drift": true, "scout_recommend": true,
	"net_budget_warn": true, "net_budget_crit": true,
}

// Alerter 退避推送器（同指纹指数退避——持续态告警禁变相刷屏）。
type Alerter struct {
	cli     Sender
	sysCli  Sender // 系统频发送件（第二机器人——nil=回落主 cli）
	chatID  func() string
	sysChat func() string // 系统频 chat 解析（nil/空=回落主频——双频未绑零漏件）
	on      func() bool   // enabled 热启判（nil=off）
	mu      sync.Mutex
	last    map[uint64]time.Time
	sent    map[uint64]int // 同指纹已推次数（ladder 档位）
}

// NewAlerter enabled/chatID 缺席=fail-closed 降级（on=nil 或 false=只 log）。
func NewAlerter(cli Sender, chatID func() string, on func() bool) *Alerter {
	return &Alerter{cli: cli, chatID: chatID, on: on,
		last: map[uint64]time.Time{}, sent: map[uint64]int{}}
}

// WithSysChat 挂系统频解析（fluent——双频分流件；nil=全量走主频旧行为）。
func (a *Alerter) WithSysChat(fn func() string) *Alerter { a.sysChat = fn; return a }

// WithSysSender 挂系统频发送件（第二机器人——业主令 2026-11-15 meta 飞书挪作
// 系统通道；nil/typed-nil=回落主 cli 不炸）。
func (a *Alerter) WithSysSender(cli Sender) *Alerter { a.sysCli = cli; return a }

// cliNil typed-nil 判：接口零值或内裹 nil 指针皆判缺（反射兜底——主修在装配点）。
func cliNil(s Sender) bool {
	if s == nil {
		return true
	}
	return reflect.ValueOf(s).Kind() == reflect.Ptr && reflect.ValueOf(s).IsNil()
}

func (a *Alerter) sysChatID() string {
	if a.sysChat == nil {
		return ""
	}
	return a.sysChat()
}

// Alert 推送（kind+fingerprint 指纹退避；禁重试——单次失败计数返回）。
func (a *Alerter) Alert(ctx context.Context, kind, payload string) {
	if a == nil || cliNil(a.cli) || a.chatID == nil || a.chatID() == "" || (a.on != nil && !a.on()) {
		return
	}
	// T-0081 返修：指纹按 kind+title 定（payload 变额每抖重置阶梯=持续差
	// 退避失效病灶——title 稳定=同病同阶）。
	body0, _ := HumanText(kind, payload)
	title := strings.SplitN(body0, "\n", 2)[0]
	fp := fnv.New64a()
	_, _ = fp.Write([]byte(kind + "|" + title))
	k := fp.Sum64()
	a.mu.Lock()
	n := a.sent[k]
	if n > 0 { // 首报即时（n=0）不受退避
		wait := backoffBase << min(n-1, 4) // 5→10→20→40→60 顶（4 档封）
		if wait > backoffCap {
			wait = backoffCap
		}
		if t, ok := a.last[k]; ok && time.Since(t) < wait {
			a.mu.Unlock()
			return
		}
	}
	a.last[k] = time.Now()
	a.sent[k] = n + 1
	a.mu.Unlock()
	body, actable := HumanText(kind, payload)
	sys := !chanMoney[kind]
	chat := a.chatID()
	cli := a.cli
	if sys && !cliNil(a.sysCli) && a.sysChatID() != "" {
		// 系统频成对翻（cli+chat 必须同对——跨对=目标会话机器人不在席发送必败）
		cli, chat = a.sysCli, a.sysChatID()
		if cs, ok := cli.(CardSender); ok { // 系统频富卡（业主令——败则回落文本不丢件）
			card := AlertCard(title, fmt.Sprintf("%s\n—— %s", body, time.Now().Format("15:04:05")), actable)
			if _, err := cs.CreateInteractive(ctx, chat, card); err == nil {
				return
			} else {
				log.Printf("[alert] feishu card push err (%s): %v", kind, err)
			}
		}
	}
	prefix := "[经营事件] "
	if actable {
		prefix = "[O告警] "
	}
	text := fmt.Sprintf("%s%s | %s", prefix, body, time.Now().Format("15:04:05"))
	if _, err := cli.SendText(ctx, chat, text); err != nil {
		log.Printf("[alert] feishu push err (%s): %v", kind, err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
