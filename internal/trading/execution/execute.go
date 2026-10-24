// Package execution —— 双腿原子执行（G7）。
// 铁律：令牌验证为第一件事（fin_risk_decision 直读——execution 不 import risk）；
// live_switch 非 true=拒执 fail-closed；leg2 败/歧义=LEGGED→自动回滚 leg1；
// 回滚败=rollback_failed 升级留痕不沉默；venue 调用全走注入 Executor（产线
// 接 okxexec——T-0008 已验径，禁另起）。
package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/notify"
	"phase/internal/platform/rejects"
)

// Leg 腿规格（execution 自有型——不引 intent 他域型）。
type Leg struct {
	Venue    string
	InstID   string
	Side     string // buy|sell
	Sz       string // 数量（venue 方言——okx 张数口径由 venue 层换算）
	Notional float64
	Close    bool  // close 语义位（rollback/平仓腿=true——executor 按位分派平仓径，禁反向新开）
	Ticket   int64 // MT5 平仓凭据（leg1 回报 ticket——hedge 户 OrderClose 必携）
}

// OrderResult venue 回报（filled/live/absent/ambiguous 归一为 leg 状态机值）。
type OrderResult struct {
	ClOrdID    string
	State      string // filled|live|partially_filled|absent|unknown
	FillPx     float64
	FillSz     float64
	FillTicket int64 // venue 仓凭据（MT5 ticket——hedge 户平仓 OrderClose 必携；okx=0 略）
}

// Executor venue 下单面（测试注入 fake——产线=okxexec 包装）。
type Executor interface {
	Place(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error)
	// Resolve 歧态势查单（okx clOrdId 定态三分支同语义）。
	Resolve(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error)
}

// EarnMover 开仓前 Earn 停泊赎回面（D-0006 停泊态：闲资默认停 Earn——开仓先赎回
// notional+buffer 回拢交易户，败=fail-closed 不开仓。注入式——execution 不识
// earn/venue 方言，产线=earn.Worker）。
type EarnMover interface {
	RedeemToTrading(ctx context.Context, usdAmt float64) error
}

// Intent 执行意图（decision_id 为闸令牌引）。Kind=open|close——
// live_switch 是开仓闸非退仓闸：close 免检（拦平仓=存风险，与 kill_switch
// 不拦 close 同款逻辑——委员会关闸时有仓必须退得掉）。
type Intent struct {
	Kind        string // open（缺省）|close
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
	earn EarnMover
	rej  *rejects.Door
	al   *notify.Alerter
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client, exec Executor) *Runner {
	return &Runner{pool: pool, cfg: cfg, exec: exec,
		rej: rejects.New("execution", "token_missing", "token_expired", "token_used",
			"token_not_allow", "live_switch_off", "missing_param",
			"hedge_band", "rollback_failed", "earn_redeem_failed", "write_err")}
}

// WithAlerter 挂告警器（nil=静默降级）。
func (r *Runner) WithAlerter(a *notify.Alerter) *Runner { r.al = a; return r }

// WithEarn 挂接停泊赎回面（cmd 装配层接线；未挂=停泊关闭——资金常驻交易户径）。
func (r *Runner) WithEarn(m EarnMover) *Runner { r.earn = m; return r }

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

func (r *Runner) getB(k string) (bool, error) { return r.cfg.GetBool(k) }

func (r *Runner) getF(k string) (float64, error) { return r.cfg.GetFloat(k) }
