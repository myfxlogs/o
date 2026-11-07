// probe.go —— ⑤漂移探针（T-0432）：dialect 行全谱周期对拍——SymbolParams
// 语义指纹漂移（broker 改后缀/下架/换组/ctVal 变=单位错配病重演防）+GetQuote
// 探活（报价径断）。异常=ops_event 升件（esc_class='venue漂移'——业主飞书
// 可见径）——监测面不跃迁状态机（paper 照跑，执行缝 fail-closed 自拒）。
// 升件去重：同事件同域末行同签名=skip（事件簿防洪——T-0370 收缩同律）；
// 会话拨号缝独立窄面（Session 探测窄面不含 Spec/Quote——禁宽化既有缝）。
package onboard

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"o/internal/mt5"
)

// probeSession 漂移探针会话窄缝（mt5.ProbeAdapter 满足；桩测注假——禁桩真
// venue 同律；独立窄面不宽化 Session 探测缝。hash/ctVal 标量出缝——proto
// 型不出 mt5 包界，D-514 方向白名单守 onboard 禁直引 gen 面）。
type probeSession interface {
	Connect(ctx context.Context, cr mt5.Credentials) error
	SpecHashOf(ctx context.Context, symbol string) (hash string, ctVal float64, err error)
	GetQuoteOne(ctx context.Context, symbol string) (mt5.QuoteSnapshot, error)
	Disconnect(ctx context.Context)
	Close()
}

// probePace 逐行 SymbolParams/GetQuote 步进限速（catalogPace 同款——var=桩测
// 覆写锚）；probeBudget 单户拍死线（网关/venue IO 帽——逐行限速内自然兜底）。
var (
	probePace   = 120 * time.Millisecond
	probeBudget = 45 * time.Second
)

// probeEscClass 探针升件类（ops_event esc_class——升件面同域聚类）。
const probeEscClass = "venue漂移"

// probePass 一拍探针：凭据在册族逐户——dialect 行全谱对拍（无行=skip——
// 映射未落无漂移面）。败=日志留痕不上抛（监测面败不阻主链——scan 主务
// 状态机推进零依赖本拍）。
func (r *Runner) probePass(ctx context.Context, brokers map[string]credFamily) {
	for _, b := range brokers {
		if b.missing {
			continue // 凭据撤——拨号无从，漂移监测随撤如实缺席
		}
		if err := r.probeVenue(ctx, b); err != nil {
			log.Printf("WARN [onboard] %s 漂移探针拍败: %v", b.venue, err)
		}
	}
}

// probeVenue 单户探针拍：dialect 行全谱→SymbolSpec 指纹对拍+GetQuote 探活
// （逐行限速步进）→漂移/死价集=升件（去重同签名）。
func (r *Runner) probeVenue(ctx context.Context, b credFamily) error {
	login := b.fields["login"]
	if login == "" {
		return nil // 多账户粒度键缺——无绑账户无漂移面
	}
	rows, err := r.dialectRows(ctx, b.venue, login)
	if err != nil || len(rows) == 0 {
		return err // 无映射行=无探针面（映射=确认件——向导第④步前置）
	}
	sess, err := r.dialProbe(ctx, b)
	if err != nil {
		// 拨号败=全户探活断——网关/凭据病（T-0379 败径升件同域不重复：
		// 接入链自有 stay 行）。探针面只记日志——接入链 beat 自报在案。
		return fmt.Errorf("拨号: %w", err)
	}
	defer sess.Close()
	var drifted, dead []string
	for i, row := range rows {
		if i > 0 && probePace > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(probePace): // sched-exempt-ok：逐行限速步进（限频面守——catalogPace 同款拍内步进非调度原语）
			}
		}
		h, ctVal, err := sess.SpecHashOf(ctx, row.VenueSymbol)
		if err != nil {
			drifted = append(drifted, fmt.Sprintf("%s: SymbolParams 败(%v)——下架/换组疑", row.Canonical, err))
			continue
		}
		if h != row.SpecHash {
			drifted = append(drifted, fmt.Sprintf("%s: 指纹漂移(%s→%s) ctVal=%g→%g——broker 改面/单位错配病防",
				row.Canonical, shortHash(row.SpecHash), shortHash(h), row.CtVal, ctVal))
		}
		q, err := sess.GetQuoteOne(ctx, row.VenueSymbol)
		if err != nil || q.Mid() <= 0 {
			dead = append(dead, row.Canonical)
		}
	}
	if len(drifted) > 0 {
		r.escalateDedup(ctx, b.venue, login, "方言漂移", drifted)
	}
	if len(dead) > 0 {
		r.escalateDedup(ctx, b.venue, login, "方言探活断", dead)
	}
	return nil
}

// escalateDedup 升件+去重：同 event+esc_scope 末行注记同签名=skip（已升在案
// 不重发——同态拍间重复禁）；新签名=落行。签名=排序后逗连（集等判——序无关）。
func (r *Runner) escalateDedup(ctx context.Context, venue, login, event string, items []string) {
	sig := strings.Join(sorted(items), ",")
	scope := venue + ":" + login
	var last string
	err := r.fin.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event=$1 AND esc_scope=$2 ORDER BY ts DESC LIMIT 1`, event, scope).Scan(&last)
	if err == nil && strings.Contains(last, sig) {
		return // 同签名已升在案——skip（事件簿防洪）
	}
	note := fmt.Sprintf("beat=漂移探针 key=%s %s", scope, sig)
	if _, err := r.fin.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,$2,$3,$4,$5,$6)`, seatName, event, ticketID, note, probeEscClass, scope); err != nil {
		// T-0402 同律：写败不静默。
		log.Printf("WARN [onboard] 漂移升件写败 venue=%s event=%s: %v", venue, event, err)
		r.rej.Inc("write_err")
	}
}

// dialProbe 探针会话拨号（凭据族实测——r.dialProbeFn 缝 nil=mt5.Dial 生产径；
// probeBudget 死线——网关/venue IO 帽）。
func (r *Runner) dialProbe(ctx context.Context, b credFamily) (probeSession, error) {
	addr, err := r.cfgStr(ctx, "mt5.gateway_addr")
	if err != nil {
		return nil, fmt.Errorf("mt5.gateway_addr: %w", err)
	}
	dial := r.dialProbeFn
	if dial == nil {
		dial = func(addr string) (probeSession, error) {
			c, err := mt5.Dial(addr)
			if err != nil {
				return nil, err
			}
			return mt5.ProbeAdapter{C: c}, nil
		}
	}
	sess, err := dial(addr)
	if err != nil {
		return nil, err
	}
	login, _ := strconv.ParseUint(b.fields["login"], 10, 64)
	pctx, cancel := context.WithTimeout(ctx, probeBudget)
	defer cancel()
	if err := sess.Connect(pctx, mt5.Credentials{
		Login:    login,
		Password: b.fields["password"],
		Server:   b.fields["server"],
		Host:     b.fields["host"]}); err != nil {
		sess.Close()
		return nil, fmt.Errorf("connect: %w", err)
	}
	return sess, nil
}

// dialectRows 方言行读件（fin 链仓——var=桩测缝，单测注假仓）。
func (r *Runner) dialectRows(ctx context.Context, venue, login string) ([]mt5.DialectRow, error) {
	if r.dialectRowsFn != nil {
		return r.dialectRowsFn(ctx, venue, login)
	}
	return mt5.NewDialectStore(r.fin).Rows(ctx, venue, login)
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func shortHash(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	return h
}
