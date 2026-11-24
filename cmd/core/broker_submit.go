// broker_submit.go —— T-0407/T-0430 座舱 broker 接入写面（三步向导
// verify-then-write——凭据提交点先跑验连：MT5=mtapi connect+AccountSummary
// 实测（mt-accounts skill 钦定形：未获正确回包一律拒写零落表）、CEX=venue
// 只读端实测（okx=/api/v5/account/balance）；败=拒写即回原因，无效凭据
// 零落表——先验后开写，零行保证构造性成立）。凭据落 readcred 钦定
// app_secrets <broker>.<proto>.* 键形（mt5=login/password/server+host 选填；
// cex=api_key/api_secret/passphrase 三件对形），config_changed 触发器原生
// 唤 T-0379 链零改；前端禁直写凭据域——提交走本 RPC 同源闸窄写。
//
//	· 写面序：Origin 同源闸同 earn 件→形闸（mt5：venue token/login/密码/
//	  服务器全必携 host 选填；cex：venue∈册〔首档 okx〕+api 三件必携）
//	  →verify→master key Seal→单事务族 upsert（族原子落——半族=形坏凭据
//	  fail-closed 病灶防）→ops_event 审计行（seat=dashboard-ui 同 earn 件
//	  口径，esc_scope=venue——读面 phase 派生源；note 携 venue/proto/login/
//	  verify 摘要——password/api_key/api_secret/passphrase 永禁入注入日志
//	  入响应）。
//	· proto 命名空间：mt5 族=T-0379 链接管域（credscan '.mt5.' 族锚）；cex
//	  族=T-0430 新面（<venue>.cex.*）；mt4 族=存量遗留（业主令 2026-11-07
//	  不处理——提交闸拒，读面兼容见 broker_face.go）。
//	· 验连 seam：dialSession/cexBalance（coreService 注入缝——nil=生产径；
//	  桩测注假在 seam 注 fake Session/cexBalance，禁桩真 venue）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect/v2"
	"github.com/jackc/pgx/v5"

	"o/internal/data/okxpriv"
	"o/internal/gen/dashboardv1"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/signutil"
	"o/internal/trading/okxexec"
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

// brokerCredNS cred_ns 推导（O-37450 roster upsert 同源——roster 行在=
// cred_ns 真源沿用；裸键族在=首户兼容 cred_ns=venue 名〔存量 mt5/okx/
// okx_read 无中缀形〕；否则 venue+'.'+proto 新户键前缀——与凭据键写
// `<cred_ns>.<field>` 同式自洽）。
func brokerCredNS(ctx context.Context, tx pgx.Tx, venue, proto string) (string, error) {
	var ns string
	err := tx.QueryRow(ctx, `SELECT cred_ns FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&ns)
	if err == nil {
		return ns, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	probe := venue + "." + map[string]string{"mt5": "login", "cex": "api_key"}[proto]
	var bare bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_secrets WHERE key=$1)`, probe).Scan(&bare); err != nil {
		return "", err
	}
	if bare {
		return venue, nil
	}
	return venue + "." + proto, nil
}

// mt5VerifySession mtapi 验连会话窄面（onboard.Session 同构窄化本地面——
// *mt5.Client 结构满足；桩测注假 seam：fake Session 喂真形回包，禁桩真
// venue）。
type mt5VerifySession interface {
	Connect(ctx context.Context, cr mt5.Credentials) error
	AccountSummary(ctx context.Context) (eq, fm, bal float64, err error)
	Disconnect(ctx context.Context)
	Close()
}

// verifyBudget 提交验连死线（网关/venue IO 帽——调用方无更短死线时兜底）。
var verifyBudget = 45 * time.Second

// dialSessionDefault 生产径：mtapi 网关拨号（onboard 同 addr 键——
// mt5.gateway_addr owner 册）。
func dialSessionDefault(addr string) (mt5VerifySession, error) {
	return mt5.Dial(addr)
}

