// stuckesc_sig.go —— stuck_esc 探针 incident/condition 分册件
// （工单#18002,supply_esc 棘轮满线分拆——同 audit_esc 先例）。
package main

import (
	"regexp"
	"strings"
)

// stuckIncidentScopes incident 类 scope→签名抽件册——该类每火=离散新
// 事件（新 sha/新会话/新败迹），同 scope 名复燃≠同一病灶未愈；配对须
// 同签名=同败因∨同病灶证，note 可机械抽的败因键。condition 类（断流/
// 能力债/数据缺等持续条件）不入册维持同 scope 配对；deploy_push_blocked/
// deploy_diverged 败因面单一同 scope 即同签名，无签名键不入册。
// 签名抽不出=证据不足不配对（fail-closed——incident 类无签名宁可漏报
// 不误报）。
var stuckIncidentScopes = map[string]func(string) string{
	"cognition_spawn_fail": stuckSigReg(`reason=[^ )，]+`),
	"audit_spawn_fail":     stuckSigReg(`reason=[^ )，]+`),
	"deploy_blocked":       deployBlockedSig,
}

func stuckSigReg(pat string) func(string) string {
	re := regexp.MustCompile(pat)
	return re.FindString
}

var (
	deployRcRe  = regexp.MustCompile(`rc=([0-9]+)`)
	deployShaRe = regexp.MustCompile(`[0-9a-f]{7,40}|ts=\S+|\d{4}-\d{2}-\d{2}[T ][0-9:]+`)
	deployTokRe = regexp.MustCompile(`[\p{L}\p{N}_]+`)
)

// deployBlockedSig 部署败签名=rc+尾面词干谱——尾文本剥 sha/ts 噪后
// 取前两词，同败因类（脏树/锁/网/OCI败）跨 sha 跨拍同签名、异败因
// 异签名；rc 或尾面缺席=签不出。
func deployBlockedSig(note string) string {
	rc := deployRcRe.FindStringSubmatch(note)
	i := strings.Index(note, "尾=")
	if rc == nil || i < 0 {
		return ""
	}
	toks := deployTokRe.FindAllString(deployShaRe.ReplaceAllString(note[i+len("尾="):], " "), 2)
	if len(toks) == 0 {
		return ""
	}
	return rc[1] + "|" + strings.Join(toks, "|")
}
