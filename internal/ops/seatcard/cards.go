package seatcard

// 卡消费者——meta internal/feishu/card.go 三件套模式级复用（抄模式不依赖
// meta 包）：①内容哈希闸（渲染→sha256→变才 PATCH——天然合并刷帧）；②懒建
// （首见活性席→create 发卡→message_id 落 ops_seat_card 行；零活性席不补）；
// ③启动对账（起环扫活跃席无卡补建，一次性幂等）。
// 出站双缝 Create/Patch 注入（cmd 缺省接 notify.FeishuClient 卡两方法；
// 桩测注伪缝）。暗发闸 Live()=false 时全程只渲不发（dry-run——渲染 JSON
// 落日志验渲染正确一周期，CTO 裁定⑥灰度序）。
// 卡败不吞账不炸环：出站败→ops_event seatcard_err 行（ErrGap 节流——
// 链账 append-only 防刷）+循环继续（订阅环不死）。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"
)

// Config 卡消费者装配（缺省值 NewCards 补齐；Create/Patch/Live 为必配缝——
// live 语义装配定，本包不猜缺省出站）。
type Config struct {
	Seats []SeatDef
	// Live 暗发闸（exec.seatcard_live 按次读——false=dry-run 只渲不发）。
	Live func() bool
	// Create 发卡→message_id；Patch 原地刷新卡（interactive 卡 REST 面）。
	Create func(ctx context.Context, chatID, content string) (string, error)
	Patch  func(ctx context.Context, messageID, content string) error
	// Probe 班态探针（缺省 SystemdProbe——桩测注入伪缝）。
	Probe ProbeUnits
	// ChatID 发卡落点群（feishu-notify.sh 同径同 chat——FEISHU_MANAGE_CHAT_ID）。
	ChatID string
	// MinPatchGap 每席 PATCH 下限（CTO 裁定③去抖闸之二——秒级事件塌帧）。
	MinPatchGap time.Duration
	// ErrGap seatcard_err 落账节流（链账 append-only 防刷——同席同因窗口内零重写）。
	ErrGap time.Duration
	// Now 时钟缝（桩测定针；缺省 wall）。
	Now  func() time.Time
	Logf func(format string, args ...any)
}

// Cards 卡渲染消费者（Refresh 由 dm 帧与兜底拍驱动——同径单线程；mutex
// 护 Reconcile/Refresh 并发面）。
type Cards struct {
	cfg Config
	st  Store

	mu        sync.Mutex
	lastHash  map[string]string    // 席→上次成功落卡内容 sha256 hex（含 DB last_hash 启动种子）
	shownHash map[string]string    // 席→上次 dry-run 已渲内容哈希（渲染册——同内容零重复日志）
	lastPatch map[string]time.Time // 席→上次成功 PATCH 时刻（MinPatchGap 闸）
	lastErr   map[string]time.Time // 席→上次 seatcard_err 落账时刻（ErrGap 闸）
}