// verifyMT5 MT5 验连：connect+AccountSummary 实测（skill canon 序——无正确
// 回包=无效凭据拒写）。
func (c coreService) verifyMT5(ctx context.Context, login uint64, password, server, host string) (string, error) {
	dial := c.dialSession
	if dial == nil {
		dial = dialSessionDefault
	}
	addr, err := c.cfg.GetString("mt5.gateway_addr")
	if err != nil {
		return "", fmt.Errorf("mt5.gateway_addr: %w", err)
	}
	sess, err := dial(addr)
	if err != nil {
		return "", fmt.Errorf("网关不可达 dial: %w", err)
	}
	defer sess.Close()
	vctx, cancel := context.WithTimeout(ctx, verifyBudget)
	defer cancel()
	if err := sess.Connect(vctx, mt5.Credentials{Login: login,
		Password: password, Server: server, Host: host}); err != nil {
		return "", fmt.Errorf("connect: %w", err)
	}
	defer sess.Disconnect(context.Background())
	eq, _, bal, err := sess.AccountSummary(vctx)
	if err != nil {
		return "", fmt.Errorf("AccountSummary: %w", err)
	}
	return fmt.Sprintf("connect ok eq=%.2f bal=%.2f", eq, bal), nil
}

// verifyCEX CEX 验连：CexVenue adapter verify 面（O-37450——册内 venue
// fail-closed 不猜端点；桩测 cexBalance seam 优先，生产径=adapter
// Verify→okxexec.BalanceEquity 只读端实测 totalEq 面值出摘要）。
func (c coreService) verifyCEX(ctx context.Context, venue string, apiKey, apiSecret, passphrase string) (string, error) {
	creds := signutil.Credentials{Key: apiKey, Secret: apiSecret, Passphrase: passphrase}
	if bal := c.cexBalance; bal != nil {
		m, err := bal(ctx, creds)
		if err != nil {
			return "", err
		}
		var total float64
		for _, v := range m {
			total += v
		}
		return fmt.Sprintf("balance ok totalEq=%.2f", total), nil
	}
	ad, ok := okxpriv.CexVenueFor(venue)
	if !ok {
		return "", fmt.Errorf("venue 册未收——CexVenue adapter 无 %s（fail-closed 不猜端点）", venue)
	}
	return ad.Verify(ctx, creds)
}

