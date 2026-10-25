// Package risk —— 风控单入口（全系统唯一交易前闸 G6）。
// Check(Intent)=唯一 pre-trade 闸：fail-closed、每判留痕 fin_risk_decision、
// deny 按 reason 闭集计数、allow 产单次时效放行令牌。红线：risk 不 import
// invest/research/execution 域包——判据只读 config+DB 事实行。
package risk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
)

// Intent 执行意图（完整性校验缺字段=deny）。Kind 空值=open（兼容既有 caller）；
// close=平仓径——kill_switch/各上限不拦（敞口只减不增，拦平仓=存风险），留痕照落。
type Intent struct {
	StrategyRef int64    `json:"strategy_ref"`
	Venue       string   `json:"venue"`
	InstID      string   `json:"inst_id"`
	Legs        []string `json:"legs"`
	NotionalUSD float64  `json:"notional_usd"`
	Leverage    float64  `json:"leverage"`
	Kind        string   `json:"kind"`
}

// kindOrOpen kind 归一（空=open）——留痕写面统一口径。
func (in Intent) kindOrOpen() string {
	if in.Kind == "" {
		return "open"
	}
	return in.Kind
}

// Decision 判定出件（allow 附令牌——执行件只认令牌）。
type Decision struct {
	Allow      bool
	Reason     string
	DecisionID int64
	TokenID    string
	Expires    time.Time
}

// Gate 风控闸（判据全 risk.* 热启读，缺任一键=deny missing_param）。
type Gate struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	rej  *rejects.Door
	// equity venue 净权益现读件（T-0048：快照行出局——决策时点 balances/AccountSummary
	// 实读注入；nil=concentration 判件缺席 fail-closed 拒开）。
	equity func(ctx context.Context, venue string) (float64, error)
}

// WithEquity 挂 venue 权益现读件（cmd 装配层接线——risk 域不识 venue 方言）。
func (g *Gate) WithEquity(fn func(ctx context.Context, venue string) (float64, error)) *Gate {
	g.equity = fn
	return g
}

func NewGate(pool *pgxpool.Pool, cfg *config.Client) *Gate {
	return &Gate{pool: pool, cfg: cfg,
		rej: rejects.New("risk", "kill_switch", "missing_param", "intent_incomplete", "equity_missing",
			"notional_cap", "venue_concentration", "exposure_cap",
			"token_missing", "token_expired", "token_used", "write_err")}
}

func (g *Gate) getF(k string) (float64, error) { return g.cfg.GetFloat(k) }
func (g *Gate) getB(k string) (bool, error)    { return g.cfg.GetBool(k) }

// Check 唯一交易前闸。判序：kill switch（最高优先，close 径不适用）→intent 完整性
// →ttl 读备（close 径唯一判据键）→open 径：判据读备（缺键=deny）→名义上限
// →venue 集中度→总敞口→allow+令牌。in 本体不改动——intent blob 序列化须与
// 执行件 ConsumeToken 侧逐字节一致（令牌 intent_hash 契约），kind 只进留痕列。
func (g *Gate) Check(ctx context.Context, in Intent) Decision {
	kind := in.kindOrOpen()
	// kill switch（close 径跳过——敞口只减不增，拦平仓=存风险；留痕 deny 照落 open 径）。
	if kind != "close" {
		kill, err := g.getB("risk.kill_switch")
		if err == nil && kill {
			return g.deny(ctx, in, "kill_switch")
		}
	}
	if in.StrategyRef == 0 || in.Venue == "" || in.InstID == "" || len(in.Legs) == 0 || in.NotionalUSD <= 0 {
		return g.deny(ctx, in, "intent_incomplete")
	}
	ttl, errTTL := g.getF("risk.token_ttl_seconds")
	if errTTL != nil {
		return g.deny(ctx, in, "missing_param")
	}
	if kind != "close" {
		maxN, err1 := g.getF("risk.max_notional_usd")
		maxPct, err2 := g.getF("risk.max_venue_pct")
		maxExp, err3 := g.getF("risk.max_exposure_usd")
		if err1 != nil || err2 != nil || err3 != nil {
			return g.deny(ctx, in, "missing_param")
		}
		if in.NotionalUSD > maxN {
			return g.deny(ctx, in, "notional_cap")
		}
		// venue 集中度：该 venue 现有 open 敞口+本意图 > equity*max_pct。
		// 权益=venue 现读（快照出局——读败=fail-closed 拒开，禁陈拍裁决）。
		var eq, vExp, tExp float64
		if g.equity == nil {
			return g.deny(ctx, in, "missing_param")
		}
		eqV, err := g.equity(ctx, in.Venue)
		if err != nil {
			return g.deny(ctx, in, "equity_missing")
		}
		eq = eqV
		// 敞口读面=非退役策略名义额事实行（paper/probe/live/degraded 占额）。
		_ = g.pool.QueryRow(ctx,
			"SELECT COALESCE(sum(notional_usd),0) FROM fin_strategy WHERE venue=$1 AND state<>'retired'", in.Venue).Scan(&vExp)
		_ = g.pool.QueryRow(ctx,
			"SELECT COALESCE(sum(notional_usd),0) FROM fin_strategy WHERE state<>'retired'").Scan(&tExp)
		if eq > 0 && (vExp+in.NotionalUSD) > eq*maxPct {
			return g.deny(ctx, in, "venue_concentration")
		}
		if (tExp + in.NotionalUSD) > maxExp {
			return g.deny(ctx, in, "exposure_cap")
		}
	}
	return g.allow(ctx, in, int64(ttl))
}

