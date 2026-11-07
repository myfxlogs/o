// dmwrite.go —— DM 写域闸+自治写径（T-0376 业主令：「如何去赚」全参数域归
// 机器自主；业主面=凭证/资金上限/停机总闸/对账/通知通道 fail-closed 不可达——
// 锚④静态断言+测试钉）。分册正本=scripts/config-write-domain.tsv，本件
// dmWritableGlobs/ownerKeyGlobs 同集镜像（漂移=check_config_write_domain.sh
// 闸红）。写径=单事务三拍：域闸→app_config upsert（generation bump/
// config_changed notify 触发器原生——热启链零改）→ops_event `参数调整` 行
// （参数/旧值/新值/证据锚必携——自主≠无账，周评审 T-0360 对账复核）。
package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"

	"github.com/jackc/pgx/v5"
)

// 写域二分（T-0376 分册两类——机读口径与 TSV 类列同字面）。
const (
	DomainDM    = "dm"    // DM/系统全权（每次自调必落 ops_event 参数调整行）
	DomainOwner = "owner" // 业主面——DM 写径不可达（锚④）
)

// dmActor DM 写径 app_config updated_by 身份（审计列留痕——T-0178 律）。
const dmActor = "决策席-DM写径"

// dmWritableGlobs —— dm 类通配（分册 dm 行同集镜像；顺序无关——凭证形
// regex 前置硬拒、owner 行次之，本集末查）。
var dmWritableGlobs = []string{
	"research.*",                  // 评估阈/族参数/扫描族启停
	"guard.*",                     // 守仓参数（flip/breakeven/max_hold——执行域）
	"invest.lifecycle.*",          // 放行线+paper→probe→live 判据值
	"invest.paper.*",              // paper 仓位判定
	"scout.*",                     // venue 测榜参数
	"lifecycle.*",                 // 生命周期 TTL/冷却族
	"strategy.*",                  // 降级 TTL 族
	"gate.kill.*",                 // 族级风险停机闸（退役闸族——T-0312 白名单续认）
	"exec.*",                      // 执行参数（探针仓位/超时/冷却/cfd_map）
	"funding.scan_timeout_s",      // funding 扫描超时
	"gapscan.scan_timeout_s",      // gap 扫描超时
	"dm.hunger.*",                 // 饥饿响应自参数（检测窗）
	"earn.auto_purchase_enabled",  // 闲置金 Earn 自动转入（资金运用重排）
	"earn.sweep_whitelist",        // Earn 扫袋白名单（T-0439 计价币放行线）
	"earn.residual_clear_min_usd", // 杂币实存即清粉尘阈（T-0439 闸值族）
	"trx.floor_pct",               // Tron 质押保底率（Earn 族）
	"trx.vote_enabled",            // Tron 投票开关（Earn 族）
	"trx.vote_max_rebal_per_day",  // Tron 再平衡频次（Earn 族）
	"trx.poll_seconds",            // Tron 轮询间隔（Earn 族）
	"okxws.tick_scan_s",           // 粗扫节拍秒（T-0401 两级采集制——60s 钦定值覆写键）
}

// ownerKeyGlobs —— 业主面键（分册 owner 行同集镜像；与 dm 集不交——交叠=闸红）。
var ownerKeyGlobs = []string{
	"risk.max_notional_usd", // 资金上限（锚④硬界）
	"risk.kill_switch",      // 舰队停机总闸（guard 自动翻位径不受限）
	"ledger.recon_tol_usd",  // 对账容差（红线三）
	"feishu.chat_id",        // 业主通知通道
	"notify.*",              // 通知投递面
	"trx.stake_address",     // 资金去向地址
	"trx.rental_payers",     // 资金对手方注册表
	"bridge.*",              // 采集基建面（fail-closed）
	"calendar.*",            // 宏观日历面（fail-closed）
	"cmc.ws_disabled",       // 采集基建面（fail-closed）
	"net.budget_kbps",       // 带宽预算（基建面）
	"mt5.gateway_addr",      // 网关拓扑（基建面）
	"trx.grpc_endpoint",     // Tron RPC 拓扑（基建面）
	"okx.symbols",           // 采集宇宙清单（GetStringList 数组键——基建面）
	"mt5.symbols",           // 采集宇宙清单（GetStringList 数组键——基建面）
	"readcred.entitlement",  // 读凭据权限白名单（访问控制面——fail-closed）
}

