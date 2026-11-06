// Package decision —— ops_decision 决策记忆写径（T-0415 决策记忆表——钱径判据
// 与席位裁词同构落账）。病灶对治：仓位「为什么平」零凭据、DM 裁决「通过」裸行
// ——拒的有案、活的没案、死的没墓志铭。
// 写面分工：seat 裁词=库内镜像触发器全径必经（ops_event 裁决/驳回/参数调整行
// →seat 行；lesson status→verdicted→裁定席行——fin_0129，异构写面库内收口同
// fin_0121 判据）；fin 机器域行=本包写径（lifecycle/guard/review 调）。
// 纪律二件：①词集/裸词=库内守（表 CHECK+ops_decision_write RAISE——写径不
// 自造词不重复校验）；②写败=升件自报禁静默（LostEsc——记忆丢失不许无声；
// 记忆行继事实行后写不阻钱径——钱径事实不因记忆面败停）。
package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier 写缝（pgxpool.Pool/pgx.Tx 同构——非事务与嵌事务写径同口）。
type Querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Row 决策行。fin 域 actor=机器域件名（lifecycle/guard/review）、
// action=open|close|adjust|reject、ref_id=strategy/proposal id；
// seat 域 actor=席位名、action=裁决|驳回|参数调整|升件裁量、ref_id=票号/案号。
type Row struct {
	Domain, Actor, RefID, Action, Reason string
	Payload                              map[string]any // nil=无载荷
}

// Write 单行落账（库函数 ops_decision_write 同径——词集 CHECK/裸词 RAISE
// 库内单源）。败=error 上抛——调用面接 LostEsc 升件径（禁静默）。
func Write(ctx context.Context, q Querier, r Row) error {
	var payload any
	if r.Payload != nil {
		b, err := json.Marshal(r.Payload)
		if err != nil {
			return fmt.Errorf("ops_decision payload: %w", err)
		}
		payload = string(b)
	}
	_, err := q.Exec(ctx, `SELECT ops_decision_write($1,$2,$3,$4,$5,$6::jsonb)`,
		r.Domain, r.Actor, r.RefID, r.Action, r.Reason, payload)
	return err
}

// WriteEsc 写+败升件合件（本包钦定调用形——写败即升件不沉默）。
func WriteEsc(ctx context.Context, pool *pgxpool.Pool, r Row) {
	if err := Write(ctx, pool, r); err != nil {
		LostEsc(ctx, pool, r, err)
	}
}

// MoneyFacts 「实现的vs期望的」钦定载荷件（平仓行必携）：期望=提案评估
// net_apr（strategy→proposal→eval 链），实现=fin_cashflow 净合。
// positionID>0=单仓口径（guard 平仓行）；0=策略全仓口径（lifecycle 退役行）。
// 读败/行缺=零值键仍在（缺省显形非省键——消费面对拍口径稳）。
func MoneyFacts(ctx context.Context, q Querier, strategyID, positionID int64) map[string]any {
	var expected *float64
	var realized float64
	if positionID > 0 {
		err := q.QueryRow(ctx, `SELECT
				(SELECT pr.net_apr::float8 FROM fin_proposal pr
					JOIN fin_strategy s ON s.proposal_ref=pr.id WHERE s.id=$2),
				(SELECT COALESCE(sum(cf.amount),0)::float8 FROM fin_cashflow cf
					WHERE cf.position_ref=$1)`,
			positionID, strategyID).Scan(&expected, &realized)
		if err != nil {
			log.Printf("WARN [decision] MoneyFacts 仓读败 #%d/#%d: %v", strategyID, positionID, err)
		}
	} else {
		err := q.QueryRow(ctx, `SELECT
				(SELECT pr.net_apr::float8 FROM fin_proposal pr
					JOIN fin_strategy s ON s.proposal_ref=pr.id WHERE s.id=$1),
				(SELECT COALESCE(sum(cf.amount),0)::float8 FROM fin_cashflow cf
					JOIN fin_position pos ON pos.id=cf.position_ref
					WHERE pos.strategy_ref=$1)`,
			strategyID).Scan(&expected, &realized)
		if err != nil {
			log.Printf("WARN [decision] MoneyFacts 策略读败 #%d: %v", strategyID, err)
		}
	}
	return map[string]any{"expected_net_apr": expected, "realized_usd": realized}
}

// LostEsc 写败升件自报（禁静默钦定件——ops_event 升件行 esc_class=写败
// esc_scope=ops_decision 判类册登记词；前缀同窗 24h 去重防每拍爆量——
// ReportBreak 同窗去重同式；升件行自身写败=log 末环——自报径不再递归自报）。
func LostEsc(ctx context.Context, pool *pgxpool.Pool, r Row, cause error) {
	prefix := fmt.Sprintf("ops_decision 写败——%s/%s ref=%s", r.Domain, r.Action, r.RefID)
	tag, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件','',$2,'写败','ops_decision'
		WHERE NOT EXISTS (SELECT 1 FROM ops_event WHERE event='升件'
			AND note LIKE $3 AND ts>now()-interval '24 hours')`,
		r.Actor, prefix+": "+fmt.Sprint(cause), prefix+"%")
	if err != nil {
		log.Printf("WARN [decision] 写败升件行落账败（末环静默——升件径自报禁递归）: %v", err)
		return
	}
	if tag.RowsAffected() == 0 {
		return // 同窗已报——24h 内至多一报
	}
	log.Printf("[decision] ops_decision 写败已升件自报: %s: %v", prefix, cause)
}
