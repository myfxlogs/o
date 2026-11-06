// broker_face.go —— T-0407 座舱 broker 接入人机径（票面：凭据表单入
// T-0379 机件链——凭据落 readcred 钦定 app_secrets <broker>.<proto>.*
// 键形，config_changed 触发器原生唤链零改；前端禁直写凭据域——提交走
// 本 RPC 同源闸窄写）。
//
//	· 写面：Origin 同源闸同 earn 件→形闸（venue token/proto/login/密码/
//	  服务器全必携）→master key Seal→单事务三键 upsert（族原子落——半族
//	  =形坏凭据 fail-closed 病灶防）→ops_event 审计行（seat=dashboard-ui
//	  同 earn 件口径；note 携 venue/proto/login/server/备注——password
//	  永禁入注入日志入响应）。
//	· proto 命名空间：mt5 族=T-0379 链接管域（credscan '.mt5.' 族锚）；
//	  mt4 族=同构 .mt4. 在册——接入链 MT5 族限不接管，列表如实显形不
//	  假冒接入态（票面协议下拉落账非死件）。
//	· 读面：在册 broker=凭据族∪台账并集；login=凭据域唯一出面字段
//	  （服务端单件解密——票面「在册 broker 列表 login 列」钦定）；
//	  接入态=fin_venue_candidate 镜像+ops_event「venue接入」末拍注记
//	  +败径标；在跑策略数=fin_strategy 非 retired 行数。
package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	connect "connectrpc.com/connect/v2"

	"o/internal/gen/dashboardv1"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
)

// brokerCredEvent 凭据提交审计事件名（ops_event seat=dashboard-ui 同 earn
// 件口径——机件链「venue接入」行域不混）。
const (
	brokerCredEvent  = "venue凭据提交"
	brokerCredTicket = "T-0407"
	brokerCredActor  = "dashboard-ui"
)

// brokerVenueRE venue token 形闸（同 credscan credKeySplit 字符集——小写
// 数线横；冒号/空白/点禁入台账键域）。
var brokerVenueRE = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

