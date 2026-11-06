// coverage.go —— 腿-行情覆盖探针（T-0392 防再发——业主令「不可能次次
// 我来提」）。病灶原型：funding-hedge 族现货腿 activeSet 派生臂缺失→
// 新开非 BTC/ETH 币现货腿零 tickers→腿估值/预警/basis 供价失明（「有物
// 无人喂」静默盲）。探针挂饥饿拍同槽（ProbeFamilies 同位省协程）：每拍
// 对拍「live/paper 策略各腿所需行情键×md_ticks 有无新鲜行」——腿键集
// =legset.ActiveSetUnion 全谱（fundingHedge 派生现货∪inst 本键∪dcd uly
// ∪basis 键，订阅集同源表达式禁自造第二套）；所需键=腿键 ∩订阅宇宙
// （okx.symbols 名单+IsUSDTInst 闸——虚构键宇宙外不求供价不炸）。腿缺
// 行情=ops_event `覆盖失明` 行+观测盲升件（24h 同窗按键去重）。只报不修
// ——探针升件不自动修（修归常票）；断流面死归 staleWatch/ops_face 域非本件。
package hunger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"o/internal/platform/legset"
)

const (
	// coverFreshMin 腿行情新鲜窗（分钟）——md_ticks.ts 血统龄阈。宽于
	// stalewatch 面死 3min：本件判单键覆盖非面死，低流动腿安静间隙
	// 免误报；持仓级标的 10min 无 tick 已属实质失明。
	coverFreshMin = 10
	// coverDedupHours 同窗去重——同键 24h 至多一报（esc_class+esc_scope
	// 等值键，结构性限度同族沉默探针先例）。
	coverDedupHours = 24
	// coverEscClass 升件类（判类册登记词——观测盲区类升件，族沉默同款）。
	coverEscClass = "观测盲"
	// coverTicket 探针归属票号（T-0392 立件——例行行携环票号同 T-0384 例）。
	coverTicket = "T-0392"
)

// AttachUniverse 挂宇宙扩件供给件（T-0416——探针读宇宙挂点；与
// collector ExpandSymbols/粗扫臂谱域同一供给件同读）。
func (s *Sensor) AttachUniverse(fn func(ctx context.Context) ([]string, error)) *Sensor {
	s.expandUniverse = fn
	return s
}

// ProbeLegCoverage 腿-行情覆盖探针一拍：腿键集×md_ticks 新鲜行对拍→
// 缺行情腿逐键落 `覆盖失明` 行+观测盲升件。读败=上抛（hungerTick 败情
// 自报位）非静默。
func (s *Sensor) ProbeLegCoverage(ctx context.Context) error {
	universe, err := s.subUniverse(ctx)
	if err != nil {
		return fmt.Errorf("腿覆盖探针 宇宙读败: %w", err)
	}
	required, err := s.legRequiredKeys(ctx, universe)
	if err != nil {
		return fmt.Errorf("腿覆盖探针 腿键集读败: %w", err)
	}
	if len(required) == 0 {
		return nil
	}
	var freshList []string
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE(array_agg(DISTINCT inst_id),'{}'::text[])
		FROM md_ticks WHERE inst_id=ANY($1) AND ts>now()-make_interval(mins=>$2)`,
		required, coverFreshMin).Scan(&freshList); err != nil {
		return fmt.Errorf("腿覆盖探针 新鲜行读败: %w", err)
	}
	fresh := map[string]bool{}
	for _, k := range freshList {
		fresh[k] = true
	}
	var blind []string
	for _, k := range required {
		if !fresh[k] {
			blind = append(blind, k)
		}
	}
	sort.Strings(blind)
	for _, k := range blind {
		if err := s.reportBlindLeg(ctx, k); err != nil {
			return fmt.Errorf("腿覆盖探针 %s 落行: %w", k, err)
		}
	}
	return nil
}

// subUniverse 订阅宇宙（okx.symbols 名单——collector 同款键；腿覆盖探针
// 与依赖对枚举同读禁第二套）。 "*" 放形=宇宙扩件供给件实读
// （AttachUniverse 挂线——未挂/读败=fail-closed 让位不虚构标的集）。
func (s *Sensor) subUniverse(ctx context.Context) (map[string]bool, error) {
	raw, err := s.cfg.Get("okx.symbols")
	if err != nil {
		return nil, err
	}
	var all []string
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("okx.symbols JSON 解析: %w", err)
	}
	out := make(map[string]bool, len(all))
	for _, sym := range all {
		if sym == "*" {
			if s.expandUniverse == nil {
				return nil, errors.New("okx.symbols=* 宇宙扩件探针未挂——fail-closed 不虚构标的集")
			}
			exp, err := s.expandUniverse(ctx)
			if err != nil {
				return nil, fmt.Errorf("宇宙扩读败（fail-closed 不虚构标的集）: %w", err)
			}
			for _, e := range exp {
				out[e] = true
			}
			continue
		}
		out[sym] = true
	}
	return out, nil
}

// legRequiredKeys 腿行情所需键集（归一链=legset.ActiveSetUnion 同源表达式
// ——订阅集同串禁第二套；∩订阅宇宙+IsUSDTInst 闸=派生合法性滤，虚构键
// 宇宙外不入所需集——同 syncActiveSet tickGate 判据语义）。
func (s *Sensor) legRequiredKeys(ctx context.Context, universe map[string]bool) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT inst_id FROM (`+legset.ActiveSetUnion+`) t`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		if legset.IsUSDTInst(k) && universe[k] {
			out = append(out, k)
		}
	}
	return out, rows.Err()
}

// reportBlindLeg 单缺行情腿落账：`覆盖失明` 行+观测盲升件（同窗 24h 按
// esc_scope=键去重——同键不暴，新键照报）。
func (s *Sensor) reportBlindLeg(ctx context.Context, inst string) error {
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		coverEscClass, inst, coverDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("同窗去重读败: %w", err)
	}
	if dup > 0 {
		return nil
	}
	note := fmt.Sprintf("腿行情覆盖失明 inst=%s：腿键所需而 md_ticks %dmin 内零新鲜行——腿估值/预警/basis 供价失明（探针只报不修，修归常票）",
		inst, coverFreshMin)
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'覆盖失明',$2,$3)`,
		hungerSeat, coverTicket, note); err != nil {
		return fmt.Errorf("覆盖失明行: %w", err)
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, coverTicket,
		fmt.Sprintf("腿行情覆盖失明 %s：live/paper 策略腿所需行情键近 %dmin md_ticks 零新鲜行——归一链订阅面/镜像谓词/写入闸断点待查（探针只报不修）",
			inst, coverFreshMin),
		coverEscClass, inst); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}
