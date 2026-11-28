// Package ossfam —— OSS 三族骨架评估臂（工单O-41394径乙：candidate 态
// funding-extreme-fade/mark-last-disloc-fade/xsec-perp-mom-basket 在
// grammar/roster 有名无实——md_1106 骨架卡登 candidate 但 fin_opp_eval
// 全 0 行，breed_stall 24h 内 3 火。径甲判据：泛化「按 roster/卡态枚举
// 待评族」评估件不存在——researchArms 逐族硬编码枚举、hypothesis 只产
// 卡不产 eval、grammar 对 Existing=true 族跳过、stakecarry 只写
// template='stake-carry'；故径乙逐族 Runner 实例，臂形仿
// dirmom/gridcfd 同构）。
//
// 影子产出零执行零提案径（spec 钦定）：本包只写 fin_opp_eval——verdict
// 全真值（proposed/rejected/unevaluable 照判据实判），fin_proposal/
// fin_strategy/执行面零触；三族收养=席权 decide 窄门另件，execfam 册载
// paper_only 不在本臂消费。枚举≠评估：grammar.go 三族定义行/探针/
// execfam 册均禁动。
package ossfam

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
	"o/internal/platform/writefail"
	"o/internal/research/armwatch"
	"o/internal/research/evalgate"
)

// evalRow 一族一拍评估卡（inst 级=逐标的行；篮级=单行 inst_id='XSEC-MOM-BASKET'）。
type evalRow struct {
	inst     string
	verdict  string // proposed/rejected/unevaluable——判据实判不美化
	reason   string
	grossAPR float64
	netAPR   float64
	legs     []string // 卡 leg_primitives 原语名入 legs 列
	evidence map[string]any
}

// arm 三族共享臂壳：pool/cfg/watch/rej/seat/template/direction 单点持有，
// Run 环+commit 写径单点——族差分全在 tick（params/scan/eval 语义各文件）。
type arm struct {
	pool      *pgxpool.Pool
	cfg       *config.Client
	watch     *armwatch.Watch
	rej       *rejects.Door
	seat      string
	desc      string
	template  string // fin_opp_eval.template=fin_opp_template.template 册键
	direction string // fin_opp_eval.direction 词表值=族 grammar Direction
}

func newArm(pool *pgxpool.Pool, cfg *config.Client, rej *rejects.Door, seat, desc, template, direction string) *arm {
	return &arm{pool: pool, cfg: cfg, watch: armwatch.New(seat), rej: rej,
		seat: seat, desc: desc, template: template, direction: direction}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修同径）。
func (a *arm) AttachAlerter(al *notify.Alerter) *arm { a.watch.AttachAlerter(al); return a }

// Run 在岗环（注册+心跳+ticker 拍——tick 返下拍间隔，判据每拍重读热启；
// sched-exempt-ok：评估节拍非轮询拉取）。
func (a *arm) Run(ctx context.Context, errc chan<- error, tick func(ctx context.Context) (time.Duration, error)) {
	if err := roster.Register(ctx, a.pool, a.seat, "research", a.desc, nil); err != nil {
		errc <- fmt.Errorf("%s register: %w", a.seat, err)
		return
	}
	go roster.BeatLoop(ctx, a.pool, a.seat, 30*time.Second, errc)
	iv := 4 * time.Hour // 判据未读前兜底节距
	for {
		if d, err := tick(ctx); err != nil {
			log.Printf("[research/%s] 拍败: %v", a.template, err)
			writefail.Report(ctx, "research/"+a.template, "工单#O-41394", "RESEARCH-TICK/"+a.template, "拍败: "+err.Error())
		} else if d > 0 {
			iv = d
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(iv): // sched-exempt-ok：评估节拍非轮询拉取
		}
	}
}

// commit 评估卡批落表：evalgate 指纹闸去重（verdict+net_apr 未迁跳 INSERT
// ——md_1097 版本史语义不破）+INSERT fin_opp_eval。零提案零执行：本臂
// 不写 fin_proposal/fin_strategy——candidate 影子产出，收养另件。
func (a *arm) commit(ctx context.Context, rows []evalRow) error {
	if len(rows) == 0 {
		return nil
	}
	return pgx.BeginFunc(ctx, a.pool, func(tx pgx.Tx) error {
		for _, c := range rows {
			ev, _ := json.Marshal(c.evidence)
			legs, _ := json.Marshal(c.legs)
			_, same, err := evalgate.SameFingerprint(ctx, tx, c.inst, a.template, a.direction, verdictOf(c), c.netAPR)
			if err != nil {
				return err
			}
			if same {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
				c.inst, a.template, a.direction, c.grossAPR, c.netAPR,
				verdictOf(c), legs, ev); err != nil {
				return err
			}
		}
		return nil
	})
}

func verdictOf(c evalRow) string { return c.verdict }