// ListBrokerOnboard 在册 broker 列表读面（只读——凭据族枚举+login 单件解密
// 出面+台账态+末拍注记+在跑策略数四源并集）。
func (c coreService) ListBrokerOnboard(ctx context.Context,
	_ *dashboardv1.ListBrokerOnboardRequest) (*dashboardv1.ListBrokerOnboardResponse, error) {
	venues := map[string]*dashboardv1.BrokerOnboardRow{}
	rowOf := func(v string) *dashboardv1.BrokerOnboardRow {
		r := venues[v]
		if r == nil {
			r = &dashboardv1.BrokerOnboardRow{Venue: v}
			venues[v] = r
		}
		return r
	}
	// ① 凭据族（app_secrets <broker>.<proto>.* 键形+单例 mt5.*；login 件
	// 解密出面——password/server 密文不解）。
	rows, err := c.pool.Query(ctx, `SELECT key,ciphertext FROM app_secrets
		WHERE key LIKE 'mt5.%' OR strpos(key,'.mt5.')>0 OR strpos(key,'.mt4.')>0`)
	if err != nil {
		return nil, fmt.Errorf("凭据族枚举: %w", err)
	}
	var mk []byte
	for rows.Next() {
		var k string
		var ct []byte
		if err := rows.Scan(&k, &ct); err != nil {
			rows.Close()
			return nil, err
		}
		venue, proto, field, ok := brokerCredKeySplit(k)
		if !ok {
			continue
		}
		r := rowOf(venue)
		r.Proto = proto // 同 venue 跨族=后写盖前（实态一族一户——形坏共存如实盖）
		if field != "login" {
			continue
		}
		if mk == nil {
			if mk, err = config.ReadMasterKey(); err != nil {
				rows.Close()
				return nil, fmt.Errorf("master key: %w", err)
			}
		}
		if pt, err := cryptoutil.Open(mk, ct); err == nil {
			r.Login = pt // 解败=空串如实缺态（凭据坏由链侧 fail-closed 自报）
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// ② 台账态（fin_venue_candidate SG 镜像——md 正本投影只读面）。
	crows, err := c.pool.Query(ctx, `SELECT venue,state FROM fin_venue_candidate`)
	if err != nil {
		return nil, fmt.Errorf("台账读: %w", err)
	}
	for crows.Next() {
		var v, st string
		if err := crows.Scan(&v, &st); err != nil {
			crows.Close()
			return nil, err
		}
		rowOf(v).State = st
	}
	crows.Close()
	if err := crows.Err(); err != nil {
		return nil, err
	}
	// ③ 末拍注记（ops_event「venue接入」esc_scope=venue:login——venue 段
	// 归键，末行取注记+败径标+时刻）。
	brows, err := c.pool.Query(ctx, `SELECT DISTINCT ON (split_part(esc_scope,':',1))
		split_part(esc_scope,':',1),note,esc_class IS NOT NULL,ts
		FROM ops_event WHERE event='venue接入' AND esc_scope IS NOT NULL
		ORDER BY 1,id DESC`)
	if err != nil {
		return nil, fmt.Errorf("拍注读: %w", err)
	}
	for brows.Next() {
		var v, note string
		var failed bool
		var at time.Time
		if err := brows.Scan(&v, &note, &failed, &at); err != nil {
			brows.Close()
			return nil, err
		}
		r := rowOf(v)
		r.BeatNote = note
		r.BeatFailed = failed
		r.LastBeatAt = tsProto(at)
	}
	brows.Close()
	if err := brows.Err(); err != nil {
		return nil, err
	}
	// ④ 在跑策略数（fin_strategy 非 retired 行×venue）。
	srows, err := c.pool.Query(ctx, `SELECT venue,count(*) FROM fin_strategy
		WHERE state<>'retired' GROUP BY venue`)
	if err != nil {
		return nil, fmt.Errorf("策略数读: %w", err)
	}
	for srows.Next() {
		var v string
		var n int64
		if err := srows.Scan(&v, &n); err != nil {
			srows.Close()
			return nil, err
		}
		rowOf(v).Strategies = n
	}
	srows.Close()
	if err := srows.Err(); err != nil {
		return nil, err
	}
	resp := &dashboardv1.ListBrokerOnboardResponse{}
	for _, r := range venues {
		resp.Rows = append(resp.Rows, r)
	}
	sort.Slice(resp.Rows, func(i, j int) bool { return resp.Rows[i].Venue < resp.Rows[j].Venue })
	return resp, nil
}

// SubmitBrokerCred 凭据提交窄写面（同源闸→形闸→Seal→单事务族 upsert+
// 审计行；config_changed 触发器原生唤 T-0379 链——本件零链触码）。
func (c coreService) SubmitBrokerCred(ctx context.Context,
	req *dashboardv1.SubmitBrokerCredRequest) (*dashboardv1.SubmitBrokerCredResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	venue := req.GetVenue()
	if !brokerVenueRE.MatchString(venue) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"venue 形拒——[a-z0-9_-]{1,32} 小写数线横 token（凭据键前缀同 credscan 形闸）")
	}
	proto := req.GetProto()
	if proto == "" {
		proto = "mt5"
	}
	if proto != "mt5" && proto != "mt4" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"proto 形拒——mt5|mt4 二值")
	}
	if _, err := strconv.ParseUint(req.GetLogin(), 10, 64); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"login 形拒——MT 户号须纯数字（同 checkCredShape 形闸）")
	}
	if req.GetPassword() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "password 空——凭据必备件")
	}
	server := strings.TrimSpace(req.GetServer())
	if server == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "server 空——凭据必备件")
	}
	mk, err := config.ReadMasterKey()
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, "master key 缺——凭据写径不可用")
	}
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "tx: "+err.Error())
	}
	defer tx.Rollback(ctx)
	fields := map[string]string{
		"login":    req.GetLogin(),
		"password": req.GetPassword(),
		"server":   server,
	}
	n := 0
	for f, p := range fields {
		ct, err := cryptoutil.Seal(mk, p)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, "seal: "+err.Error())
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)
			ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext`,
			venue+"."+proto+"."+f, ct); err != nil {
			return nil, connect.NewError(connect.CodeInternal, "凭据写: "+err.Error())
		}
		n++
	}
	note := fmt.Sprintf("venue=%s proto=%s login=%s server=%s", venue, proto, req.GetLogin(), server)
	if proto == "mt4" {
		note += "（MT4 族凭据在册——接入链 MT5 族限，候 MT4 臂）"
	}
	if nb := strings.TrimSpace(req.GetNote()); nb != "" {
		note += " 备注=" + nb
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,$2,$3,$4)`, brokerCredActor, brokerCredEvent, brokerCredTicket, note); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "审计行写: "+err.Error())
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "tx commit: "+err.Error())
	}
	return &dashboardv1.SubmitBrokerCredResponse{Venue: venue, FieldsWritten: int32(n)}, nil
}

// brokerCredKeySplit 凭据键→(venue,proto,field)：单例 mt5.<field>→('mt5',
// 'mt5',field)；<broker>.mt5.<field>/<broker>.mt4.<field>→(broker,proto,
// field)。venue token 形闸同 credscan（小写数线横——冒号/空白禁入）。
func brokerCredKeySplit(k string) (string, string, string, bool) {
	if strings.HasPrefix(k, "mt5.") {
		return "mt5", "mt5", k[4:], true
	}
	var v, f string
	var ok bool
	if v, f, ok = strings.Cut(k, ".mt5."); !ok {
		if v, f, ok = strings.Cut(k, ".mt4."); !ok {
			return "", "", "", false
		}
	}
	if v == "" || f == "" {
		return "", "", "", false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return "", "", "", false
		}
	}
	proto := "mt5"
	if strings.Contains(k, ".mt4.") {
		proto = "mt4"
	}
	return v, proto, f, true
}
