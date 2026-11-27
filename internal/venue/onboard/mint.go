// mint.go —— O-41378 件3：dialect-mint 拍——读 fin_venue_symbols 普查表做
// 配对（表上纯函数 census×registry 见 mintpair.go，禁边拉边配），别名命中
// →参数对拍（registry spec：ctVal approxEq/dollarPegged/digits>0/
// trade_mode=4——判据全 mt5.* 单源与手工 verifySymbolMapping 同尺）→
// GetQuote 探活（mint 拍内现拉，落行刻实况）→UpsertVerifiedWithAudit 落
// dialect 行（actor=auto-mint 同证据面）+census.map_state=auto_minted；
// 败=map_state=flagged+map_reason 落库可查（不猜不补——席位处置面
// SELECT census WHERE map_state='flagged'）。registry 补登记后 flagged
// 行下拍自愈转 mint（幂等重扫——dialect 同 PK upsert 三面合流不冲突）。
package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"o/internal/mt5"
)

// mintActor 自动落行审计 actor（ops_event seat 位——与手工 brokerCredActor
// 区分，撤行=席位 DELETE dialect 行 fail-closed 复归天然可逆）。
const mintActor = "auto-mint"

// mintPass 一拍 mint：registry enabled 全载→别名索引→凭据在册 mt5 族
// 逐户——pending/flagged 行配对判（表上纯函数）→过闸探活→落行/flag。
// 败=日志留痕不上抛（监测面同 probePass 律）。
func (r *Runner) mintPass(ctx context.Context, brokers map[string]credFamily) {
	reg, err := mt5.NewRegistryStore(r.fin).Enabled(ctx)
	if err != nil {
		log.Printf("WARN [onboard] registry 读败: %v", err)
		return
	}
	if len(reg) == 0 {
		return // 空登记表=无配对域——全 pending 如实等席位登记
	}
	idx := registryAliasIndex(reg)
	for _, b := range brokers {
		if b.missing || (b.kind != "" && b.kind != "mt5") || b.fields["login"] == "" {
			continue
		}
		if err := r.mintVenue(ctx, b, idx); err != nil {
			log.Printf("WARN [onboard] %s 方言 mint 拍败: %v", b.venue, err)
		}
	}
}

// mintVenue 单户 mint 拍：pending/flagged 行逐行配对（先纯函数判词——
// 命中才拨会话探活；零命中户零会话零 GetQuote——表上判数不拉数）。
func (r *Runner) mintVenue(ctx context.Context, b credFamily, idx map[string][]mt5.RegistryRow) error {
	rows, err := r.fin.Query(ctx, `SELECT venue_symbol,base_guess,suffix_guess,family_guess,
		ct_val,tick_sz,min_sz,lot_step,digits,trade_mode,profit_ccy,spec_hash
		FROM fin_venue_symbols
		WHERE venue=$1 AND account=$2 AND map_state IN ('pending','flagged')
		ORDER BY venue_symbol`, b.venue, b.fields["login"])
	if err != nil {
		return err
	}
	var list []censusMintRow
	for rows.Next() {
		var m censusMintRow
		if err := rows.Scan(&m.VenueSymbol, &m.BaseGuess, &m.SuffixGuess, &m.FamilyGuess,
			&m.CtVal, &m.TickSz, &m.MinSz, &m.LotStep, &m.Digits, &m.TradeMode,
			&m.ProfitCcy, &m.SpecHash); err != nil {
			rows.Close()
			return err
		}
		list = append(list, m)
	}
	rows.Close()
	if len(list) == 0 {
		return nil
	}
	// 第一遍纯函数配对——分出「需探活集」与「直接 flag 集」（会话只为探活集拨）。
	type liveCand struct {
		m   censusMintRow
		reg mt5.RegistryRow
	}
	var live []liveCand
	minted, flagged := 0, 0
	for _, m := range list {
		reg, reason := pairCensusRegistry(m, idx)
		if reg == nil {
			r.setCensusFlag(ctx, b, m.VenueSymbol, reason)
			flagged++
			continue
		}
		if reason := censusParamCheck(m, *reg); reason != "" {
			r.setCensusFlag(ctx, b, m.VenueSymbol, reason)
			flagged++
			continue
		}
		live = append(live, liveCand{m, *reg})
	}
	if len(live) == 0 {
		return r.mintReport(ctx, b, minted, flagged, len(list))
	}
	sess, err := r.dialCensus(ctx, b)
	if err != nil {
		return fmt.Errorf("探活拨号: %w", err) // 会话断=探活集如实缺席下拍重试（不 flag——非判词）
	}
	defer sess.Close()
	for i, it := range live {
		if i > 0 && probePace > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(probePace): // sched-exempt-ok：逐符探活步进（限频面守）
			}
		}
		if err := r.mintOne(ctx, b, sess, it.m, it.reg); err != nil {
			log.Printf("WARN [onboard] %s mint %s 败: %v", b.venue, it.m.VenueSymbol, err)
			r.setCensusFlag(ctx, b, it.m.VenueSymbol, "mint_err:"+err.Error())
			flagged++
			continue
		}
		minted++
	}
	return r.mintReport(ctx, b, minted, flagged, len(list))
}