// deny 留痕+计数+返回（fail-closed 统一径）。
func (g *Gate) deny(ctx context.Context, in Intent, reason string) Decision {
	g.rej.Inc(reason)
	blob, _ := json.Marshal(in)
	var id int64
	if err := g.pool.QueryRow(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
		VALUES($1,'deny',$2,$3,$4) RETURNING id`, blob, reason, g.cfg.Generation(), in.kindOrOpen()).Scan(&id); err != nil {
		g.rej.Inc("write_err")
	}
	return Decision{Reason: reason, DecisionID: id}
}

// allow 留痕+产放行令牌（单次+时效——机械防绕过）。
func (g *Gate) allow(ctx context.Context, in Intent, ttl int64) Decision {
	blob, _ := json.Marshal(in)
	var id int64
	var exp time.Time
	if err := g.pool.QueryRow(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind,expires_at)
		VALUES($1,'allow','allow',$2,$3,now()+($4||' seconds')::interval) RETURNING id,expires_at`,
		blob, g.cfg.Generation(), in.kindOrOpen(), strconv.FormatInt(ttl, 10)).Scan(&id, &exp); err != nil {
		g.rej.Inc("write_err")
		return Decision{Reason: "write_err"}
	}
	sum := sha256.Sum256(blob)
	var tokID string
	if err := g.pool.QueryRow(ctx, `INSERT INTO fin_risk_token(decision_id,intent_hash,expires_at)
		VALUES($1,$2,$3) RETURNING token_id`, id, hex.EncodeToString(sum[:]), exp).Scan(&tokID); err != nil {
		g.rej.Inc("write_err")
		return Decision{Reason: "write_err"}
	}
	return Decision{Allow: true, Reason: "allow", DecisionID: id, TokenID: tokID, Expires: exp}
}

// ConsumeToken G7 执行件消费闸：令牌存在+未过期+未使用+intent 匹配→置 used_at 返 ok。
// 无令牌/过期/已用=deny（机械防绕过——非约定防）。
func (g *Gate) ConsumeToken(ctx context.Context, tokenID string, in Intent) Decision {
	blob, _ := json.Marshal(in)
	sum := sha256.Sum256(blob)
	var used *time.Time
	var exp time.Time
	var hash string
	err := g.pool.QueryRow(ctx,
		"SELECT used_at,expires_at,intent_hash FROM fin_risk_token WHERE token_id=$1", tokenID).Scan(&used, &exp, &hash)
	if err != nil {
		return g.deny(ctx, in, "token_missing")
	}
	if used != nil {
		return g.deny(ctx, in, "token_used")
	}
	if time.Now().After(exp) {
		return g.deny(ctx, in, "token_expired")
	}
	if hash != hex.EncodeToString(sum[:]) {
		return g.deny(ctx, in, "token_missing")
	}
	if _, err := g.pool.Exec(ctx,
		"UPDATE fin_risk_token SET used_at=now() WHERE token_id=$1 AND used_at IS NULL", tokenID); err != nil {
		return g.deny(ctx, in, "token_missing")
	}
	return Decision{Allow: true, Reason: "allow", TokenID: tokenID}
}

// Rejects 观测面。
func (g *Gate) Rejects() map[string]int64 { return g.rej.Snapshot() }
