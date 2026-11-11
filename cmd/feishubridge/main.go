// feishubridge —— 飞书交互端常驻件（钦定分工：飞书=交互端——条件请求
// 交互卡推送+钮回调+文本口令三径入，业主域窄门；web 面板只读不写，
// 飞书永不长报表）。
//
// 出站：轮询 fin_condition_gap pending（30s）→interactive 卡
// [已办/驳回/销] 三钮对 state 机三终态。
// 入站：lark ws 长连接（零公网口）收 card.action.trigger+
// im.message.receive_v1。窄门=绑定 chat_id 吻合（卡件 Context.
// OpenChatID/消息 ChatId）+ 动作/state 闭集 + 发送者 user——
// 命中→fin_condition_gap 结案（pending 才动）+ops_event 业主令意图行
// append-only。凭据同 feishu-notify 栈（app_secrets 密文+主权钥+
// feishu.chat_id），零新凭据面。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/notify"

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "feishubridge:", err)
		os.Exit(1)
	}
}

// bridge 件持有面。
type bridge struct {
	pool   *pgxpool.Pool // roster_rw 写/读户（请求面+ops_event+secrets 读）
	rpool  *pgxpool.Pool // condreq_resolver 结案户（fin_0165——nil=结案径未配 fail-closed）
	fs     *notify.FeishuClient
	chatID string
	sent   map[int64]string // req_id→message_id（本程内推送记录）
}

