// dialect.go —— 方言对表面（T-0432）：官话↔venue 符号方言映射仓
// （fin_symbol_dialect——映射=确认件非推断件：候选机造+席位挑+机器硬验后
// 落行，本件只持仓不持判——判在 cmd/core 硬验机）。运行时查无映射=
// fail-closed 升件（ops_event）禁官名硬发——执行缝（mtexec.Dialect）与
// 漂移探针（onboard）经本件消费；粒度=(venue,account) 换服务器组=重映射
// 不继承品牌级（票面钦定）。
package mt5

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// ErrNoMapping 查无映射哨兵（fail-closed 判据——调用方升件禁 fallback 位：
// 官名硬发/静默换名皆禁线，唯一合法径=升件+拒单）。
var ErrNoMapping = errors.New("dialect: 查无映射")

// DialectRow 方言映射行（fin_symbol_dialect 投影——venue 语义词全 mtapi 实返
// 禁自造：ct_val=ContractSize；tick_sz=TickSize>0 实值否则 Points（T-0143
// Exness 实拍判据）；min_sz/lot_step=SymGroup.MinLots/LotsStep）。
type DialectRow struct {
	Venue, Account, Canonical, VenueSymbol string
	CtVal, TickSz, MinSz, LotStep          float64
	LotUnit                                string
	Digits, TradeMode                      int32
	SpecHash                               string
}

// DialectStore 方言映射仓（fin 链——SG 钱径域单库：dashboard 硬验落行/
// 执行缝查表/漂移探针对拍同源）。
type DialectStore struct{ Pool *pgxpool.Pool }

// NewDialectStore 装配（pool=fin 链库——调用方持对应池）。
func NewDialectStore(pool *pgxpool.Pool) *DialectStore { return &DialectStore{Pool: pool} }

// Lookup 官名→方言行（查无=ErrNoMapping——禁猜名禁官名硬发的判据原点）。
func (s *DialectStore) Lookup(ctx context.Context, venue, account, canonical string) (DialectRow, error) {
	var r DialectRow
	err := s.Pool.QueryRow(ctx, `SELECT venue,account,canonical,venue_symbol,ct_val,tick_sz,
		min_sz,lot_step,lot_unit,digits,trade_mode,spec_hash FROM fin_symbol_dialect
		WHERE venue=$1 AND account=$2 AND canonical=$3`,
		venue, account, canonical).Scan(&r.Venue, &r.Account, &r.Canonical, &r.VenueSymbol,
		&r.CtVal, &r.TickSz, &r.MinSz, &r.LotStep, &r.LotUnit, &r.Digits, &r.TradeMode, &r.SpecHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return DialectRow{}, fmt.Errorf("%w: %s@%s/%s", ErrNoMapping, venue, account, canonical)
	}
	if err != nil {
		return DialectRow{}, err
	}
	return r, nil
}

// UpsertVerified 硬验过闸行落库（唯一写径——判在 cmd/core 硬验机，本件不持判；
// quoteSnapshot=protojson SymbolVerifyEvidence 入 jsonb——架构律跨面载荷
// proto 唯一源）。
func (s *DialectStore) UpsertVerified(ctx context.Context, r DialectRow, quoteSnapshot []byte) error {
	return s.upsert(ctx, s.Pool, r, quoteSnapshot)
}