// mintOne 单符落行：GetQuote 探活（bid/ask>0+点差带内——mt5.SpreadCapBPS
// 同尺）→DialectRow 全谱自 census 行补全→UpsertVerifiedWithAudit
// （actor=auto-mint 同证据面）→census.map_state=auto_minted+spread_live 实记。
func (r *Runner) mintOne(ctx context.Context, b credFamily, sess censusSession,
	m censusMintRow, reg mt5.RegistryRow) error {
	q, err := sess.GetQuoteOne(ctx, m.VenueSymbol)
	if err != nil {
		return fmt.Errorf("GetQuote 探活: %w", err)
	}
	mid := q.Mid()
	if mid <= 0 {
		return fmt.Errorf("探活拒：bid=%g ask=%g 死价快照", q.Bid, q.Ask)
	}
	sbps := q.SpreadBPS()
	if sbps > mt5.SpreadCapBPS {
		return fmt.Errorf("点差拒：%.0f bps > %d 带上（数据病快照拒收非警告）", sbps, mt5.SpreadCapBPS)
	}
	// O-41379：canonical=venue_symbol 恒等式——策略 inst_id=venue 原符是
	// 唯一活命名空间（Lookup(canonical=inst_id) 才可达）；registry 降为
	// 验证规范不借名，reg.Canonical 留 note/evidence/canonical_ref 作
	// 「验证归属族」证物位。
	row := mt5.DialectRow{
		Venue: b.venue, Account: b.fields["login"],
		Canonical: m.VenueSymbol, VenueSymbol: m.VenueSymbol,
		CtVal: *m.CtVal, TickSz: deref(m.TickSz), MinSz: deref(m.MinSz),
		LotStep: deref(m.LotStep), LotUnit: "lots",
		Digits: *m.Digits, TradeMode: *m.TradeMode,
		SpecHash: specHashStr(m.SpecHash),
	}
	ev, _ := json.Marshal(map[string]any{
		"bid": q.Bid, "ask": q.Ask, "mid": mid, "spread_bps": sbps,
		"source": "auto-mint", "registry": reg.Canonical, "registry_note": reg.Note,
	})
	login := b.fields["login"]
	note := fmt.Sprintf("auto-mint venue=%s login=%s sym=%s registry=%s ctVal=%g spread=%.0fbps",
		b.venue, login, m.VenueSymbol, reg.Canonical, row.CtVal, sbps)
	if err := mt5.NewDialectStore(r.fin).UpsertVerifiedWithAudit(ctx, row, ev,
		mintActor, "venue符号映射", note, b.venue+":"+login); err != nil {
		return fmt.Errorf("落行: %w", err)
	}
	_, err = r.fin.Exec(ctx, `UPDATE fin_venue_symbols SET map_state='auto_minted',
		canonical_ref=$4,map_reason=NULL,spread_live=$5,last_seen=now()
		WHERE venue=$1 AND account=$2 AND venue_symbol=$3`,
		b.venue, login, m.VenueSymbol, reg.Canonical, sbps)
	return err
}

// setCensusFlag flag 落账（败因全载明零空值——席位处置面
// SELECT census WHERE map_state='flagged' 可查；重判下拍覆盖）。
func (r *Runner) setCensusFlag(ctx context.Context, b credFamily, sym, reason string) {
	if _, err := r.fin.Exec(ctx, `UPDATE fin_venue_symbols SET map_state='flagged',
		map_reason=$4 WHERE venue=$1 AND account=$2 AND venue_symbol=$3`,
		b.venue, b.fields["login"], sym, reason); err != nil {
		log.Printf("WARN [onboard] %s flag 落账败 %s: %v", b.venue, sym, err)
	}
}

// mintReport 拍末计数行（ops_event 只载计数——明细归表不归事件流，spec 钉）。
func (r *Runner) mintReport(ctx context.Context, b credFamily, minted, flagged, scanned int) error {
	if minted == 0 && flagged == 0 {
		return nil
	}
	note := fmt.Sprintf("beat=mint key=%s:%s 扫描=%d 落行=%d flag=%d",
		b.venue, b.fields["login"], scanned, minted, flagged)
	_, err := r.fin.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,$2,$3,$4,$5,$6)`, seatName, "方言mint", ticketID, note, "", b.venue+":"+b.fields["login"])
	return err
}

func deref(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func specHashStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
