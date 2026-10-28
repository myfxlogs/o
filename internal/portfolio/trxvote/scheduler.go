// TRX 投票收益优化器调度器。SSOT: docs/D-269设计…Claude.md §5
package trxvote

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"o/internal/platform/safego"
)

// AccountDetail 账户投票详情——读面本地域结构（D-117：与 gotron proto 解耦，
// tronscan REST 主读 / gRPC 兜底同源供给）。
type AccountDetail struct {
	Votes     map[string]int64 // SR 地址 → 已投票数（TRX 单位）
	TronPower int64            // 总 TP=总票数
}

// AccountFetcher 获取账户详情（当前投票+总 TP），便于 mock 测试。
type AccountFetcher interface {
	GetAccountDetail(ctx context.Context, addr string) (*AccountDetail, error)
}

// Scheduler 调度 TRX 投票收益优化器的两个独立 tick。
type Scheduler struct {
	executor             *Executor
	store                Store
	address              string
	tronscanBase         string
	config               RebalanceConfig
	account              AccountFetcher // 获取当前投票+总 TP
	now                  func() time.Time
	lastClaimFailAt      time.Time // 领奖阻塞冷却（失败/带宽不足共用，15min）
	lastRebalanceBlockAt time.Time // 改投阻塞冷却（带宽不足/查询失败共用，15min）
	aboveThreshold       bool      // reward 是否在 >10e6 区间（状态迁移才打日志）
	accountQueryFailing  bool      // D-117：账户查询失败 episode 标记（告警去重+恢复边沿）
	// 事件驱动改投（30min 盲醒 tick 已摘除，见 boundary.go）。
	boundaryCh        chan string
	rekickSteps       []time.Duration                              // T-0192②：账户读败退避重 kick 步谱（60/300/900s 心跳律 sched-exempt-ok）
	rekickAt          int                                          // 当前退避档（读通归零）
	witFallback       func(ctx context.Context) ([]SRInfo, error)  // T-0192③：witness 备读（trongrid listwitnesses）
	nextMaintFn       func(ctx context.Context) (time.Time, error) // GetNextMaintenanceTime 源（failsafe 用）
	nextMaint         time.Time                                    // 缓存下一维护期边界
	nextMaintMu       sync.Mutex
	failsafeEvery     time.Duration                             // failsafe 核醒间隔（默认 60s，纯本地比对）
	confirmSteps      []time.Duration                           // 自发 tx 确认窗节奏（测试可注）
	onEvaluated       func(ctx context.Context)                 // 评估完成回调（apr/推送刷新）
	onSelfTxConfirmed func(ctx context.Context)                 // 自发 tx 确认回调（account/apr 重取）
	priceFn           func(ctx context.Context) (float64, bool) // T-0220：TRX 现价源（领奖 USD 折算；nil=省略折算段）
}

// Store 是 Scheduler 依赖的存储接口（解耦 pgstore.Store 供测试注入）。
type Store interface {
	InsertTrxVoteRebalance(ctx context.Context, rec TRXVoteRebalanceRecord) (int64, error)
	UpdateTrxVoteRebalanceResult(ctx context.Context, id int64, txHash, status string) error
	TrxVoteRebalancedToday(ctx context.Context, maxPerDay int) bool
	LastExecutedRebalanceAt(ctx context.Context) (time.Time, error) // D-293 S3：防抖查询
	LastRebalanceDecidedAt(ctx context.Context) (time.Time, error)  // D-137：boot 补课（最近评估时刻）
}

// TRXVoteRebalanceRecord 是 Store.InsertTrxVoteRebalance 的输入（解耦 pgstore struct 供测试）。
type TRXVoteRebalanceRecord struct {
	CurrentYield   float64
	TargetYield    float64
	ImprovementPct float64
	NeedRebalance  bool
	Reason         string
	TargetVotes    []byte // JSONB
	Status         string
}

// NewScheduler 构造调度器。
func NewScheduler(executor *Executor, store Store, address, tronscanBase string, config RebalanceConfig, account AccountFetcher) *Scheduler {
	return &Scheduler{
		executor:     executor,
		store:        store,
		address:      address,
		tronscanBase: tronscanBase,
		config:       config,
		account:      account,
		now:          time.Now,
		boundaryCh:   make(chan string, 4),
		rekickSteps:  []time.Duration{60 * time.Second, 300 * time.Second, 900 * time.Second}, // sched-exempt-ok 账户读败退避心跳
	}
}

// SetNow 替换 now 函数（测试用）。
func (s *Scheduler) SetNow(fn func() time.Time) { s.now = fn }

