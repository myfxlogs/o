// Package scanwm —— 族谱水位闸通用形制（T-0440③a）：「新加 CEX 还是这
// 问题」的通用对治——期望族集=注册面派生（调用面注入本 venue 应扫族集，
// 闸零 venue 硬编码；新 venue 注册即自动纳入闸域），实落水位=面自持读缝
// （逐族末落时间——Landed），对差（族零鲜落：零行史或水位龄超阈）持续
// ≥阈→ops_event 升件（谱域缺行，1/族/日 NOT EXISTS 原子去重——重启复拍
// 不重升）。注册面缺席（期望集空）=闸自身 fail-closed 升「不可判」（谱域
// 不可判，1/域/日去重）——不可判也 loud 非静默放行。观测面零侵入：任何
// 败（水位读败/落行败）WARN 不上抛不拖主径；只报不修——修归常票。
// 判/去重/升件沿 selfprobe/venueFailGate 公共模子族形（第三形：按族分域
// 对差判——前两形为单面行龄）。
package scanwm

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// DefaultPersist 对差持续阈（票面钦定 ≥10min——粗扫 60s 拍 10 拍裕度；
// 水位龄即持续时长实证——鲜落族水位每拍前进，龄超阈=持续零鲜落）。
const DefaultPersist = 10 * time.Minute

// 升件类（判类册 precedent_classes.tsv 追记位——谱域缺行/谱域不可判）。
const (
	ClassMissing     = "谱域缺行"
	ClassUndecidable = "谱域不可判"
)

// EscDB 升件写面最小缝（*pgxpool.Pool 同形——桩测可覆写；去重=SQL 侧
// NOT EXISTS 1/域日——桩测捕 Exec 断言载荷）。
type EscDB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Landed 实落水位读缝（族→末落时间；nil=零行史——各面自持查询与池，
// 闸只管判/去重/升件，selfprobe Face 同律）。
type Landed func(ctx context.Context, fam string) (*time.Time, error)

// Gate 族谱水位闸（期望集/venue/seat/ticket 全装配注入——闸体零 venue 词）。
type Gate struct {
	esc     EscDB
	seat    string // ops_event seat（上报组件身份）
	ticket  string // 归属票号
	venue   string // 判例域（esc_scope 前缀——新 venue 自动分域）
	fams    []string
	persist time.Duration
	Landed  Landed
	now     func() time.Time

	mu    sync.Mutex
	never map[string]time.Time // 零行史族→首见缺行拍（冷启动 10min 裕度锚）
}

// New 装配（esc/landed 缺件=Beat 守位不炸——观测臂缺席不得拖采集主径）。
func New(esc EscDB, seat, ticket, venue string, fams []string, landed Landed) *Gate {
	return &Gate{esc: esc, seat: seat, ticket: ticket, venue: venue,
		fams: fams, persist: DefaultPersist, Landed: landed,
		now: time.Now, never: map[string]time.Time{}}
}

// Beat 一拍：逐族读实落水位→对差判→跨阈升件。零返回零上抛（观测面零
// 侵入——主径照跑）；nil 闸守位。
func (g *Gate) Beat(ctx context.Context) {
	if g == nil || g.esc == nil || g.Landed == nil {
		return // 守位（桩测/装配缺件——如实缺升件，采集成果不得被观测臂拖崩）
	}
	if len(g.fams) == 0 {
		g.escalateUndecidable(ctx) // 注册面缺席=闸自身 fail-closed——不可判 loud
		return
	}
	now := g.now()
	for _, fam := range g.fams {
		wm, err := g.Landed(ctx, fam)
		if err != nil {
			log.Printf("WARN [scanwm:%s] 族 %s 实落水位读败——本拍不可判下拍重读: %v", g.venue, fam, err)
			continue
		}
		if wm != nil && now.Sub(*wm) < g.persist {
			g.clear(fam) // 鲜落复常清零（滞回——对差撤销）
			continue
		}
		// 零鲜落两分：零行史（水位缺席）锚首见拍计持续；有史陈水（水位
		// 龄超阈）——水位龄即「零鲜落持续」实证，即时达阈。
		var dur time.Duration
		if wm == nil {
			first, seen := g.firstSeen(fam, now)
			if !seen || now.Sub(first) < g.persist {
				continue // 首见锚起/裕度未满
			}
			dur = now.Sub(first)
		} else {
			dur = now.Sub(*wm)
		}
		g.escalate(ctx, fam, wm, dur)
	}
}

// escalate 族对差升件（谱域缺行；scope=<venue>:<族>——组合域不落 venue 列
// 既有律；1/族/日 NOT EXISTS 原子去重——重启复拍不重升）。
func (g *Gate) escalate(ctx context.Context, fam string, wm *time.Time, dur time.Duration) {
	scope := g.venue + ":" + fam
	wmStr := "零行史"
	if wm != nil {
		wmStr = fmt.Sprintf("末落 %s", wm.UTC().Format("2006-01-02 15:04Z"))
	}
	note := fmt.Sprintf("族谱水位闸：%s 族零鲜落持续 %s（阈=%s；%s；期望集=注册面派生 %d 族——闸只报不修对拍归人肉/上游活性面）",
		fam, dur.Round(time.Second), g.persist, wmStr, len(g.fams))
	if _, err := g.esc.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件',$2,$3,$4,$5 WHERE NOT EXISTS (SELECT 1 FROM ops_event
		WHERE event='升件' AND esc_class=$4 AND esc_scope=$5 AND ts>=date_trunc('day',now()))`,
		g.seat, g.ticket, note, ClassMissing, scope); err != nil {
		log.Printf("WARN [scanwm:%s] 族 %s 升件行落败: %v", g.venue, fam, err)
	}
}

// escalateUndecidable 注册面缺席升件（谱域不可判；scope=venue 域级）。
func (g *Gate) escalateUndecidable(ctx context.Context) {
	note := "族谱水位闸：注册面缺席（应扫族集空）——闸自身 fail-closed 升「不可判」非静默放行（对拍装配面注册径；闸只报不修）"
	if _, err := g.esc.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件',$2,$3,$4,$5 WHERE NOT EXISTS (SELECT 1 FROM ops_event
		WHERE event='升件' AND esc_class=$4 AND esc_scope=$5 AND ts>=date_trunc('day',now()))`,
		g.seat, g.ticket, note, ClassUndecidable, g.venue); err != nil {
		log.Printf("WARN [scanwm:%s] 不可判升件行落败: %v", g.venue, err)
	}
}

func (g *Gate) firstSeen(fam string, now time.Time) (time.Time, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	first, ok := g.never[fam]
	if !ok {
		g.never[fam] = now
	}
	return first, ok
}

func (g *Gate) clear(fam string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.never, fam)
}
