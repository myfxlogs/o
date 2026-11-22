// stalltriage.go —— 候选滞留分诊（工单O-33223）：fin_venue_candidate
// state='recommended' 超龄候选的静默死面根修——产出开集（venue 候选来
// 一个分诊一个）消费闭集（credscan 只识 .mt5. 族）之间每来一个新方言
// 凭据必原样再断。每拍扫超龄 recommended，双支分诊：
//
//	a) 凭据缺——app_secrets 无 <venue>.% 前缀键（通用键扫不局限
//	   .mt5. 方言）→fin_condition_gap kind='credential' surface=
//	   'venue-credential' 请求盒（condreq pending 撞刷新非重行=幂等
//	   硬闸——业主面催办不静默死）。
//	b) 凭据在但方言不识——<venue>.% 键在册而 discover() 不产该 venue
//	   族（非 .mt5. 方言，如 hyperliquid.account/agent_key 实证）→
//	   ops_esc() 升件 scope='venue_dialect_gap'（函件层 24h note 前缀
//	   窗去重——接入方言=工程缺口归会话面）。
//	c) 凭据已识（discover 产族——.mt5. 方言在册）→零动作，接入链自走。
package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"o/internal/platform/condreq"
)

// stallTriageAge 滞留阈——recommended 发出后业主供凭据=人延迟（实证
// mexc/kucoin/dydx 卡 2 天零催办、hyperliquid 方言缺口 2 天零升件）；
// 24h 与 ops_esc 24h 前缀去重窗/condreq pending 刷新节奏同量级，阈内
// 不分诊防推荐首日误催。var=桩测覆写锚（同 probeRetryGap 先例）。
var stallTriageAge = 24 * time.Hour

// stallTriage 滞留分诊拍（scan 每拍尾挂——枚举败不上抛不阻探针拍；
// 逐 venue 败 log 留痕不炸全拍）。fams=本拍 discover() 产物——「方言
// 已识」判据复用零二次解密。
func (r *Runner) stallTriage(ctx context.Context, fams map[string]credFamily) error {
	rows, err := r.md.Query(ctx, `SELECT venue FROM fin_venue_candidate
		WHERE state='recommended' AND last_request_at < $1`,
		r.now().Add(-stallTriageAge))
	if err != nil {
		return fmt.Errorf("滞留候选枚举: %w", err)
	}
	var venues []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		venues = append(venues, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, v := range venues {
		if err := r.triageStall(ctx, v, fams); err != nil {
			log.Printf("[onboard] %s 滞留分诊败: %v", v, err)
			r.rej.Inc("write_err")
		}
	}
	return nil
}

// triageStall 单 venue 双支分诊（幂等全在写径层——condreq pending 撞
// 刷新/ops_esc 24h 前缀窗，本函数重跑零副作用）。
func (r *Runner) triageStall(ctx context.Context, venue string, fams map[string]credFamily) error {
	keys, err := r.venueKeys(ctx, venue)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		payload, _ := json.Marshal(map[string]string{
			"ask": fmt.Sprintf("请注册/供凭据 %s——venue_req 达阈候凭据推进接入链", venue)})
		if err := condreq.Request(ctx, r.fin, "credential", venue,
			"venue-credential", string(payload)); err != nil {
			return fmt.Errorf("credential gap 写败: %w", err)
		}
		return nil
	}
	if _, ok := fams[venue]; ok {
		return nil // c) .mt5. 方言已识——接入链自走零动作
	}
	// b) 键在册但 discover() 不产族——方言工程缺口升件（函件层去重）。
	var id *int64
	err = r.fin.QueryRow(ctx, `SELECT ops_esc($1,$2,$3,$4,$5,$6)`,
		seatName, ticketID, "", "venue_dialect_gap", "venue接入方言缺口",
		fmt.Sprintf("venue=%s 凭据键在册但 credscan 不识——接入方言工程缺口 键形=%s",
			venue, keyShape(keys[0]))).Scan(&id)
	if err != nil {
		return fmt.Errorf("方言缺口升件写败: %w", err)
	}
	return nil
}

// venueKeys <venue>.% 前缀键枚举（app_secrets 键名非密文可枚举——凭据
// 缺/方言不识分诊的「键在册」判据；strpos 前缀锚防 'gate' 误吞
// 'gateio.' 邻名）。LIMIT 16——键形摘要只取首键，超出截断无碍判定。
func (r *Runner) venueKeys(ctx context.Context, venue string) ([]string, error) {
	rows, err := r.fin.Query(ctx, `SELECT key FROM app_secrets
		WHERE strpos(key, $1 || '.') = 1 ORDER BY key LIMIT 16`, venue)
	if err != nil {
		return nil, fmt.Errorf("venue 键扫: %w", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// keyShape 首键形摘要（<venue>.<dialect…> 前两段——'hyperliquid.
// account'→'hyperliquid.account'、'ibkr.paper_account'→'ibkr.paper'；
// 键名非密文可入 note，ciphertext 永不落）。
func keyShape(k string) string {
	parts := strings.SplitN(k, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return k
}