// SetPriceFn 注入 TRX 现价源（T-0220 领奖 USD 折算——ok=false/nil=告警省略
// 折算段不阻断推送）。
func (s *Scheduler) SetPriceFn(fn func(ctx context.Context) (float64, bool)) { s.priceFn = fn }

// Run 启动事件驱动改投评估（D-137：reward 差分/自 tx 确认/failsafe/boot 补课四触发源）。
func (s *Scheduler) Run(ctx context.Context) {
	safego.GoLoop("trxvote:rebalance", func() { s.eventLoop(ctx) })
}

// ClaimCheck 领奖条件触发（搭车 60s 采集流）。三态：≤10e6 跳过；冷却<15min 跳过；否则预检→ClaimReward。
func (s *Scheduler) ClaimCheck(ctx context.Context, rewardSun int64) {
	if rewardSun <= 10_000_000 { // 1. 奖励不足 10 TRX → 零 RPC 跳过（状态迁移才打日志）
		if s.aboveThreshold {
			s.aboveThreshold = false
			log.Printf("[trxvote] claim check: reward=%d sun ≤10e6, entered below-threshold (skip)", rewardSun)
		}
		return
	}
	if !s.aboveThreshold {
		s.aboveThreshold = true
		log.Printf("[trxvote] claim check: reward=%d sun >10e6, entered above-threshold", rewardSun)
	}
	// 2. 冷却：距上次领奖阻塞 <15min → 跳过（防重复告警轰炸）
	if !s.lastClaimFailAt.IsZero() && s.now().Sub(s.lastClaimFailAt) < 15*time.Minute {
		log.Printf("[trxvote] claim check: cooling down (last block %v ago), skip", s.now().Sub(s.lastClaimFailAt).Round(time.Second))
		return
	}
	// 3. 带宽预检（领奖 ~250 BP）
	ok, err := s.executor.HasBandwidth(ctx, BandwidthNeedClaimBP)
	if err != nil {
		log.Printf("[trxvote] claim bandwidth check: %v", err)
		s.executor.Alert("trxvote_engine", "[投票领奖带宽查询失败] "+err.Error())
		s.lastClaimFailAt = s.now()
		return
	}
	if !ok {
		log.Println("[trxvote] 带宽不足，跳过领奖（避免烧 TRX）")
		s.executor.Alert("trxvote_engine", "[投票领奖跳过] 带宽不足，等待恢复")
		s.lastClaimFailAt = s.now() // 带宽不足设冷却防告警轰炸
		return
	}
	// 4. 执行领奖
	txHash, amount, err := s.executor.ClaimReward(ctx)
	if err != nil {
		log.Printf("[trxvote] claim failed: %v", err)
		s.executor.Alert("trxvote_engine", "[投票领奖失败] "+err.Error())
		s.lastClaimFailAt = s.now()
		return
	}
	if txHash == "" {
		log.Println("[trxvote] claim: reward dropped below threshold, skip")
		return
	}
	log.Printf("[trxvote] claim success: tx=%s amount=%d sun (%.4f TRX)", txHash, amount, float64(amount)/1e6)
	s.executor.Alert("trxvote_claim", formatClaimAlert(txHash, amount, s.trxPrice(ctx)))
	s.watchSelfTx(ctx, txHash) // D-137③：自发 tx 有界确认窗
}

// trxPrice T-0220：现价读取（priceFn 缺席/失败=0——告警省略 USD 折算段）。
func (s *Scheduler) trxPrice(ctx context.Context) float64 {
	if s.priceFn == nil {
		return 0
	}
	if p, ok := s.priceFn(ctx); ok {
		return p
	}
	return 0
}

