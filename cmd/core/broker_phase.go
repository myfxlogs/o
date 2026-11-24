// broker_phase.go —— broker 接入读面 phase 派生三件（O-37450 从
// broker_face.go 拆出——棘轮闸 300 行基线；语义注释随件迁移原样）。
package main

import (
	"strings"

	"o/internal/gen/dashboardv1"
)

// brokerPhaseRoster 名册户 phase 派生（O-37450——roster.state 生命周期
// 机→读面相：cred_missing>removed/disabled 直通>active=verified>
// degraded=failed（连败在册）>connecting/registered=connecting）。
func brokerPhaseRoster(state string, credBad bool) string {
	if credBad {
		return "cred_missing"
	}
	switch state {
	case "active":
		return "verified"
	case "degraded":
		return "failed"
	case "disabled":
		return "disabled"
	case "removed":
		return "removed"
	default: // connecting/registered/未知态——候首次拍如实显形
		return "connecting"
	}
}

// brokerPhase 非名册行 phase 派生（T-0430 registered→connecting→
// verified/failed——判态服务端单源）：mt4 遗留=空（链未接如实显形）；
// 末拍败径=failed；管道已过探测拍（probed/mapped/paper）∨提交审计
// 在册=verified；台账 registered 探测在途=connecting；余=registered
// （候发现拍）。cex 落库即 verified（提交门禁构造保证——验败凭据零落表）。
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