// SubmitBrokerCred 凭据提交窄写面（同源闸→形闸→verify-then-write→Seal→
// 单事务族 upsert+审计行；config_changed 触发器原生唤 T-0379 链——本件零
// 链触码；验败=CodeInvalidArgument 即回原因，零凭据落库零审计行）。
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
	if proto != "mt5" && proto != "cex" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"proto 形拒——mt5|cex 两值（MT4 不处理——业主令 2026-11-07，存量 .mt4. 键读面兼容）")
	}
	var fields map[string]string
	var note, verifyNote string
	switch proto {
	case "mt5":
		login, err := strconv.ParseUint(req.GetLogin(), 10, 64)
		if err != nil {
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
		host := strings.TrimSpace(req.GetHost())
		verifyNote, err = c.verifyMT5(ctx, login, req.GetPassword(), server, host)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, "验连败——凭据未落库: "+err.Error())
		}
		fields = map[string]string{"login": req.GetLogin(), "password": req.GetPassword(), "server": server}
		if host != "" {
			fields["host"] = host
		}
		note = fmt.Sprintf("venue=%s proto=mt5 login=%s server=%s %s", venue, req.GetLogin(), server, verifyNote)
	case "cex":
		if _, ok := okxpriv.CexVenueFor(venue); !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				"venue 册未收——CexVenue adapter 无此族（他所见 venue 册再扩，fail-closed 不猜端点）")
		}
		apiKey, apiSecret, passphrase := req.GetApiKey(), req.GetApiSecret(), req.GetPassphrase()
		if apiKey == "" || apiSecret == "" || passphrase == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				"api_key/api_secret/passphrase 三件必携——CEX 凭据族形闸")
		}
		vn, err := c.verifyCEX(ctx, venue, apiKey, apiSecret, passphrase)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, "验连败——凭据未落库: "+err.Error())
		}
		verifyNote = vn
		fields = map[string]string{"api_key": apiKey, "api_secret": apiSecret, "passphrase": passphrase}
		note = fmt.Sprintf("venue=%s proto=cex %s", venue, verifyNote)
	}
	if nb := strings.TrimSpace(req.GetNote()); nb != "" {
		note += " 备注=" + nb
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
	// 墓碑闸（O-38013——removed=终态不可复活：向导重交同 venue token 须
	// fail-closed 拒写，防凭据族复活+roster upsert 误触；复挂=新 venue
	// token 新行同 SetBrokerState 律）。
	var rstate string
	if err := tx.QueryRow(ctx, `SELECT state FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&rstate); err == nil {
		if rstate == "removed" {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				"venue="+venue+" 已 removed 墓碑——凭据向导拒写（终态不可复活，复挂=新 venue token）")
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, "roster 态读: "+err.Error())
	}
	// cred_ns 推导（O-37450——凭据键前缀=名册 cred_ns 同源：roster 行在
	// 沿用/裸键族首户兼容/新户 venue+'.'+proto；键写 `<cred_ns>.<field>`）。
	credNS, err := brokerCredNS(ctx, tx, venue, proto)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "cred_ns 推导: "+err.Error())
	}
	for f, p := range fields {
		ct, err := cryptoutil.Seal(mk, p)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, "seal: "+err.Error())
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)
			ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext`,
			credNS+"."+f, ct); err != nil {
			return nil, connect.NewError(connect.CodeInternal, "凭据写: "+err.Error())
		}
	}
	// roster upsert 同事务（O-37450 增径——凭据族+名册原子落；验连败上方
	// 已拒零行；state/roles 冲突不覆盖——人工停用/授权面不被提交面改写，
	// 新行 state='connecting' 待会话环自报接管，roles 缺省 {collect}）。
	var loginArg any
	if proto == "mt5" {
		loginArg = req.GetLogin()
	}
	meta, _ := json.Marshal(map[string]string{"verify_note": verifyNote, "submit_ticket": brokerCredTicket})
	if _, err := tx.Exec(ctx, `INSERT INTO fin_venue_roster(venue,proto,cred_ns,login,roles,state,meta)
		VALUES($1,$2,$3,$4,'{collect}','connecting',$5)
		ON CONFLICT (venue) DO UPDATE SET cred_ns=EXCLUDED.cred_ns,
		login=EXCLUDED.login,updated_at=now()`,
		venue, proto, credNS, loginArg, meta); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "名册 upsert: "+err.Error())
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_scope)
		VALUES($1,$2,$3,$4,$5)`, brokerCredActor, brokerCredEvent, brokerCredTicket, note, venue); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "审计行写: "+err.Error())
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "tx commit: "+err.Error())
	}
	// verify_note=verify 摘要（eq/totalEq 面值——凭据零回显）。
	return &dashboardv1.SubmitBrokerCredResponse{Venue: venue,
		FieldsWritten: int32(len(fields)), Verified: true, VerifyNote: verifyNote}, nil
}

// okxVerifyNote OKX verify 生产实现（装配层注入 CexVenue——BalanceEquity
// 只读端实测，totalEq 面值出摘要；凭据零回显）。
func okxVerifyNote(ctx context.Context, creds signutil.Credentials) (string, error) {
	m, err := okxexec.BalanceEquity(ctx, creds, false)
	if err != nil {
		return "", err
	}
	var total float64
	for _, v := range m {
		total += v
	}
	return fmt.Sprintf("balance ok totalEq=%.2f", total), nil
}
