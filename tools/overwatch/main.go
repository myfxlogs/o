// overwatch —— 外监工哨（业主「随时把握 system 工作情况」的机械层）：
// 日拍扫 SQL 可判指标 → ① ops_event 监账行（哨报=每拍 digest、哨:=旗标
// 24h 值指纹抑、里程碑=一次性）② 飞书活卡原地刷（首建存 mid→PatchCard
// 每拍定点刷；bad 旗首火附文本推屏）。判词类=scripts/watch/OVERWATCH.md。
//
// 用法（同 ops_relay 栈）：
//
//	DB_DSN|FIN_PG_DSN=postgres://... O_MASTER_KEY_FILE=/opt/o/secrets/o_master_key \
//	  go run ./tools/overwatch          # 全径
//	go run ./tools/overwatch -nofeishu  # 只 PG扫+账行
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/notify"
)

type metric struct{ label, status, value string } // status: ok|warn|bad|info
type flag struct {
	slug, note, level string // level: warn|bad|mile
	fresh             bool   // 本拍首火(抑窗内无同slug)
}

func main() {
	noFeishu := len(os.Args) > 1 && os.Args[1] == "-nofeishu"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = os.Getenv("FIN_PG_DSN")
	}
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "overwatch: DB_DSN/FIN_PG_DSN 空——fail-closed")
		os.Exit(1)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "overwatch: dsn:", err)
		os.Exit(1)
	}
	defer pool.Close()

	var ms []metric
	var flags []flag
	one := func(l, s string, v any) { ms = append(ms, metric{l, s, fmt.Sprint(v)}) }
	num := func(q string, a ...any) int64 {
		var n int64
		if err := pool.QueryRow(ctx, q, a...).Scan(&n); err != nil {
			fmt.Fprintln(os.Stderr, "overwatch: 查询败:", err)
			return -1
		}
		return n
	}
	str := func(q string, a ...any) (s string) {
		_ = pool.QueryRow(ctx, q, a...).Scan(&s)
		return
	}
	addf := func(slug, level, note string) { flags = append(flags, flag{slug, note, level, false}) }
	st := func(cond bool) string {
		if cond {
			return "warn"
		}
		return "ok"
	}

	// ── 信号面 ──
	openEsc := num(`SELECT count(*) FROM ops_event e WHERE e.event='升件'
		AND e.ts>now()-interval '7 days' AND NOT EXISTS
		(SELECT 1 FROM ops_event x WHERE x.ts>e.ts AND x.note LIKE '%升件#'||e.id::text||'%')`)
	oldEscH := num(`SELECT COALESCE(round(max(extract(epoch from (now()-ts))/3600)),0)::bigint FROM ops_event e
		WHERE e.event='升件' AND e.ts>now()-interval '7 days' AND NOT EXISTS
		(SELECT 1 FROM ops_event x WHERE x.ts>e.ts AND x.note LIKE '%升件#'||e.id::text||'%')`)
	if openEsc > 0 && oldEscH > 24 {
		addf("esc-backlog", "warn", fmt.Sprintf("哨:esc-backlog 未决升件%d件最陈%dh——信号面积压", openEsc, oldEscH))
	}
	one("未决升件7d", st(openEsc > 0 && oldEscH > 24), fmt.Sprintf("%d（最陈%dh）", openEsc, oldEscH))

	// ── 工单面 ──
	const openTkSQL = `FROM ops_event t WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='处置' AND x.ts>t.ts
		AND x.note LIKE '%工单#'||t.id::text||'%')`
	openTk := num(`SELECT count(*) ` + openTkSQL)
	overdueTk := num(`SELECT count(*) ` + openTkSQL + `
		AND t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' < now()`)
	if overdueTk > 0 {
		ids := str(`SELECT string_agg(id::text,'/') FROM (SELECT t.id ` + openTkSQL + `
			AND t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' < now()
			ORDER BY t.id LIMIT 5) x`)
		addf("ticket-overdue", "warn", fmt.Sprintf("哨:ticket-overdue %d张过限未销(#%s)——认领尸/挂起无注/真滞分裁", overdueTk, ids))
	}
	one("工单面", st(overdueTk > 0), fmt.Sprintf("开%d 过限%d", openTk, overdueTk))

	// ── 唤起链 ──
	wake24 := num(`SELECT count(*) FROM ops_event WHERE event='唤起' AND seat='唤知机' AND ts>now()-interval '24 hours'`)
	noClose := num(`SELECT count(*) FROM ops_event e WHERE e.event='唤起' AND e.seat='唤知机'
		AND e.ts BETWEEN now()-interval '24 hours' AND now()-interval '2 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='会话收口'
			AND x.ts BETWEEN e.ts AND e.ts+interval '2 hours')`)
	wakeBad := wake24 >= 5 && noClose*100/wake24 > 30
	if wakeBad {
		addf("wake-no-close", "warn", fmt.Sprintf("哨:wake-no-close 24h唤起%d无收口%d(%d%%)——会话死中率超阈",
			wake24, noClose, noClose*100/wake24))
	}
	one("唤起24h", st(wakeBad), fmt.Sprintf("%d唤 %d无收口", wake24, noClose))

	// 票→唤延迟（末张工单→其后首个唤起,分钟;-1=2h内无唤）
	tk2wake := num(`SELECT COALESCE(round(extract(epoch from (w.mts-t.ts))/60)::bigint,-1)
		FROM ops_event t LEFT JOIN LATERAL
			(SELECT min(ts) mts FROM ops_event WHERE event='唤起' AND ts>t.ts AND ts<t.ts+interval '2 hours') w ON true
		WHERE t.event='工单' ORDER BY t.id DESC LIMIT 1`)
	tkSt, tkVal := "ok", fmt.Sprintf("%dmin", tk2wake)
	if tk2wake < 0 {
		tkSt, tkVal = "warn", "无唤>2h"
	} else if tk2wake > 20 {
		tkSt = "warn"
		addf("ticket-wake-lag", "warn", fmt.Sprintf("哨:ticket-wake-lag 末票→唤%dmin——mech帧白名单未生效疑面", tk2wake))
	}
	one("票→唤", tkSt, tkVal)

	// ── 钱径活性（fin_* 表 ops户无读权——从量表行解析,免授权面）──
	posNow := str(`SELECT substring(note from '仓([0-9]+)') FROM ops_event
		WHERE event='量表' AND ticket='self_state' ORDER BY id DESC LIMIT 1`)
	one("在仓", st(posNow == "0" || posNow == ""), posNow)
	near := str(`SELECT substring(note from '近阈 \S+ ([0-9]+)%') FROM ops_event
		WHERE event='量表' AND ticket='armed_distance' ORDER BY id DESC LIMIT 1`)
	one("armed近阈", "info", map[bool]string{true: near + "%", false: "无近阈"}[near != ""])
	earn := str(`SELECT substring(note from '赚\[([^]]*)\]') FROM ops_event
		WHERE note LIKE '%策略盈亏归因%' ORDER BY id DESC LIMIT 1`)
	if earn == "" {
		one("赚[]", "bad", "空")
	} else {
		one("赚[]", "ok", "非空")
		addf("earn-nonempty", "mile", fmt.Sprintf("哨:earn-nonempty 赚[]首非空——%s（破局形态首现）", earn))
	}
	ret24 := num(`SELECT count(*) FROM ops_event WHERE ts>now()-interval '24 hours'
		AND (note LIKE '%fin_retire%' OR note LIKE '%已退役%' OR note LIKE '%stratretire%')`)
	if ret24 > 3 {
		addf("retire-surge", "bad", fmt.Sprintf("哨:retire-surge 24h退役%d件——核是否误退armed设计态（#13081口径:只许执行断/腿缺证据）", ret24))
	}
	one("退役24h", st(ret24 > 3), ret24)

	// ── 前沿/数据面 ──
	kline := num(`SELECT count(*) FROM information_schema.tables
		WHERE table_name ILIKE '%kline%' OR table_name ILIKE '%candle%' OR table_name ILIKE '%ohlc%'`)
	one("日K数据面", map[bool]string{true: "ok", false: "info"}[kline > 0],
		map[bool]string{true: fmt.Sprintf("%d表", kline), false: "无"}[kline > 0])
	stuck := num(`SELECT count(*) FROM ops_event WHERE event='升件' AND esc_scope='stuck_esc'
		AND ts>now()-interval '24 hours'`)
	if stuck > 5 {
		addf("stuck-esc-tax", "warn", fmt.Sprintf("哨:stuck-esc-tax 24h复火%d——支名复用税超阈,dedup键该修", stuck))
	}
	one("stuck_esc24h", st(stuck > 5), stuck)
	econAge := num(`SELECT COALESCE(round(extract(epoch from (now()-max(ts)))/86400),-1)::bigint FROM fin_econ_lesson`)
	if econAge > 5 {
		addf("econ-miner-dry", "warn", fmt.Sprintf("哨:econ-miner-dry %dd零新产——矿写径/触发面查", econAge))
	}
	one("econ矿龄", st(econAge > 5), fmt.Sprintf("%dd", econAge))
	one("库体积", "info", str(`SELECT pg_size_pretty(pg_database_size('o'))`))

	// ── 落账 ──
	insRow := func(note string) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note) VALUES('监工','监账','',$1)`, note); err != nil {
			fmt.Fprintln(os.Stderr, "overwatch: 监账写败:", err)
		}
	}
	var dg strings.Builder
	dg.WriteString("哨报|")
	for _, m := range ms {
		fmt.Fprintf(&dg, "%s=%s ", m.label, m.value)
	}
	insRow(dg.String())
	for i, f := range flags {
		win := ` AND ts>now()-interval '24 hours'` // 旗标=日窗抑
		if f.level == "mile" {
			win = "" // 里程碑=永不重火
		}
		dup := num(`SELECT count(*) FROM ops_event WHERE seat='监工' AND event='监账'
			AND note LIKE '%'||$1||'%'`+win, f.slug)
		if dup == 0 {
			insRow(f.note)
			flags[i].fresh = true
		}
	}

	// ── 飞书活卡 ──
	if noFeishu {
		return
	}
	mk, err := config.ReadMasterKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "overwatch: 主权钥:", err)
		os.Exit(1)
	}
	getSecret := func(k string) (p string) {
		var ct []byte
		if pool.QueryRow(ctx, `SELECT ciphertext FROM app_secrets WHERE key=$1`, k).Scan(&ct) == nil {
			p, _ = cryptoutil.Open(mk, ct)
		}
		return
	}
	appID, appSec := getSecret("feishu.app_id"), getSecret("feishu.app_secret")
	if appID == "" || appSec == "" {
		fmt.Fprintln(os.Stderr, "overwatch: feishu凭据缺——卡面跳过（账行已落）")
		return
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "overwatch: config:", err)
		os.Exit(1)
	}
	chatID, _ := cfg.GetString("feishu.chat_id")
	if chatID == "" {
		fmt.Fprintln(os.Stderr, "overwatch: feishu.chat_id 未绑定——卡面跳过")
		return
	}
	fs := notify.NewFeishu(appID, appSec)

	worst := "green" // 头色=最重级:bad→red warn→orange
	for _, m := range ms {
		if m.status == "bad" {
			worst = "red"
		} else if m.status == "warn" && worst == "green" {
			worst = "orange"
		}
	}
	for _, f := range flags {
		if f.level == "bad" {
			worst = "red"
		} else if f.level == "warn" && worst == "green" {
			worst = "orange"
		}
	}
	var flds []string
	icon := map[string]string{"ok": "🟢", "warn": "🟠", "bad": "🔴", "info": "⚪"}
	for _, m := range ms {
		flds = append(flds, fmt.Sprintf(
			`{"is_short":true,"text":{"tag":"lark_md","content":"**%s**\n%s %s"}}`,
			m.label, icon[m.status], m.value))
	}
	fic := map[string]string{"warn": "🟠", "bad": "🔴", "mile": "⭐"}
	var fb strings.Builder
	if len(flags) == 0 {
		fb.WriteString("🟢 无哨旗")
	}
	for i, f := range flags {
		if i > 0 {
			fb.WriteString(`\n`)
		}
		fmt.Fprintf(&fb, "%s %s", fic[f.level], f.note)
	}
	card := fmt.Sprintf(`{"config":{"wide_screen_mode":true},
		"header":{"template":"%s","title":{"tag":"plain_text","content":"O 监工哨 · %s"}},
		"elements":[{"tag":"div","fields":[%s]},{"tag":"hr"},
		{"tag":"div","text":{"tag":"lark_md","content":"**哨旗**\n%s"}},
		{"tag":"note","elements":[{"tag":"plain_text","content":"6h拍|史=ops_event event=监账|判读哨=scripts/watch/OVERWATCH.md"}]}]}`,
		worst, time.Now().Format("01-02 15:04"), strings.Join(flds, ","), fb.String())

	home, _ := os.UserHomeDir()
	midFile := filepath.Join(home, ".local/state/o-worker/overwatch_card_mid")
	mid, _ := os.ReadFile(midFile)
	midStr := strings.TrimSpace(string(mid))
	patched := midStr != "" && fs.PatchCard(ctx, midStr, card) == nil
	if !patched {
		nm, err := fs.CreateInteractive(ctx, chatID, card)
		if err != nil {
			fmt.Fprintln(os.Stderr, "overwatch: 建卡败:", err)
			return
		}
		_ = os.MkdirAll(filepath.Dir(midFile), 0o755)
		_ = os.WriteFile(midFile, []byte(nm), 0o600)
	}
	for _, f := range flags { // bad 旗首火附文本推屏（卡留痕,屏要到人）
		if f.level == "bad" && f.fresh {
			if _, err := fs.SendText(ctx, chatID, f.note); err != nil {
				fmt.Fprintln(os.Stderr, "overwatch: urgent推屏败:", err)
			}
		}
	}
}