// doRebalance 执行一轮改投决策（设计 §5）。
func (s *Scheduler) doRebalance(ctx context.Context) bool {
	// 1. 检查今日是否已改投
	if s.store.TrxVoteRebalancedToday(ctx, s.config.MaxRebalancePerDay) {
		log.Println("[trxvote] 今日已改投，跳过")
		return true
	}
	// 1.1 防抖：距上次 executed < MinInterval → 跳过（不落库）
	if s.config.MinInterval > 0 {
		lastExec, err := s.store.LastExecutedRebalanceAt(ctx)
		if err == nil && !lastExec.IsZero() && s.now().Sub(lastExec) < s.config.MinInterval {
			log.Printf("[trxvote] 距上次改投 %v < MinInterval %v，跳过（防抖）",
				s.now().Sub(lastExec).Round(time.Second), s.config.MinInterval)
			return true
		}
	}
	// 1.2 带宽阻塞冷却：距上次带宽不足/查询失败 <15min → 跳过（防重复告警轰炸）
	if !s.lastRebalanceBlockAt.IsZero() && s.now().Sub(s.lastRebalanceBlockAt) < 15*time.Minute {
		log.Printf("[trxvote] rebalance bandwidth: cooling down (last block %v ago), skip", s.now().Sub(s.lastRebalanceBlockAt).Round(time.Second))
		return true
	}
	// 2. 拉 SR 列表（独立 HTTP fetch TronScan /api/witness）
	srList, err := s.fetchSRListFallback(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("[trxvote] fetch SR list canceled (container shutdown)")
			return false
		}
		log.Printf("[trxvote] fetch SR list: %v", err)
		s.executor.Alert("trxvote_engine", "[投票采集失败] "+err.Error())
		return false // T-0192①：读败返 false=边界不消费下拍 failsafe 重试
	}

	// 3. 拉当前投票分布 + 总 TP（D-117：tronscan /api/vote 主读，trongrid gRPC 兜底——
	//    gRPC 读面 17:59Z 起持续性 429 实证；失败告警 episode 去重不再每 tick 刷）。
	accDetail, err := s.account.GetAccountDetail(ctx, s.address)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("[trxvote] get account detail canceled (container shutdown)")
			return false
		}
		log.Printf("[trxvote] get account detail: %v", err)
		if !s.accountQueryFailing { // episode 首报 warn，期内失败只 log
			s.accountQueryFailing = true
			s.executor.Alert("trxvote_engine", "[投票账户查询失败] "+err.Error())
		}
		s.scheduleRekick(ctx) // T-0192②：账户读失败退避重 kick（触发链单点饿死根修）
		return false
	}
	if s.accountQueryFailing { // 恢复边沿各报一条（去重对账完整）
		s.accountQueryFailing = false
		s.executor.Alert("trxvote_engine", "[投票账户查询恢复] tronscan/gRPC 账户读路恢复")
	}
	currentVotes := convertVotes(accDetail.Votes)
	tp := accDetail.TronPower

	// 4. 决策
	decision := Decide(RebalanceInput{
		SRRanking:    srList,
		CurrentVotes: currentVotes,
		TotalTP:      tp,
		Config:       s.config,
	})
	// 5. 落库决策记录（无论是否执行）
	targetJSON, _ := json.Marshal(decision.Target)
	rec := TRXVoteRebalanceRecord{
		CurrentYield:   decision.CurrentYield,
		TargetYield:    decision.TargetYield,
		ImprovementPct: decision.Improvement,
		NeedRebalance:  decision.NeedRebalance,
		Reason:         decision.Reason,
		TargetVotes:    targetJSON,
		Status:         "pending",
	}
	id, err := s.store.InsertTrxVoteRebalance(ctx, rec)
	if err != nil {
		log.Printf("[trxvote] insert rebalance record: %v", err)
		return true // 决策已落定=边界消费（执行败不吞窗）
	}
	// 6. 执行（仅在 needRebalance=true 时）
	if !decision.NeedRebalance {
		log.Printf("[trxvote] 无需改投: %s", decision.Reason)
		return true
	}

	// 6.1 带宽预检：改投需要 ~500 BP，不足则跳过避免烧 TRX
	ok, err := s.executor.HasBandwidth(ctx, BandwidthNeedVoteBP)
	if err != nil {
		log.Printf("[trxvote] bandwidth check: %v", err)
		s.executor.Alert("trxvote_engine", "[投票带宽查询失败] "+err.Error())
		s.lastRebalanceBlockAt = s.now()
		return true
	}
	if !ok {
		log.Println("[trxvote] 带宽不足，跳过改投（避免烧 TRX）")
		s.executor.Alert("trxvote_engine", "[投票改投跳过] 带宽不足，等待恢复")
		s.lastRebalanceBlockAt = s.now() // 设冷却防重复告警
		return true
	}

	txHash, err := s.executor.Vote(ctx, decision.Target)
	if err != nil {
		log.Printf("[trxvote] rebalance failed: %v", err)
		s.executor.Alert("trxvote_engine", "[投票改投失败] "+err.Error())
		return true
	}
	if err := s.store.UpdateTrxVoteRebalanceResult(ctx, id, txHash, "executed"); err != nil {
		log.Printf("[trxvote] update rebalance result: %v", err)
	}
	log.Printf("[trxvote] rebalance success: tx=%s yield %s→%s TRX/yr (+%s%%) SR %s→%s",
		txHash, formatFloat(decision.CurrentYield), formatFloat(decision.TargetYield),
		formatFloat(decision.Improvement), topSRPrefix(currentVotes), topSRPrefix(decision.Target))
	s.executor.Alert("trxvote_rebalance", formatRebalanceAlert(txHash, decision, currentVotes, tp))
	s.watchSelfTx(ctx, txHash) // D-137③：自发 tx 有界确认窗
	return true
}
