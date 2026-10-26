// mirror.go —— 八桥表投影 SQL 面（T-0168/D-0030 §三谓词+§六 per-table 键）。
// 全部 upsert 语义按 D-0018 同键最新赢：ts 判表 WHERE EXCLUDED.ts>本表.ts；
// md_funding 同 ts=同值 DO NOTHING；md_macro_event 无 ts 全列覆写；
// fin_proposal 前向禁写 state 列组（回环禁——SG ev_proposal_sg 按 UPDATE OF
// state 触发，不写不触发）。numeric 以文本搬运保精度（零 float 折损）。
package projector

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 活跃集信号谓词（D-0030 §三：carry_diff/xvenue_spread 全量，funding_view/depth5 活跃集键）。
const signalGlobalKinds = `('carry_diff','xvenue_spread')`
const signalActiveKinds = `('funding_view','depth5')`

// md 查询面（正本侧）——watermark 后的谓词切片。

const qTicks = `SELECT ts,venue,inst_id,last,bid,ask,vol_24h,recv_ts FROM md_ticks
	WHERE ts>$1 AND inst_id=ANY($2) ORDER BY ts`

const qMarkIdx = `SELECT ts,venue,inst_id,channel,px,recv_ts FROM md_mark_index
	WHERE ts>$1 AND inst_id=ANY($2) ORDER BY ts`

const qFunding = `SELECT ts,venue,inst_id,funding_rate,next_funding_rate,next_funding_ts,recv_ts FROM md_funding
	WHERE ts>$1 AND inst_id=ANY($2) ORDER BY ts`

const qSignal = `SELECT kind,key,val,meta,ts FROM md_signal
	WHERE ts>$1 AND (kind IN ` + signalGlobalKinds + ` OR (kind IN ` + signalActiveKinds + ` AND key=ANY($2)))
	ORDER BY ts`

const qDcd = `SELECT ts,venue,surface,uly,product_id,direction,strike,expiry,annualized_yield,min_size,max_size,spot,raw
	FROM md_dcd_product WHERE ts>$1 AND product_id=ANY($2) ORDER BY ts`

const qEvalFresh = `SELECT e.id,e.ts,e.inst_id,e.template,e.direction,e.gross_apr,e.net_apr,e.verdict,e.legs,e.evidence
	FROM fin_opp_eval e WHERE e.ts>$1
	AND EXISTS (SELECT 1 FROM fin_proposal p WHERE p.eval_id=e.id) ORDER BY e.ts`

const qProposalByID = `SELECT id,eval_id,inst_id,direction,net_apr,created_at FROM fin_proposal WHERE id=ANY($1)`

const qEvalByID = `SELECT id,ts,inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence
	FROM fin_opp_eval WHERE id=ANY($1)`

const qMacroAll = `SELECT event_date,kind,title,source FROM md_macro_event ORDER BY event_date`

// dualwrite 旁写泵变体（T-0169 对拍期——无谓词全量搬运：SG 旧行为保真，旁写语义）。
const qTicksDual = `SELECT ts,venue,inst_id,last,bid,ask,vol_24h,recv_ts FROM md_ticks WHERE ts>$1 ORDER BY ts`
const qMarkIdxDual = `SELECT ts,venue,inst_id,channel,px,recv_ts FROM md_mark_index WHERE ts>$1 ORDER BY ts`
const qFundingDual = `SELECT ts,venue,inst_id,funding_rate,next_funding_rate,next_funding_ts,recv_ts FROM md_funding WHERE ts>$1 ORDER BY ts`
const qSignalDual = `SELECT kind,key,val,meta,ts FROM md_signal WHERE ts>$1 ORDER BY ts`
const qDcdDual = `SELECT ts,venue,surface,uly,product_id,direction,strike,expiry,annualized_yield,min_size,max_size,spot,raw
	FROM md_dcd_product WHERE ts>$1 ORDER BY ts`
const qEvalDual = `SELECT e.id,e.ts,e.inst_id,e.template,e.direction,e.gross_apr,e.net_apr,e.verdict,e.legs,e.evidence
	FROM fin_opp_eval e WHERE e.ts>$1 ORDER BY e.ts`

const qConfigAll = `SELECT key,value::text,generation,updated_at FROM app_config ORDER BY key`

const uConfig = `INSERT INTO app_config(key,value,generation,updated_at)
	VALUES($1,$2::jsonb,$3,$4::timestamptz)
	ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value,generation=EXCLUDED.generation,
		updated_at=EXCLUDED.updated_at`

const dConfigGone = `DELETE FROM app_config WHERE key=ANY($1)`

// SG 落地面（镜像侧 upsert——per-table 键/新旧判照 D-0030 §六表）。

