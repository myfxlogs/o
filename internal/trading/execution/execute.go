// Package execution —— 双腿原子执行（G7）。
// 铁律：令牌验证为第一件事（fin_risk_decision 直读——execution 不 import risk）；
// live_switch 非 true=拒执 fail-closed；leg2 败/歧义=LEGGED→自动回滚 leg1；
// 回滚败=rollback_failed 升级留痕不沉默；venue 调用全走注入 Executor（产线
// 接 okxexec——T-0008 已验径，禁另起）。
package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/rejects"
)

// Leg 腿规格（execution 自有型——不引 intent 他域型）。
type Leg struct {
	Venue    string
	InstID   string
	Side     string // buy|sell
	Sz       string // 数量（venue 方言——okx 张数口径由 venue 层换算）
	Notional float64
}

// OrderResult venue 回报（filled/live/absent/ambiguous 归一为 leg 状态机值）。
type OrderResult struct {
	ClOrdID string
	State   string // filled|live|partially_filled|absent|unknown
	FillPx  float64
	FillSz  float64
}

// Executor venue 下单面（测试注入 fake——产线=okxexec 包装）。
type Executor interface {
	Place(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error)
	// Resolve 歧态势查单（okx clOrdId 定态三分支同语义）。
	Resolve(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error)
}

// Intent 执行意图（decision_id 为闸令牌引）。
type Intent struct {
	DecisionID  int64
	StrategyRef int64
	InstID      string
	NotionalUSD float64
	Legs        [2]Leg
}

// Runner 执事岗。
type Runner struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	exec Executor
	rej  *rejects.Door
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client, exec Executor) *Runner {
	return &Runner{pool: pool, cfg: cfg, exec: exec,
		rej: rejects.New("execution", "token_missing", "token_expired", "token_used",
			"token_not_allow", "live_switch_off", "missing_param",
			"hedge_band", "rollback_failed", "write_err")}
}

// verifyToken 令牌验证为第一件事：不存在/非 allow/过期/已消费=拒（四径）。
// 通过则同 tx 置 used_at（单次消费原子性）。
func (r *Runner) verifyToken(ctx context.Context, decisionID int64) (string, error) {
	var tok string
	var used *time.Time
	var exp time.Time
	var dec string
	err := r.pool.QueryRow(ctx, `SELECT t.token_id,t.used_at,t.expires_at,d.decision
		FROM fin_risk_token t JOIN fin_risk_decision d ON d.id=t.decision_id
		WHERE t.decision_id=$1`, decisionID).Scan(&tok, &used, &exp, &dec)
	if err != nil {
		r.rej.Inc("token_missing")
		return "", fmt.Errorf("token_missing")
	}
	if dec != "allow" {
		r.rej.Inc("token_not_allow")
		return "", fmt.Errorf("token_not_allow")
	}
	if used != nil {
		r.rej.Inc("token_used")
		return "", fmt.Errorf("token_used")
	}
	if time.Now().After(exp) {
		r.rej.Inc("token_expired")
		return "", fmt.Errorf("token_expired")
	}
	res, err := r.pool.Exec(ctx,
		"UPDATE fin_risk_token SET used_at=now() WHERE token_id=$1 AND used_at IS NULL", tok)
	if err != nil || res.RowsAffected() == 0 {
		r.rej.Inc("token_used")
		return "", fmt.Errorf("token_used")
	}
	return tok, nil
}

func (r *Runner) getB(k string) (bool, error) {
	raw, err := r.cfg.Get(k)
	if err != nil {
		return false, err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return false, err
	}
	return s == "true", nil
}
func (r *Runner) getF(k string) (float64, error) {
	raw, err := r.cfg.Get(k)
	if err != nil {
		return 0, err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}