// NewCards 装配（不起环——Refresh/Reconcile 由调用方驱动）。
func NewCards(cfg Config, st Store) *Cards {
	if cfg.Probe == nil {
		cfg.Probe = SystemdProbe
	}
	if cfg.MinPatchGap <= 0 {
		cfg.MinPatchGap = 5 * time.Second
	}
	if cfg.ErrGap <= 0 {
		cfg.ErrGap = 5 * time.Minute
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	return &Cards{
		cfg: cfg, st: st,
		lastHash:  map[string]string{},
		shownHash: map[string]string{},
		lastPatch: map[string]time.Time{},
		lastErr:   map[string]time.Time{},
	}
}

// SeedHash 启动种子——DB last_hash 入内存册（重启不重发无变卡）。
// 读败=零种子降级（重启后首拍至多补一 PATCH，meta 同语义）。
func (c *Cards) SeedHash(ctx context.Context) {
	for _, def := range c.cfg.Seats {
		_, _, hash, ok, err := c.st.CardRow(ctx, def.Card)
		if err != nil {
			c.cfg.Logf("卡册种子读败 seat=%s: %v", def.Card, err)
			continue
		}
		if ok && hash != "" {
			c.mu.Lock()
			c.lastHash[def.Card] = hash
			c.mu.Unlock()
		}
	}
}

// Refresh 全席重渲一拍（dm 帧触发与兜底拍同径——装配→渲染→哈希→闸门）。
// 任一席装配败不吞账（seatcard_err 节流落账）且不阻其余席（订阅环不死）。
// 返回值=本拍渲染的卡面 map（dry-run 验证面/桩测断言面；live 拍同返）。
func (c *Cards) Refresh(ctx context.Context) map[string]string {
	now := c.cfg.Now()
	states, err := Assemble(ctx, c.st, c.cfg.Probe, c.cfg.Seats, now)
	if err != nil {
		c.recordErr(ctx, "全席", fmt.Sprintf("装配败: %v", err))
		return nil
	}
	out := map[string]string{}
	for _, cs := range states {
		if !ActiveSeat(cs, now) {
			continue // 零活性席——懒建闸同款，卡面永寂
		}
		content := Render(cs)
		out[cs.Seat] = content
		c.refreshSeat(ctx, cs, content, now)
	}
	return out
}

// refreshSeat 单席闸门链（哈希闸→dry-run 分叉→懒建/PATCH）。
func (c *Cards) refreshSeat(ctx context.Context, cs CardState, content string, now time.Time) {
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])

	c.mu.Lock()
	same := c.lastHash[cs.Seat] == hash
	c.mu.Unlock()
	if same {
		return // 内容未变——哈希闸挡出站（同帧零 PATCH）
	}

	if !c.cfg.Live() {
		// dry-run（exec.seatcard_live 缺省 false）——只渲不发：渲染 JSON
		// 落日志验渲染正确一周期（CTO 裁定⑥灰度序第一步）。渲染册哈希
		// 闸同款——同内容零重复日志，内容变（含分钟窗翻转）才再渲。
		c.mu.Lock()
		shown := c.shownHash[cs.Seat] == hash
		c.shownHash[cs.Seat] = hash
		c.mu.Unlock()
		if !shown {
			c.cfg.Logf("[dry-run] seat=%s status=%s card=%s", cs.Seat, cs.Status, content)
		}
		return
	}

	mid, chat, _, ok, err := c.st.CardRow(ctx, cs.Seat)
	if err != nil {
		c.recordErr(ctx, cs.Seat, fmt.Sprintf("卡册读败: %v", err))
		return
	}
	if !ok {
		// 懒建：活性席无卡→create 落 message_id（零活性席上行已挡）。
		newID, err := c.cfg.Create(ctx, c.pickChat(chat), content)
		if err != nil {
			c.recordErr(ctx, cs.Seat, fmt.Sprintf("卡创建败: %v", err))
			return // 不落行不落哈希——下拍重试（meta 同语义）
		}
		if err := c.st.SaveCard(ctx, cs.Seat, newID, c.pickChat(chat), hash); err != nil {
			c.recordErr(ctx, cs.Seat, fmt.Sprintf("卡行落库败 mid=%s: %v", newID, err))
			return
		}
		c.mu.Lock()
		c.lastHash[cs.Seat] = hash
		c.lastPatch[cs.Seat] = now
		c.mu.Unlock()
		c.cfg.Logf("卡创建 seat=%s mid=%s", cs.Seat, newID)
		return
	}

	// PATCH 下限闸（MinPatchGap）——窗内不 PATCH 不落哈希，下拍重渲
	// 最新内容再判（秒级事件自然塌帧至窗后一拍）。
	c.mu.Lock()
	lastPatch := c.lastPatch[cs.Seat]
	c.mu.Unlock()
	if !lastPatch.IsZero() && now.Sub(lastPatch) < c.cfg.MinPatchGap {
		return
	}
	if err := c.cfg.Patch(ctx, mid, content); err != nil {
		c.recordErr(ctx, cs.Seat, fmt.Sprintf("卡 PATCH 败 mid=%s: %v", mid, err))
		return // PATCH 败不落哈希——下拍重试
	}
	c.mu.Lock()
	c.lastHash[cs.Seat] = hash
	c.lastPatch[cs.Seat] = now
	c.mu.Unlock()
	if err := c.st.SaveHash(ctx, cs.Seat, hash); err != nil {
		c.cfg.Logf("卡哈希落库败 seat=%s: %v", cs.Seat, err) // 内存册已更——展示不断，痕面 WARN
	}
}

// pickChat 落点群——卡册已有 chat_id 优先（原地刷同卡），缺省装配值
// （FEISHU_MANAGE_CHAT_ID 同径同 chat，不新引凭据）。
func (c *Cards) pickChat(stored string) string {
	if stored != "" {
		return stored
	}
	return c.cfg.ChatID
}

// Reconcile 启动对账（一次性幂等）：活跃席无卡补建——live 时走 Create
// 径（懒建），dry-run 时日志验渲染。对账同 Refresh 重渲径——有卡席按
// 哈希闸零出站（重启种子后无变卡零 PATCH）。
func (c *Cards) Reconcile(ctx context.Context) {
	c.Refresh(ctx) // 同径——懒建/哈希闸/dry-run 分叉全在 refreshSeat
}

// recordErr 卡败落账（ErrGap 节流——同席窗口内零重写；落账败只打日志
// 不递归不炸环）。
func (c *Cards) recordErr(ctx context.Context, seat, note string) {
	c.cfg.Logf("卡败 seat=%s: %s", seat, note)
	c.mu.Lock()
	skip := time.Since(c.lastErr[seat]) < c.cfg.ErrGap
	if !skip {
		c.lastErr[seat] = time.Now()
	}
	c.mu.Unlock()
	if skip {
		return
	}
	if err := c.st.RecordErr(ctx, seat, note); err != nil {
		c.cfg.Logf("seatcard_err 落账败 seat=%s: %v", seat, err)
	}
}
