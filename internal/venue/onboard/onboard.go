// Package onboard —— MT5 族 broker 接入标准径（T-0379 业主令「添加broker」）：
// 凭据落地发现→四拍流水（①探测→②映射草拟→③map 闸→④paper 激活）→
// fin_venue_candidate 状态机补尾（registered 后 probed/mapped/paper 三态，
// md_1090/fin_0118）。MT5 族通用——同 mtapi 协议 broker-agnostic 零 per-venue 码。
//
//	凭据约定（票面钉死）：app_secrets <broker>.mt5.<field>（login/password/
//	server/host——<broker> 前缀=venue 名；单例 mt5.* 为「首个 broker」=venue
//	'mt5'）。凭据域只读（密文列出 cryptoutil.Open 正径——明文禁入日志/文件）。
//	多账户粒度=<venue>:<login>（台账/观测面键——venue 列 CHECK 禁冒号，组合
//	键落 ops_event esc_scope/evidence 不落 venue 列）。
//
//	边界：凭据域/owner 类键只读——map 落闸默认业主单次批（fin_condition_gap
//	请求盒推 O 群携草案，bridge.xvenue.map owner 类键不破域；备选
//	bridge.xvenue.map.<login> 子键降域=另票议本件不降）。④激活止于
//	state='paper' 声明——paper 开户走既有 T-0377 admitXvenuePair 提案有机径
//	（零新模板零新径）；live 晋升照生命周期判据+live_switch（真单径不动）。
//
//	败径（锚④）：凭据坏/网关不可达/符号表空=fail-closed+ops_event 自报
//	（T-0369 律）+evidence 锚记 attempt/backoff——stay 'registered' 不跃迁。
//	事件面：finPool LISTEN config_changed（app_secrets 凭据落/app_config
//	map 落同帧脊）+10min 活性兜底；发现拍=凭据到达事件驱动零轮询。
package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
)

const (
	seatName  = "接市官"
	eventName = "venue接入"
	ticketID  = "T-0379"
	liveness  = 10 * time.Minute
)

var (
	probeRetryGap = 30 * time.Minute // 探测败径重试间隔（var=桩测覆写锚——var 非 const）
	maxMapPairs   = 32               // 草案映射对数上限（防全表膨胀——xvenue 谱域内足够）
)

// Session mtapi 探测会话窄面（*mt5.Client 满足；桩测注假——协议面 seam）。
type Session interface {
	Connect(ctx context.Context, cr mt5.Credentials) error
	AccountSummary(ctx context.Context) (eq, fm, bal float64, err error)
	SymbolList(ctx context.Context) ([]string, error)
	SymbolFacts(ctx context.Context, sym string) (map[string]any, error)
	Disconnect(ctx context.Context)
	Close()
}

// Runner 接入拍岗。md=PG-md 正本（fin_venue_candidate 写面——bridge 表正本侧，
// SG 侧写会被投影覆盖）；fin=SG fin（app_secrets/ops_event/fin_condition_gap
// 正本面）；cfg=md 侧 app_config 副本读（gateway/okx.symbols/bridge.xvenue.map
// 键域——投影热启延迟容忍，闸检测以 owner 写后事件拍为准）。
type Runner struct {
	md, fin *pgxpool.Pool
	cfg     *config.Client
	mk      func() ([]byte, error)
	dial    func(addr string) (Session, error)
	rej     *rejects.Door
	now     func() time.Time
}

// New 装配（mk/dial 缺省=生产径；桩测覆写）。
func New(md, fin *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{md: md, fin: fin, cfg: cfg, mk: config.ReadMasterKey,
		dial: func(addr string) (Session, error) { return mt5.Dial(addr) },
		rej:  rejects.New("onboard", "cred_read_err", "probe_failed", "map_gate_err", "write_err"),
		now:  time.Now}
}

// Run 上岗：席位注册+心跳+config_changed 事件脊（凭据/map 同帧）+活性兜底。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.fin, seatName, "venue",
		"MT5 族 broker 接入岗——凭据→探测→映射→paper 四拍", nil); err != nil {
		errc <- fmt.Errorf("%s register: %w", seatName, err)
		return
	}
	go roster.BeatLoop(ctx, r.fin, seatName, 30*time.Second, errc)
	events.ListenOrDeadline(ctx, r.fin, "venue-onboard", []string{"config_changed"},
		func(context.Context) time.Duration { return liveness }, r.scan)
}

// scan 一拍：凭据族发现→各 venue 状态机推进（循环至一拍内不可再进——
// 事件拍内全链可达，幂等行重跑零副作用）。
func (r *Runner) scan(ctx context.Context) error {
	brokers, err := r.discover(ctx)
	if err != nil {
		return err
	}
	for _, b := range brokers {
		if err := r.ensureCandidate(ctx, b.venue); err != nil {
			return err
		}
		if err := r.advance(ctx, b, 0); err != nil {
			log.Printf("[onboard] %s 推进败: %v", b.venue, err)
		}
	}
	// 已入册 venue（creds 撤/历史态残留）——mapped 闸检测不赖凭据在册（
	// evidence 已载探测符号表），独立推进防停滞。
	rows, err := r.md.Query(ctx, `SELECT venue FROM fin_venue_candidate WHERE state='mapped'`)
	if err != nil {
		return err
	}
	var mapped []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		mapped = append(mapped, v)
	}
	rows.Close()
	for _, v := range mapped {
		if _, ok := brokers[v]; ok {
			continue
		}
		if err := r.advance(ctx, credFamily{venue: v, missing: true}, 0); err != nil {
			log.Printf("[onboard] %s mapped 闸检测败: %v", v, err)
		}
	}
	return nil
}

