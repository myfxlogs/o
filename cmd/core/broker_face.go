// broker_face.go —— T-0407/T-0430 座舱 broker 接入读面（在册 broker 列表
// 五源并集——凭据族枚举+MT5 login 单件解密出面+台账态+末拍注记+提交审计
// +策略数；接入态 phase=三源派生服务端单源——末拍败径>管道态>提交审计行，
// cex 落库即 verified 构造保证，前端纯渲染零判据）。凭据族=readcred 钦定
// app_secrets <broker>.mt5.*|<venue>.cex.* 键形+单例 mt5.*+存量 okx.* 无中
// 缀三键（首户遗留并入 cex 族）+存量 .mt4.（链未接如实显形）；login=MT5
// 凭据唯一出面字段（cex 凭据零出面）。写面/verify-then-write 见
// broker_submit.go。
package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"o/internal/gen/dashboardv1"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
)

// ListBrokerOnboard 在册 broker 列表读面（只读——凭据族枚举+MT5 login 单件
// 解密出面+台账态+末拍注记+提交审计+策略数五源并集；phase 服务端派生——
// 前端纯渲染零判据）。
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
	// ① 凭据族（app_secrets <broker>.<proto>.* 键形+单例 mt5.*+存量 okx.*
	// 无中缀三键首户遗留；MT5 login 件解密出面——password/密文余件不解）。
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
		if !ok {
			continue
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
	// ④ 提交审计面（T-0430 verify-then-write：esc_scope=venue 行存在=提交
	// 验连过闸凭据落库——验败提交零行零审计即回，故审计行只证 verified）。
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
	// ⑤ 在跑策略数（fin_strategy 非 retired 行×venue）。
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
	for _, r := range venues {
		r.Phase = brokerPhase(r, submitted)
		resp.Rows = append(resp.Rows, r)
	}
	sort.Slice(resp.Rows, func(i, j int) bool { return resp.Rows[i].Venue < resp.Rows[j].Venue })
	return resp, nil
}

// brokerPhase 接入态派生（T-0430 registered→connecting→verified/failed——
// 判态服务端单源）：mt4 遗留=空（链未接如实显形）；末拍败径=failed；
// 管道已过探测拍（probed/mapped/paper）∨提交审计在册=verified；台账
// registered 探测在途=connecting；余=registered（候发现拍）。cex 落库即
// verified（提交门禁构造保证——验败凭据零落表）。
func brokerPhase(r *dashboardv1.BrokerOnboardRow, submitted map[string]bool) string {
	if r.Proto == "" || r.Proto == "mt4" {
		return "" // 无凭据族行/mt4 遗留——raw 台账态如实显形（前端 STATE_CN 直译）
	}
	if r.BeatFailed {
		return "failed"
	}
	if r.Proto == "cex" {
		return "verified"
	}
	switch r.State {
	case "probed", "mapped", "paper":
		return "verified"
	}
	if submitted[r.Venue] {
		return "verified"
	}
	if r.State == "registered" {
		return "connecting"
	}
	return "registered"
}

// brokerCredKeySplit 凭据键→(venue,proto,field)：单例 mt5.<field>→('mt5',
// 'mt5',field)；<broker>.mt5.<field>→(broker,'mt5',field)；<venue>.cex.
// <field>→(venue,'cex',field)；<broker>.mt4.<field>→(broker,'mt4',field)
// （存量读面兼容）；存量 okx.api_key|api_secret|passphrase（无中缀首户
// 遗留）→('okx','cex',field)。venue token 形闸同 credscan（小写数线横——
// 冒号/空白禁入台账键域）。
func brokerCredKeySplit(k string) (string, string, string, bool) {
	if strings.HasPrefix(k, "mt5.") {
		return "mt5", "mt5", k[4:], true
	}
	for _, f := range []string{"api_key", "api_secret", "passphrase"} {
		if k == "okx."+f {
			return "okx", "cex", f, true
		}
	}
	for _, ns := range []struct{ sep, proto string }{
		{".mt5.", "mt5"}, {".cex.", "cex"}, {".mt4.", "mt4"},
	} {
		if v, f, ok := strings.Cut(k, ns.sep); ok {
			if v == "" || f == "" {
				return "", "", "", false
			}
			for _, c := range v {
				if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
					return "", "", "", false
				}
			}
			return v, ns.proto, f, true
		}
	}
	return "", "", "", false
}
