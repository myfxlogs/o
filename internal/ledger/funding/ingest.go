// Package funding —— venue bills→fin_cashflow.funding_settle/fee 记账件（T-0044）。
// 铁律：billId 幂等（fin_venue_bill 主键——重启/重放不双记）；触发=事件驱动
// （私流余额帧钩子，D-0008 禁轮询主径——REST 只事件后拉取+boot 补漏）；杂币不
// 入账（ccy 闭集外弃记留源行）；禁推断记账——cashflow 行 source_ref=源表行。
package funding

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Bill venue 账单行（域自有型——ledger 不反向依赖交易域，装配层适配 okxexec.Bill）。
type Bill struct {
	BillID   string
	Type     string
	SubType  string
	InstID   string
	InstType string
	Ccy      string
	TsMs     int64
	BalChg   float64
	Fee      float64
	Pnl      float64
	Sz       float64
	Px       float64
	Raw      json.RawMessage
}

// BillSource 账单源（新→旧序单页拉取；afterBillID=分页游标——空=最新页）。
type BillSource interface {
	Bills(ctx context.Context, instType, afterBillID string) ([]Bill, error)
}

const (
	// OKX 账单子类型（官方 docs-v5 核证）：173 资金费支出/174 资金费收入。
	subFundingPay = "173"
	subFundingRcv = "174"
	subDcdDebit   = "160" // dcd 申购扣款（T-0160 ④——结构化产品资金流水 160-163）
	subDcdSettle  = "161" // dcd 回款
	subDcdYield   = "162" // dcd 收益
	subDcdRefund  = "163" // dcd 退款
	maxPages      = 5     // 单次事件拉取上限页（100/页——常规结算拍远不及）
)

// instTypesAll 账单拉取型谱（T-0062 根修②：SWAP+SPOT 双采——现货腿成本归集）。
// T-0160 三返修②：instType 过滤域实证「ASSET」=51000 非法参——改无过滤
// 拉取面（空串=/account/bills 不带 instType 唯一捕获径）；SWAP/SPOT 分型
// 显注 record 分派面（现网实证 dcd 160-163 在此口进——DEP 单帧式直通）。
var instTypesAll = []string{""}

// Ingester 记账执事。
type Ingester struct {
	pool *pgxpool.Pool
	src  BillSource
}

func NewIngester(pool *pgxpool.Pool, src BillSource) *Ingester {
	return &Ingester{pool: pool, src: src}
}

// Ingest 事件后拉取：首页→逐页向旧游标推进，遇已入库 billId 即止（增量切口）。
// 返回本次新记账账单数；失败上抛由调用方记数（不炸事件主径——见 Hook 包装）。
func (i *Ingester) Ingest(ctx context.Context) (int, error) {
	inserted := 0
	for _, it := range instTypesAll {
		n, err := i.ingestType(ctx, it)
		inserted += n
		if err != nil {
			return inserted, err
		}
	}
	return inserted, nil
}

// ingestType 单 instType 增量拉取（源行幂等切面——杂币过滤在 record 内分派）。
func (i *Ingester) ingestType(ctx context.Context, it string) (int, error) {
	after := ""
	inserted := 0
	for p := 0; p < maxPages; p++ {
		bills, err := i.src.Bills(ctx, it, after)
		if err != nil {
			return inserted, err
		}
		if len(bills) == 0 {
			return inserted, nil
		}
		oldest := bills[len(bills)-1].BillID
		for _, b := range bills {
			n, err := i.record(ctx, b)
			if err != nil {
				return inserted, err
			}
			inserted += n
		}
		if len(bills) < 100 { // 尾页——更旧已无
			return inserted, nil
		}
		var exists bool
		if err := i.pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM fin_venue_bill WHERE bill_id=$1)", oldest,
		).Scan(&exists); err != nil || exists {
			return inserted, err
		}
		after = oldest
	}
	return inserted, nil
}

// record 单账单落源表（幂等切面：ON CONFLICT 零行=已记过，cashflow 不再写）
// →按子类型分派 funding_settle/fee 入账。
func (i *Ingester) record(ctx context.Context, b Bill) (int, error) {
	occurred := time.UnixMilli(b.TsMs).UTC()
	var srcID int64
	err := i.pool.QueryRow(ctx, `INSERT INTO fin_venue_bill
		(bill_id,ts,inst_id,inst_type,ccy,type,sub_type,bal_chg,fee,pnl,sz,px,raw)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (bill_id) DO NOTHING RETURNING id`,
		b.BillID, occurred, b.InstID, b.InstType, b.Ccy, b.Type, b.SubType,
		b.BalChg, b.Fee, b.Pnl, b.Sz, b.Px, rawOrEmpty(b.Raw)).Scan(&srcID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil // 冲突零行=已记过（幂等切面）——cashflow 不再写
	}
	if err != nil {
		return 0, err // 上抛（写败≠已记过）
	}
	// 杂币不入账（源行已留档——USDT 本位闭集，ccy 缺省归一 USDT）。
	ccy := b.Ccy
	if ccy == "" {
		ccy = "USDT"
	}
	if ccy != "USDT" {
		return 0, nil
	}
	switch b.SubType {
	case subFundingPay, subFundingRcv:
		amt := b.BalChg
		if amt == 0 {
			amt = b.Pnl // 个性口径兜底：资金费行 balChg 空→pnl 查询总额口径（官方口径注）
		}
		return 1, i.post(ctx, "funding_settle", amt, b, ccy, srcID, occurred)
	case subDcdDebit: // 申购扣款=earn_out（balChg 负号原生——如实进账）
		return 1, i.post(ctx, "earn_out", b.BalChg, b, ccy, srcID, occurred)
	case subDcdSettle, subDcdYield, subDcdRefund: // 回款/收益/退款=earn_in
		return 1, i.post(ctx, "earn_in", b.BalChg, b, ccy, srcID, occurred)
	default:
		if b.Fee != 0 { // 交易费/返佣（开平仓 taker 费入 fee kind）
			return 1, i.post(ctx, "fee", b.Fee, b, ccy, srcID, occurred)
		}
	}
	return 0, nil
}

// post 入账（kind/闭集校验在 ledger.Post——此处只装配）。position_ref=按
// inst_id+时窗归仓（开仓≤结算时<平仓；无归仓=NULL 记账不丢）。
func (i *Ingester) post(ctx context.Context, kind string, amt float64, b Bill, ccy string, srcID int64, occurred time.Time) error {
	var posRef *int64
	var pid int64
	err := i.pool.QueryRow(ctx, `SELECT id FROM fin_position
		WHERE inst_id=$1 AND opened_at<=$2 AND (closed_at IS NULL OR closed_at>=$2)
		ORDER BY id DESC LIMIT 1`, b.InstID, occurred).Scan(&pid)
	if err == nil {
		posRef = &pid
	} // 归仓失败=无匹配仓（独立户流水等）——不拦入账
	_, err = i.pool.Exec(ctx, `INSERT INTO fin_cashflow
		(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES($1,'okx',$2,$3,$4,$5,'fin_venue_bill',$6)`,
		kind, ccy, amt, posRef, srcID, occurred)
	return err
}

// Hook 事件驱动包装（私流余额帧触发）：失败=warn 记数不炸事件主径。
func (i *Ingester) Hook() func(context.Context) {
	return func(ctx context.Context) {
		if _, err := i.Ingest(ctx); err != nil {
			log.Printf("[funding] bills 记账拉取败（下帧重试）: %v", err)
		}
	}
}

func rawOrEmpty(r json.RawMessage) json.RawMessage {
	if len(r) == 0 {
		return json.RawMessage("{}")
	}
	return r
}