// Advance 单 venue 推进（干跑工具/桩测面——maxBeat>0 截断拍数：EXR 3001157
// 首案干跑①②=beats≤2「探测+草案不激活」）。
func (r *Runner) Advance(ctx context.Context, venue string, maxBeat int) error {
	b := credFamily{venue: venue, missing: true}
	if fam, err := r.discoverVenue(ctx, venue); err == nil {
		b = fam
	}
	return r.advance(ctx, b, maxBeat)
}

// advance 状态机推进环（一拍至多跃迁至不可进/败止——每拍 ops_event 落行）。
func (r *Runner) advance(ctx context.Context, b credFamily, maxBeat int) error {
	for beat := 1; ; beat++ {
		if maxBeat > 0 && beat > maxBeat {
			return nil
		}
		state, onboard, err := r.stateOf(ctx, b.venue)
		if err != nil {
			return err
		}
		next, err := r.beat(ctx, b, state, onboard)
		if err != nil {
			return err // 结构败（库面）——上抛；业务败已 fail-closed 落行不跃迁
		}
		if !next {
			return nil
		}
	}
}

// beat 单拍迁移（返 true=已跃迁续环）。state=registered→探测；probed→草案；
// mapped→map 闸；paper/其他=止。
func (r *Runner) beat(ctx context.Context, b credFamily, state string, ob map[string]any) (bool, error) {
	switch state {
	case "registered":
		return r.beatProbe(ctx, b, ob)
	case "probed":
		return r.beatDraft(ctx, b, ob)
	case "mapped":
		return r.beatGate(ctx, b, ob)
	default:
		return false, nil
	}
}

// stateOf venue 当前态+evidence.onboard 子档（行缺=registered 虚态——发现拍
// ensureCandidate 先行，防御形）。
func (r *Runner) stateOf(ctx context.Context, venue string) (string, map[string]any, error) {
	var state string
	var evidence []byte
	err := r.md.QueryRow(ctx,
		`SELECT state, COALESCE(evidence,'{}'::jsonb) FROM fin_venue_candidate WHERE venue=$1`,
		venue).Scan(&state, &evidence)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return "registered", map[string]any{}, nil
		}
		return "", nil, err
	}
	var ev map[string]any
	_ = json.Unmarshal(evidence, &ev)
	ob, _ := ev["onboard"].(map[string]any)
	if ob == nil {
		ob = map[string]any{}
	}
	return state, ob, nil
}

// ensureCandidate 凭据落=注册既成——行缺补 registered（scout 册 venue 幂等插行
// 同形；在册行任何态不覆写）。
func (r *Runner) ensureCandidate(ctx context.Context, venue string) error {
	ev, _ := json.Marshal(map[string]any{"onboard": map[string]any{
		"discovered_at": r.now().UTC().Format(time.RFC3339), "source": "cred_scan"}})
	_, err := r.md.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,evidence)
		VALUES($1,'cfd-broker','registered',$2::jsonb) ON CONFLICT (venue) DO NOTHING`, venue, ev)
	return err
}

// trans 迁移落账：state 更新+evidence.onboard 子档合并+ops_event「venue接入」行
// （同拍双落——业务败径用 fail 注记态不跃迁，state="" 保原态）。
func (r *Runner) trans(ctx context.Context, venue, login, state, beat string, ob map[string]any, note string, fail bool) error {
	obJSON, _ := json.Marshal(ob)
	if state != "" {
		if _, err := r.md.Exec(ctx, `UPDATE fin_venue_candidate
			SET state=$2, evidence=evidence || jsonb_build_object('onboard', COALESCE(evidence->'onboard','{}'::jsonb) || $1::jsonb)
			WHERE venue=$3`, obJSON, state, venue); err != nil {
			return err
		}
	} else {
		if _, err := r.md.Exec(ctx, `UPDATE fin_venue_candidate
			SET evidence=evidence || jsonb_build_object('onboard', COALESCE(evidence->'onboard','{}'::jsonb) || $1::jsonb)
			WHERE venue=$2`, obJSON, venue); err != nil {
			return err
		}
	}
	scope := venue + ":" + login
	esc := interface{}(nil)
	if fail {
		esc = "接入败"
	}
	_, err := r.fin.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,$2,$3,$4,$5,$6)`, seatName, eventName, ticketID,
		fmt.Sprintf("beat=%s %s=%s %s", beat, "key", scope, note), esc, scope)
	if err != nil {
		// T-0402：写败不静默——WARN 留痕+rej 计数+升件兜底（拒权时兜底同败，
		// WARN 仍出；闸面 tolerate 式吞错=本件病灶禁再犯）。
		log.Printf("WARN [onboard] ops_event 写败 venue=%s beat=%s（事件行丢——拒权/库败类不静默）: %v", venue, beat, err)
		r.rej.Inc("write_err")
		_, _ = r.fin.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			SELECT $1,'升件',$2,$3,'写败',$4
			WHERE NOT EXISTS (SELECT 1 FROM ops_event
				WHERE event='升件' AND ticket=$2 AND esc_scope=$4)`, seatName, ticketID,
			fmt.Sprintf("venue接入事件行写败: %v", err), scope)
	}
	return err
}