// credentialRE 凭证形键硬拒（先于一切通配——任何键形含凭证词即 owner；
// 与 check-env-reads 凭据域口径同族：trx.trongrid_api_key/cmc.account_id
// 实证在册键）。
var credentialRE = regexp.MustCompile(
	`(?i)(api_?key|secret|token|passphrase|password|account_id|private_key)`)

// ClassifyKey 键域判定（dm|owner）。判定序=凭证形硬拒→owner 行→dm 行→
// 未登记 fail-closed 归 owner（新键须入册，缺册=DM 不可写）。
func ClassifyKey(key string) string {
	if credentialRE.MatchString(key) {
		return DomainOwner
	}
	for _, g := range ownerKeyGlobs {
		if m, _ := path.Match(g, key); m {
			return DomainOwner
		}
	}
	for _, g := range dmWritableGlobs {
		if m, _ := path.Match(g, key); m {
			return DomainDM
		}
	}
	return DomainOwner
}

// DMWriteResult 写径结果（Old=nil=首写无旧行）。
type DMWriteResult struct {
	Key     string
	Old     json.RawMessage // 缺行=nil
	New     json.RawMessage
	Changed bool
}

// DMSet DM 写径唯一入口（域闸 fail-closed+单事务三拍；直写 SQL 绕闸=违例
// ——分册 owner 类/未登记键一律拒）。evidence=判词证据锚（禁空——无证据
// 不自调）；ticket=归属票号（ops_event 行携，常设环携环票号）。
func (c *Client) DMSet(ctx context.Context, key string, value json.RawMessage,
	evidence, ticket string) (*DMWriteResult, error) {
	if ClassifyKey(key) != DomainDM {
		return nil, fmt.Errorf("%s: 写域外（分册 owner 类/未登记——DM 写径 fail-closed）", key)
	}
	if err := validScalar(value); err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	if evidence == "" {
		return nil, fmt.Errorf("%s: 证据锚缺席——无证据不自调", key)
	}
	if ticket == "" {
		return nil, fmt.Errorf("%s: 归属票号缺席——审计行必携出处", key)
	}
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("dmset begin: %w", err)
	}
	defer tx.Rollback(ctx)
	var old json.RawMessage
	err = tx.QueryRow(ctx,
		`SELECT value FROM app_config WHERE key=$1 FOR UPDATE`, key).Scan(&old)
	hasOld := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("dmset read old: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app_config(key,value,updated_by)
		VALUES($1,$2::jsonb,$3)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value,updated_by=EXCLUDED.updated_by`,
		key, string(value), dmActor); err != nil {
		return nil, fmt.Errorf("dmset write: %w", err)
	}
	oldDisp := "∅"
	if hasOld {
		oldDisp = string(old)
	}
	note := fmt.Sprintf("key=%s 旧=%s 新=%s 证据=%s", key, oldDisp, string(value), evidence)
	if _, err := tx.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,$2,$3,$4)`,
		dmActor, "参数调整", ticket, note); err != nil {
		return nil, fmt.Errorf("dmset audit row: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("dmset commit: %w", err)
	}
	return &DMWriteResult{Key: key, Old: old, New: value,
		Changed: !hasOld || string(old) != string(value)}, nil
}

// validScalar 标量契约守（T-0018：JSON 原生类型——数字/布尔/字符串三形；
// object/array/null/双编码=结构性异类拒——audit_app_config_types 普查口径
// 前移到写径）。
func validScalar(value json.RawMessage) error {
	var v any
	if err := json.Unmarshal(value, &v); err != nil {
		return fmt.Errorf("非法 JSON: %w", err)
	}
	switch v.(type) {
	case string, float64, bool:
		return nil
	default:
		return fmt.Errorf("非标量形（%T——数字/布尔/字符串三形外拒）", v)
	}
}