// intent 业主动作意图行（ops_event append-only——业主令非相位闸词直通）。
func (b *bridge) intent(ctx context.Context, note string) {
	if _, err := b.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,note) VALUES('业主','业主令',$1)`, note); err != nil {
		log.Printf("feishubridge: intent 落账败: %v", err)
	}
}

// resolve 结案（pending 才动，condreq_resolver 户径——「写者≠结案者」
// 边界：本件池只结不改请求体）。返回是否真实翻转。
func (b *bridge) resolve(ctx context.Context, id int64, state, note string) (bool, error) {
	if b.rpool == nil {
		return false, fmt.Errorf("结案径未配（feishubridge.resolver_dsn 缺）")
	}
	r, err := b.rpool.Exec(ctx, `UPDATE fin_condition_gap SET state=$2,resolved_at=now(),resolve_note=$3
		WHERE id=$1 AND state='pending'`, id, state, note)
	if err != nil {
		return false, err
	}
	return r.RowsAffected() > 0, nil
}

// pushPending 未决请求→交互卡（水位外新发；发败水位不进——
// 下次敲门/开机扫底追偿）。
func (b *bridge) pushPending(ctx context.Context) {
	wm := readWM()
	rows, err := b.pool.Query(ctx, `SELECT id,kind,coalesce(venue,''),coalesce(surface,''),payload::text
		FROM fin_condition_gap WHERE state='pending' AND id>$1 ORDER BY id`, wm)
	if err != nil {
		log.Printf("feishubridge: pending 查败: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var kind, venue, surface, payload string
		if rows.Scan(&id, &kind, &venue, &surface, &payload) != nil {
			continue
		}
		if !ownerFacing(kind, payload) {
			continue // 会话域件不推业主——水位不进,flag 后翻仍可被推
		}
		mid, err := b.fs.CreateInteractive(ctx, b.chatID, reqCard(id, kind, venue, surface, payload))
		if err != nil {
			log.Printf("feishubridge: 卡发败 req=%d: %v", id, err)
			continue // 水位不进——下响追偿
		}
		b.sent[id] = mid
		writeWM(id)
	}
}

// listenPending condreq_pending 敲门环（fin_0164 触发器——INSERT/UPDATE
// 至 pending 即醒；开机先扫底覆盖停机期漏件；断流退避重连）。
func (b *bridge) listenPending(ctx context.Context) {
	b.pushPending(ctx) // 开机扫底——停机期漏件追偿
	for ctx.Err() == nil {
		c, err := b.pool.Acquire(ctx)
		if err != nil {
			time.Sleep(3 * time.Second) // sched-exempt-ok 断流重连退避（mechwake 同约——非轮询数据）
			continue
		}
		if _, err := c.Exec(ctx, "LISTEN condreq_pending"); err != nil {
			c.Release()
			time.Sleep(3 * time.Second) // sched-exempt-ok 同上
			continue
		}
		for ctx.Err() == nil {
			if _, err := c.Conn().WaitForNotification(ctx); err != nil {
				break // 断流→出内环重连
			}
			b.pushPending(ctx) // notify 只敲门——读仍走表正源
		}
		c.Release()
		time.Sleep(3 * time.Second) // sched-exempt-ok 断流重连退避
	}
}

// onCardAction card.action.trigger 回调——窄门(chat 吻合+动作闭集)→结案。
func (b *bridge) onCardAction(ctx context.Context, ev *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	deny := &callback.CardActionTriggerResponse{Toast: &callback.Toast{Type: "warning", Content: "非业主域"}}
	var ocid, aval string
	if e := ev.Event; e != nil && e.Context != nil {
		ocid = e.Context.OpenChatID
	}
	if e := ev.Event; e != nil && e.Action != nil {
		aval = fmt.Sprintf("%v", e.Action.Value)
	}
	log.Printf("feishubridge: 收卡钮 chat=%s val=%s", tail4(ocid), aval)
	if ev.Event == nil || ev.Event.Context == nil || ev.Event.Context.OpenChatID != b.chatID {
		return deny, nil
	}
	state, req, ok := actFromValue(ev.Event.Action.Value)
	if !ok {
		return deny, nil
	}
	who := "unknown"
	if ev.Event.Operator != nil && ev.Event.Operator.OpenID != "" {
		o := ev.Event.Operator.OpenID
		if len(o) > 8 {
			o = o[len(o)-8:]
		}
		who = "…" + o
	}
	changed, err := b.resolve(ctx, req, state, "飞书钮:"+who)
	if err != nil {
		return nil, err
	}
	if !changed { // 已结残卡回点——查现态回终态卡自愈（点下即灰）
		var cur string
		if err := b.pool.QueryRow(ctx,
			`SELECT state FROM fin_condition_gap WHERE id=$1`, req).Scan(&cur); err == nil && cur != "pending" {
			return &callback.CardActionTriggerResponse{
				Toast: &callback.Toast{Type: "info", Content: fmt.Sprintf("#%d 已结/无此件", req)},
				Card:  &callback.Card{Type: "raw", Data: doneCard(req, cur)}}, nil
		}
		return &callback.CardActionTriggerResponse{Toast: &callback.Toast{
			Type: "info", Content: fmt.Sprintf("#%d 已结/无此件", req)}}, nil
	}
	b.intent(ctx, fmt.Sprintf("条件请求#%d→%s[飞书钮 %s]", req, state, who))
	delete(b.sent, req)
	if mid := ev.Event.Context.OpenMessageID; mid != "" { // 兜底刷（文本令径/异宿无主响应卡通道）
		if err := b.fs.PatchCard(ctx, mid, doneCardJSON(req, state)); err != nil {
			log.Printf("feishubridge: 卡刷新败 req=%d: %v", req, err)
		}
	}
	return &callback.CardActionTriggerResponse{
		Toast: &callback.Toast{Type: "success", Content: fmt.Sprintf("%s #%d", stateLabel[state], req)},
		Card:  &callback.Card{Type: "raw", Data: doneCard(req, state)}}, nil // 响应内嵌终态卡=点下当场灰，无 PATCH 传播窗
}

// onMessage im.message.receive_v1——窄门(chat 吻合+user 发送)→文本口令结案。
func (b *bridge) onMessage(ctx context.Context, ev *larkim.P2MessageReceiveV1) error {
	if ev.Event == nil || ev.Event.Message == nil || ev.Event.Sender == nil {
		return nil
	}
	msg := ev.Event.Message
	log.Printf("feishubridge: 收消息 chat=%s sender=%s type=%s body=%q",
		tail4(strP(msg.ChatId)), strP(ev.Event.Sender.SenderType), strP(msg.MessageType), strP(msg.Content))
	if msg.ChatId == nil || *msg.ChatId != b.chatID {
		return nil
	}
	if ev.Event.Sender.SenderType == nil || *ev.Event.Sender.SenderType != "user" {
		return nil // 机器人回环防自伤
	}
	if msg.MessageType == nil || *msg.MessageType != "text" || msg.Content == nil {
		return nil
	}
	var body struct {
		Text string `json:"text"`
	}
	if json.Unmarshal([]byte(*msg.Content), &body) != nil {
		return nil
	}
	state, req, ok := cmdFromText(body.Text)
	if !ok {
		return nil // 非口令普通对话不响
	}
	changed, err := b.resolve(ctx, req, state, "飞书文本令")
	if err != nil {
		return err
	}
	ack := fmt.Sprintf("%s #%d", stateLabel[state], req)
	if !changed {
		ack = fmt.Sprintf("#%d 已结/无此件", req)
	} else {
		b.intent(ctx, fmt.Sprintf("条件请求#%d→%s[飞书文本]", req, state))
		if mid := b.sent[req]; mid != "" {
			if err := b.fs.PatchCard(ctx, mid, doneCardJSON(req, state)); err != nil {
				log.Printf("feishubridge: 卡刷新败 req=%d: %v", req, err)
			}
			delete(b.sent, req)
		}
	}
	if _, err := b.fs.SendText(ctx, b.chatID, ack); err != nil {
		log.Printf("feishubridge: ack 发败: %v", err)
	}
	return nil
}

func main() {
	env := flag.String("env", "/opt/o-research/env", "env 文件（取 FIN_PG_DSN）")
	mkf := flag.String("mk", "/opt/o/secrets/o_master_key", "主权钥文件")
	flag.Parse()
	dsn, err := envDSN(*env, "FIN_PG_DSN")
	die(err)
	mk, err := config.ReadMasterKeyFrom(*mkf)
	die(err)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, dsn)
	die(err)
	defer pool.Close()
	var idCt, secCt []byte
	die(pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_id'").Scan(&idCt))
	die(pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_secret'").Scan(&secCt))
	appID, err := cryptoutil.Open(mk, idCt)
	die(err)
	appSec, err := cryptoutil.Open(mk, secCt)
	die(err)
	cfg := config.New(pool)
	die(cfg.Load(ctx))
	chatID, err := cfg.GetString("feishu.chat_id")
	die(err)
	if chatID == "" {
		die(fmt.Errorf("feishu.chat_id 空（未绑定）"))
	}
	b := &bridge{pool: pool, fs: notify.NewFeishu(appID, appSec), chatID: chatID,
		sent: map[int64]string{}}
	var rdsnCt []byte // 结案户 DSN（密文入库——凭据归 PG；缺=结案面 fail-closed 推送面照跑）
	if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishubridge.resolver_dsn'").Scan(&rdsnCt); err == nil {
		if rdsn, err := cryptoutil.Open(mk, rdsnCt); err == nil {
			if rp, err := pgxpool.New(ctx, rdsn); err == nil {
				b.rpool = rp
				defer rp.Close()
			}
		}
	}
	if b.rpool == nil {
		log.Printf("feishubridge: resolver_dsn 未配——结案面禁用（推送面照跑）")
	}
	go b.listenPending(ctx)                       // condreq_pending 敲门→推卡（零轮询）
	disp := dispatcher.NewEventDispatcher("", "") // ws 长连免验签/加密键
	disp.OnP2CardActionTrigger(b.onCardAction)
	disp.OnP2MessageReceiveV1(b.onMessage)
	cli := larkws.NewClient(appID, appSec,
		larkws.WithEventHandler(disp),
		larkws.WithLogLevel(larkcore.LogLevelWarn),
		larkws.WithAutoReconnect(true))
	cli.SetOnError(func(err error) { log.Printf("feishubridge: ws 败: %v", err) })
	log.Printf("feishubridge: 起——chat=%s…%s", chatID[:4], chatID[len(chatID)-4:])
	die(cli.Start(ctx))
}
