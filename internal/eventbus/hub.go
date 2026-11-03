// Package eventbus —— T-0312 第五刀 OCI 事件总线 hub（源注册表+3s 轮
// SELECT max(id)+增→扇出；resume<current→补帧后流式跟帧；帧=信号非数据）。
package eventbus

import (
	"context"

	"github.com/jackc/pgx/v5"
	"sync"
	"time"

	"o/internal/gen/eventv1"
)

// Source 源注册表行（表名→kind+dsn+水印 SQL——列非 id 源如 md_macro_event
// event_date 须自配查询，max(id) 不可通吃）。
type Source struct {
	Table string
	Kind  string
	DSN   string
	Q     string // 水印查询（单值 bigint 返——NULL 表=0）
}

// Querier 池缝（pgxpool.QueryRow 同形——测试桩替；pgx.Row 返型同件）。
type Querier interface {
	QueryRow(ctx context.Context, q string, args ...any) pgx.Row
}

// Hub 帧扇出核（watcher 册+末帧簿——可测核 PG 抽 Querier 缝）。
type Hub struct {
	mu    sync.Mutex
	cur   map[string]*eventv1.DataAdvanced // source→末帧（含 kind——补帧重播全形）
	subs  map[int]chan *eventv1.DataAdvanced
	next  int
	since time.Time
}

func NewHub() *Hub {
	return &Hub{cur: map[string]*eventv1.DataAdvanced{}, subs: map[int]chan *eventv1.DataAdvanced{}, since: time.Now()}
}

// Advance 源推进（增→扇出；水位回退不帧——max_id 只升语义）。
func (h *Hub) Advance(src *eventv1.DataAdvanced) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if cur := h.cur[src.Source]; cur != nil && src.MaxId <= cur.MaxId {
		return
	}
	h.cur[src.Source] = src
	for _, ch := range h.subs {
		select {
		case ch <- src:
		default: // watcher 慢=丢帧不阻（信号非数据——下轮 max 仍追平）
		}
	}
}

// Watch 订阅（resume 补帧：低于 current 的源各重播末帧后跟流——帧带
// kind 全形，消费端唤醒映射同活帧语义）。
func (h *Hub) Watch(resume map[string]int64) (<-chan *eventv1.DataAdvanced, func()) {
	ch := make(chan *eventv1.DataAdvanced, 16)
	h.mu.Lock()
	id := h.next
	h.next++
	h.subs[id] = ch
	cur := map[string]*eventv1.DataAdvanced{}
	for s, m := range h.cur {
		cur[s] = m
	}
	h.mu.Unlock()
	go func() {
		for s, m := range cur {
			if resume[s] < m.MaxId {
				select {
				case ch <- m:
				default:
				}
			}
		}
	}()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

// Poller 3s 轮源注册表（增→hub.Advance；败=下拍续不炸）。
type Poller struct{ srcs map[string]Source }

func NewPoller(srcs map[string]Source) *Poller { return &Poller{srcs: srcs} }

func (p *Poller) Run(ctx context.Context, pools map[string]Querier, h *Hub) {
	tk := time.NewTicker(3 * time.Second) // sched-exempt-ok 钦定 3s 轮（票面钦定总线轮询非事件替）
	defer tk.Stop()
	for {
		for _, s := range p.srcs {
			q, ok := pools[s.DSN]
			if !ok {
				continue
			}
			var max int64
			if err := q.QueryRow(ctx, s.Q).Scan(&max); err != nil {
				continue // 查询败=下拍续
			}
			h.Advance(&eventv1.DataAdvanced{Source: s.Table, Kind: s.Kind, MaxId: max})
		}
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
		}
	}
}