const uTicks = `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask,vol_24h,recv_ts)
	VALUES($1::timestamptz,$2,$3,$4::numeric,$5::numeric,$6::numeric,$7::numeric,$8::timestamptz)
	ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last,bid=EXCLUDED.bid,
		ask=EXCLUDED.ask,vol_24h=EXCLUDED.vol_24h,recv_ts=EXCLUDED.recv_ts
	WHERE EXCLUDED.ts > md_ticks.ts`

const uMarkIdx = `INSERT INTO md_mark_index(ts,venue,inst_id,channel,px,recv_ts)
	VALUES($1::timestamptz,$2,$3,$4,$5::numeric,$6::timestamptz)
	ON CONFLICT (venue,inst_id,channel) DO UPDATE SET ts=EXCLUDED.ts,px=EXCLUDED.px,recv_ts=EXCLUDED.recv_ts
	WHERE EXCLUDED.ts > md_mark_index.ts`

const uSignal = `INSERT INTO md_signal(kind,key,val,meta,ts)
	VALUES($1,$2,$3::numeric,$4::jsonb,$5::timestamptz)
	ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val,meta=EXCLUDED.meta,ts=EXCLUDED.ts
	WHERE EXCLUDED.ts > md_signal.ts`

const uDcd = `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,direction,strike,expiry,
	annualized_yield,min_size,max_size,spot,raw)
	VALUES($1::timestamptz,$2,$3,$4,$5,$6,$7::numeric,$8::timestamptz,$9::numeric,$10::numeric,$11::numeric,$12::numeric,$13::jsonb)
	ON CONFLICT (venue,product_id) DO UPDATE SET ts=EXCLUDED.ts,surface=EXCLUDED.surface,uly=EXCLUDED.uly,
		direction=EXCLUDED.direction,strike=EXCLUDED.strike,expiry=EXCLUDED.expiry,
		annualized_yield=EXCLUDED.annualized_yield,min_size=EXCLUDED.min_size,max_size=EXCLUDED.max_size,
		spot=EXCLUDED.spot,raw=EXCLUDED.raw
	WHERE EXCLUDED.ts > md_dcd_product.ts`

const uFunding = `INSERT INTO md_funding(ts,venue,inst_id,funding_rate,next_funding_rate,next_funding_ts,recv_ts)
	VALUES($1::timestamptz,$2,$3,$4::numeric,$5::numeric,$6::timestamptz,$7::timestamptz)
	ON CONFLICT (venue,inst_id,ts) DO NOTHING` // 同键同拍=同值（D-0030 §六表）

const uMacro = `INSERT INTO md_macro_event(event_date,kind,title,source)
	VALUES($1::date,$2,$3,$4)
	ON CONFLICT (event_date) DO UPDATE SET kind=EXCLUDED.kind,title=EXCLUDED.title,source=EXCLUDED.source`

const uProposal = `INSERT INTO fin_proposal(id,eval_id,inst_id,direction,net_apr,created_at)
	VALUES($1,$2,$3,$4,$5::numeric,$6::timestamptz)
	ON CONFLICT (id) DO UPDATE SET eval_id=EXCLUDED.eval_id,inst_id=EXCLUDED.inst_id,
		direction=EXCLUDED.direction,net_apr=EXCLUDED.net_apr,created_at=EXCLUDED.created_at`

// 前向无 state 列组（D-0030 写权矩阵——lifecycle 唯一写方；不写=不触发 SG ev_proposal_sg，零回环）。

const uEval = `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
	VALUES($1,$2::timestamptz,$3,$4,$5,$6::numeric,$7::numeric,$8,$9::jsonb,$10::jsonb)
	ON CONFLICT (id) DO UPDATE SET ts=EXCLUDED.ts,inst_id=EXCLUDED.inst_id,template=EXCLUDED.template,
		direction=EXCLUDED.direction,gross_apr=EXCLUDED.gross_apr,net_apr=EXCLUDED.net_apr,
		verdict=EXCLUDED.verdict,legs=EXCLUDED.legs,evidence=EXCLUDED.evidence
	WHERE EXCLUDED.ts > fin_opp_eval.ts`

const uProposalReverse = `UPDATE fin_proposal SET state=$2,retire_reason=$3,retired_at=$4::timestamptz WHERE id=$1`

// batchExec 单事务批量落 SG（镜像写拍原子——断连半批不留）。rows=参数行构造器。
func batchExec(ctx context.Context, sg *pgxpool.Pool, stmt string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := sg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mirror tx: %w", err)
	}
	b := &pgx.Batch{}
	for _, r := range rows {
		b.Queue(stmt, r...)
	}
	br := tx.SendBatch(ctx, b)
	for i := 0; i < len(rows); i++ { // 逐条驱动批（中段错误即回滚整批）
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			_ = tx.Rollback(ctx)
			return fmt.Errorf("mirror batch[%d/%d]: %w", i+1, len(rows), err)
		}
	}
	if err := br.Close(); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("mirror batch close: %w", err)
	}
	return tx.Commit(ctx)
}
