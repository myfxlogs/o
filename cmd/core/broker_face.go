// broker_face.go —— T-0407/T-0430/O-37450 座舱 broker 接入读面。
// O-37450 主源化：fin_venue_roster=账户名册唯一真源（已录账户生命周期
// venue/proto/login/state/cred_ns 直读——login 出面户号列免解密）；
// app_secrets 凭据族枚举降为补面——roster 外孤儿凭据族=形坏显形
// （有凭据无名册=提交面外写入病灶），名册 cred_ns 必备键缺件
// →phase='cred_missing'；fin_venue_candidate=研究面普查册（候选 venue
// 非账户——分工：candidate=未接入研究候选，roster=已录账户生命周期；
// 两册零行迁移不互导，candidate 行仅在 roster/凭据族均无时补位显形）。
// 台账态②段退役——roster.state 即生命周期真源（会话环自报+人工迁移）。
// phase 派生：roster 户=brokerPhaseRoster（cred_missing>生命周期映射），
// 非名册行=brokerPhase 旧五源派生（存量凭据孤儿/候选兼容）。
// 凭据族=readcred 钦定 app_secrets <cred_ns>.<field> 键形+单例 mt5.*+
// 存量 okx.* 无中缀三键（首户遗留并入）+存量 .mt4.（链未接如实显形）；
// cex 凭据零出面。写面/verify-then-write 见 broker_submit.go，
// 态迁移面见 broker_state.go。
package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"o/internal/gen/dashboardv1"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
)

// ListBrokerOnboard 在册 broker 列表读面（roster 主源+凭据族补面+
// 候选补位+末拍注记+提交审计+策略数并集；phase 服务端派生——前端纯
// 渲染零判据）。
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
	// ① roster 主源（O-37450——fin_venue_roster=账户名册唯一真源；
	//    login=出面户号列直读免解密；cred_ns 必备键缺件→credBad 显形；
	//    removed 行凭据已删=缺件属预期不误报）。
	rosterSet := map[string]bool{}
	credBad := map[string]bool{}
	rrows, err := c.pool.Query(ctx, `SELECT venue,proto,coalesce(login,''),state,cred_ns
		FROM fin_venue_roster ORDER BY venue`)
	if err != nil {
		return nil, fmt.Errorf("名册读: %w", err)
	}
	type rosterRow struct{ proto, credNS, state string }
	rosterRows := map[string]rosterRow{}
	for rrows.Next() {
		var v, proto, login, state, credNS string
		if err := rrows.Scan(&v, &proto, &login, &state, &credNS); err != nil {
			rrows.Close()
			return nil, err
		}
		r := rowOf(v)
		r.Proto = proto
		r.Login = login // 名册 login 列=出面户号（cex=NULL→空串——凭据零出面不变）
		r.State = state
		rosterSet[v] = true
		rosterRows[v] = rosterRow{proto: proto, credNS: credNS, state: state}
	}
	rrows.Close()
	if err := rrows.Err(); err != nil {
		return nil, err
	}
	// 名册 cred_ns 必备键核（per-proto 三件——mt5=login/password/server
	// [host 选填]；cex=api_key/api_secret/passphrase；removed=凭据已删
	// 预期缺件不误报；名册小表 N+1 可承）。
	for v, rr := range rosterRows {
		if rr.state == "removed" {
			continue
		}
		var fields []string
		if rr.proto == "cfd" {
			fields = []string{"login", "password", "server"}
		} else {
			fields = []string{"api_key", "api_secret", "passphrase"}
		}
		keys := make([]string, len(fields))
		for i, f := range fields {
			keys[i] = rr.credNS + "." + f
		}
		var have int
		if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM app_secrets WHERE key=ANY($1)`, keys).Scan(&have); err != nil {
			return nil, fmt.Errorf("凭据核 %s: %w", v, err)
		}
		if have < len(fields) {
			credBad[v] = true
		}
	}
	// ② 凭据族补面（roster 外孤儿=形坏显形——已在册户跳过：名册行
	//    cred_ns 键形可能异于旧 `<venue>.<proto>.<field>` 推导，名册户
	//    凭据真源=cred_ns 核已覆；孤儿族走旧 login 解密出面径）。
	rows, err := c.pool.Query(ctx, `SELECT key,ciphertext FROM app_secrets
		WHERE key LIKE 'mt5.%' OR strpos(key,'.mt5.')>0 OR strpos(key,'.mt4.')>0
		OR strpos(key,'.cex.')>0 OR key IN ('okx.api_key','okx.api_secret','okx.passphrase')`)
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
		if !ok || rosterSet[venue] {
			continue // 名册已在册户=名册真源（孤儿族才走此补面）
		}
		r := rowOf(venue)
		r.Proto = proto // 同 venue 跨族=后写盖前（实态一族一户——形坏共存如实盖）
		if proto != "mt5" || field != "login" {
			continue // cex 凭据零出面（api_key 永不解密出）
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
	// ③ fin_venue_candidate 补位（研究面普查册——仅 roster/凭据族均无
	//    的 venue 才入列显形；roster 户不混候选态——两册分工见文件头）。
	crows, err := c.pool.Query(ctx, `SELECT venue,state FROM fin_venue_candidate`)
	if err != nil {
		return nil, fmt.Errorf("候选读: %w", err)
	}
	for crows.Next() {
		var v, st string
		if err := crows.Scan(&v, &st); err != nil {
			crows.Close()
			return nil, err
		}
		if !rosterSet[v] {
			rowOf(v).State = st // 非名册行（凭据孤儿/纯候选）——State=研究面态；roster 户不混候选态（两册分工）
		}
	}
	crows.Close()
	if err := crows.Err(); err != nil {
		return nil, err
	}
	// ④ 末拍注记（ops_event「venue接入」esc_scope=venue:login——venue 段
	//    归键，末行取注记+败径标+时刻）。
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
	// ⑤ 提交审计面（T-0430 verify-then-write：esc_scope=venue 行存在=提交
	//    验连过闸凭据落库——验败提交零行零审计即回，故审计行只证 verified）。
	submitted := map[string]bool{}
	srows, err := c.pool.Query(ctx, `SELECT DISTINCT esc_scope FROM ops_event
		WHERE event='venue凭据提交' AND esc_scope IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("提交审计读: %w", err)
	}
	for srows.Next() {
		var v string
		if err := srows.Scan(&v); err != nil {
			srows.Close()
			return nil, err
		}
		submitted[v] = true
	}
	srows.Close()
	if err := srows.Err(); err != nil {
		return nil, err
	}
	// ⑥ 在跑策略数（fin_strategy 非 retired 行×venue）。
	frows, err := c.pool.Query(ctx, `SELECT venue,count(*) FROM fin_strategy
		WHERE state<>'retired' GROUP BY venue`)
	if err != nil {
		return nil, fmt.Errorf("策略数读: %w", err)
	}
	for frows.Next() {
		var v string
		var n int64
		if err := frows.Scan(&v, &n); err != nil {
			frows.Close()
			return nil, err
		}
		rowOf(v).Strategies = n
	}
	frows.Close()
	if err := frows.Err(); err != nil {
		return nil, err
	}
	resp := &dashboardv1.ListBrokerOnboardResponse{}
	for v, r := range venues {
		if rosterSet[v] {
			r.Phase = brokerPhaseRoster(r.State, credBad[v])
		} else {
			r.Phase = brokerPhase(r, submitted)
		}
		resp.Rows = append(resp.Rows, r)
	}
	sort.Slice(resp.Rows, func(i, j int) bool { return resp.Rows[i].Venue < resp.Rows[j].Venue })
	return resp, nil
}