// UpsertVerifiedWithAudit 落行+审计行单事务（T-0430 verify-then-write 同律——
// 行/账原子零半截：行落审计缺=半交付态禁）。
func (s *DialectStore) UpsertVerifiedWithAudit(ctx context.Context, r DialectRow,
	quoteSnapshot []byte, seat, event, note, escScope string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := s.upsert(ctx, tx, r, quoteSnapshot); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,note,esc_scope)
		VALUES($1,$2,$3,$4)`, seat, event, note, escScope); err != nil {
		return fmt.Errorf("审计行写: %w", err)
	}
	return tx.Commit(ctx)
}

// upsert 行写核（pool/tx 双面——execer 接口两形皆满足）。
func (s *DialectStore) upsert(ctx context.Context, ex interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, r DialectRow, quoteSnapshot []byte) error {
	_, err := ex.Exec(ctx, `INSERT INTO fin_symbol_dialect(venue,account,canonical,
		venue_symbol,ct_val,tick_sz,min_sz,lot_step,lot_unit,digits,trade_mode,spec_hash,
		quote_snapshot,verified_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,now())
		ON CONFLICT (venue,account,canonical) DO UPDATE SET venue_symbol=EXCLUDED.venue_symbol,
		ct_val=EXCLUDED.ct_val,tick_sz=EXCLUDED.tick_sz,min_sz=EXCLUDED.min_sz,
		lot_step=EXCLUDED.lot_step,lot_unit=EXCLUDED.lot_unit,digits=EXCLUDED.digits,
		trade_mode=EXCLUDED.trade_mode,spec_hash=EXCLUDED.spec_hash,
		quote_snapshot=EXCLUDED.quote_snapshot,verified_at=now()`,
		r.Venue, r.Account, r.Canonical, r.VenueSymbol, r.CtVal, r.TickSz, r.MinSz,
		r.LotStep, r.LotUnit, r.Digits, r.TradeMode, r.SpecHash, quoteSnapshot)
	return err
}

// Rows 账户全谱方言行（漂移探针⑤读面——逐行 SymbolParams hash 对拍基线）。
func (s *DialectStore) Rows(ctx context.Context, venue, account string) ([]DialectRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT venue,account,canonical,venue_symbol,ct_val,tick_sz,
		min_sz,lot_step,lot_unit,digits,trade_mode,spec_hash FROM fin_symbol_dialect
		WHERE venue=$1 AND account=$2 ORDER BY canonical`, venue, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DialectRow
	for rows.Next() {
		var r DialectRow
		if err := rows.Scan(&r.Venue, &r.Account, &r.Canonical, &r.VenueSymbol, &r.CtVal,
			&r.TickSz, &r.MinSz, &r.LotStep, &r.LotUnit, &r.Digits, &r.TradeMode, &r.SpecHash); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Escalate 升件落账（ops_event append-only——查无映射/漂移病的业主可见径；
// 败=err 上浮不静默，调用方并入主错径）。
func (s *DialectStore) Escalate(ctx context.Context, seat, event, note, escScope string) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,esc_class,esc_scope)
		VALUES($1,$2,$3,'drift',$4)`, seat, event, note, escScope)
	return err
}

// VenueTranslator 单账户方言译件（DialectStore+(venue,account) 绑定——
// mtexec.Dialect 产线实现；查无映射=ErrNoMapping 原样上浮，升件由执行缝
// Escalate 落行）。仿真层同缝可设桩（T-0426 缝位同款）。
type VenueTranslator struct {
	S              *DialectStore
	Venue, Account string
	Seat           string // 升件落账席标（产线=o-core 执行臂）
}

// VenueSymbol 官名→venue 方言名（查无=ErrNoMapping）。
func (t VenueTranslator) VenueSymbol(ctx context.Context, canonical string) (string, error) {
	r, err := t.S.Lookup(ctx, t.Venue, t.Account, canonical)
	if err != nil {
		return "", err
	}
	return r.VenueSymbol, nil
}

// Escalate 查无映射升件（ops_event——fail-closed 拒单同拍落账）。
func (t VenueTranslator) Escalate(ctx context.Context, canonical, reason string) {
	_ = t.S.Escalate(ctx, t.seat(), "方言缺映射",
		fmt.Sprintf("canonical=%s reason=%s——fail-closed 拒单（禁官名硬发）", canonical, reason),
		t.Venue+":"+t.Account)
}

func (t VenueTranslator) seat() string {
	if t.Seat != "" {
		return t.Seat
	}
	return "o-core"
}

// SpecHash SymbolParams 语义指纹（漂移探针⑤基线——broker 改后缀/下架/换组/
// ctVal 变即指纹漂移=单位错配病重演防）。sha256 over 语义字段 canonical 序
// （全取 venue 实返字段——判据面禁自造值）。
func SpecHash(si *mt5pb.SymbolInfo, sg *mt5pb.SymGroup) string {
	h := sha256.New()
	w := func(name string, v any) {
		fmt.Fprintf(h, "%s=%v\n", name, v)
	}
	if si != nil {
		w("ContractSize", si.GetContractSize())
		w("Digits", si.GetDigits())
		w("TickSize", si.GetTickSize())
		w("Points", si.GetPoints())
		w("Spread", si.GetSpread())
		w("ProfitCurrency", si.GetProfitCurrency())
		w("MarginCurrency", si.GetMarginCurrency())
	}
	if sg != nil {
		w("MinLots", sg.GetMinLots())
		w("MaxLots", sg.GetMaxLots())
		w("LotsStep", sg.GetLotsStep())
		w("TradeMode", int32(sg.GetTradeMode()))
		w("SwapType", int32(sg.GetSwapType()))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// QuoteSnapshot 单符报价快照件（硬验探活+漂移探针共用——bid/ask≤0=死价
// 如实上浮不虚构）。GetQuote 单符径（msNotOlder 缺省=末拍回即——探测非流径）。
type QuoteSnapshot struct {
	Symbol         string
	Bid, Ask       float64
	ServerTimeUnix int64
}

// Mid 快照中价（死价=0 如实）。
func (q QuoteSnapshot) Mid() float64 {
	if q.Bid <= 0 || q.Ask <= 0 {
		return 0
	}
	return (q.Bid + q.Ask) / 2
}

// SpreadBPS 点差 bps（死价=0——合理性判在调用方硬验机）。
func (q QuoteSnapshot) SpreadBPS() float64 {
	mid := q.Mid()
	if mid <= 0 {
		return 0
	}
	return (q.Ask - q.Bid) / mid * 10000
}

// GetQuoteOne 单符报价快照（硬验机/漂移探针共用读径——QuoteService 单查非流）。
func (c *Client) GetQuoteOne(ctx context.Context, symbol string) (QuoteSnapshot, error) {
	r, err := c.mt5.GetQuote(c.authed(ctx), &mt5pb.GetQuoteRequest{Id: c.token, Symbol: symbol})
	if err != nil {
		return QuoteSnapshot{}, err
	}
	if r.GetError() != nil {
		return QuoteSnapshot{}, fmt.Errorf("getQuote %s: %s", symbol, r.GetError().GetMessage())
	}
	q := r.GetResult()
	out := QuoteSnapshot{Symbol: symbol, Bid: q.GetBid(), Ask: q.GetAsk()}
	if t := q.GetTime(); t != nil {
		out.ServerTimeUnix = t.AsTime().Unix()
	}
	return out, nil
}

// ProbeAdapter 漂移探针窄面适配（onboard probeSession 形——SpecHashOf 返
// 语义指纹+ctVal 现值标量面，proto 型不出 mt5 包界——D-514 方向白名单守）。
type ProbeAdapter struct{ C *Client }

// SpecHashOf 符号语义指纹+现 ctVal（探针对拍面——SymbolParams 实返）。
func (a ProbeAdapter) SpecHashOf(ctx context.Context, symbol string) (string, float64, error) {
	si, sg, err := a.C.SymbolSpec(ctx, symbol)
	if err != nil {
		return "", 0, err
	}
	return SpecHash(si, sg), si.GetContractSize(), nil
}

// GetQuoteOne 探活快照透传。
func (a ProbeAdapter) GetQuoteOne(ctx context.Context, symbol string) (QuoteSnapshot, error) {
	return a.C.GetQuoteOne(ctx, symbol)
}

// Connect 会话建立透传。
func (a ProbeAdapter) Connect(ctx context.Context, cr Credentials) error { return a.C.Connect(ctx, cr) }

// Disconnect 会话断开透传。
func (a ProbeAdapter) Disconnect(ctx context.Context) { a.C.Disconnect(ctx) }

// Close 连接关闭透传。
func (a ProbeAdapter) Close() { a.C.Close() }

// SymbolSpec 符号语义全形（SymbolParams 实返裸面——硬验机对拍源；nil result=
// 符号不存在如实——禁猜名判据的 venue 判原点）。
func (c *Client) SymbolSpec(ctx context.Context, symbol string) (*mt5pb.SymbolInfo, *mt5pb.SymGroup, error) {
	r, err := c.mt5.SymbolParams(c.authed(ctx), &mt5pb.SymbolParamsRequest{Id: c.token, Symbol: symbol})
	if err != nil {
		return nil, nil, err
	}
	if r.GetError() != nil {
		return nil, nil, fmt.Errorf("symbolParams %s: %s", symbol, r.GetError().GetMessage())
	}
	res := r.GetResult()
	if res == nil {
		return nil, nil, fmt.Errorf("symbolParams %s 空体", symbol)
	}
	return res.GetSymbolInfo(), res.GetSymbolGroup(), nil
}
