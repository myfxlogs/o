// facesnap.go —— 面清单快照 diff（T-0405「该建未建」类残余收口件）：
// 探针查产消对不查「系统该有什么」——对面清单快照+diff 退化为可见差集行。
//
// 语义：每拍机器可列面五类（service=ops_event 活跃席/table=public 基表/
// dep_pair=枚举器产消对签名/template=评估+模板族/gate_domain=app_config
// 键域）整面落批（ops_face_snapshot 窄表）→与上拍批 diff→新面逐条
// 「面清单新面」行+观测盲升件（esc_scope=face:<kind>:<face>，同窗 24h
// 去重同 depblind 式）。首拍=基线建册零升件（依赖对清单首跑人工基线先例
// ——面宇宙人工过一遍后新增才值得看）。快照仅留最近两拍批——diff 毕
// 旧批即清（窄表不建重型件）。只报不修，败情并入 ProbeDeps 拍级错非静默。
package hunger

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// faceTicket 面清单 diff 归属票号（T-0405 立件——例行行携环票号同 T-0394 例）。
const faceTicket = "T-0405"

// faceBaselineWhitelist 面清单人工基线册——合法新面登记位（注册式：
// key=面键 'face:<kind>:<face>'=esc_scope 同键，'*' 尾=前缀通配；value=
// 理由键）。册内面快照差集照落「面清单新面」行携理由键留账但升件免——
// 间歇写席（ops_event 7d 活跃窗出入摆动）窗摆回不重复炸票/起草；
// 拒册=不登记——可疑新面照升件等人工判。depBlindWhitelist 同式先例。
var faceBaselineWhitelist = map[string]string{
	"face:service:仓巡": "仓巡=guard Warden 在册员工（risk 域 fin_employee_registry active）——T-0419 裸袋/T-0423 守卫环自残探针 ops_event 写径合法面；间歇写席 7d 活跃窗摆动登记不重复升",
}

// faceRow 单面行（kind=面类五键，face=面键——同拍批内 kind+face 唯一）。
type faceRow struct {
	kind string
	face string
}

// faceInventory 机器可列面全集（五类——服务/表/产消对/模板/闸键域）：
// 全部库内可枚举零外部依赖；dep_pair 用本拍枚举器产出对签名（同源不虚造）。
func (s *Sensor) faceInventory(ctx context.Context, pairs []depPair) ([]faceRow, error) {
	var out []faceRow
	addRows := func(kind, q string, args ...any) error {
		rows, err := s.pool.Query(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var f string
			if err := rows.Scan(&f); err != nil {
				return err
			}
			out = append(out, faceRow{kind, f})
		}
		return rows.Err()
	}
	// 服务=ops_event 近窗活跃席（席位活性自报面——写者席即服务面投影）。
	if err := addRows("service", `SELECT DISTINCT seat FROM ops_event
		WHERE ts>now()-make_interval(days=>$1)`, depblindSignalDays); err != nil {
		return nil, fmt.Errorf("service 面枚举: %w", err)
	}
	if err := addRows("table", `SELECT table_name FROM information_schema.tables
		WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY 1`); err != nil {
		return nil, fmt.Errorf("table 面枚举: %w", err)
	}
	for _, p := range pairs {
		out = append(out, faceRow{"dep_pair", p.sig})
	}
	if err := addRows("template", `SELECT DISTINCT template FROM fin_opp_eval
		UNION SELECT DISTINCT template FROM fin_opp_template ORDER BY 1`); err != nil {
		return nil, fmt.Errorf("template 面枚举: %w", err)
	}
	if err := addRows("gate_domain", `SELECT DISTINCT split_part(key,'.',1)
		FROM app_config ORDER BY 1`); err != nil {
		return nil, fmt.Errorf("gate_domain 面枚举: %w", err)
	}
	return out, nil
}

// probeFaceDiff 面清单快照 diff 一拍：整面落批→上拍批差集=新面→逐条
// 新面行+升件（24h 同窗去重）→旧批清册仅留最近两拍。首拍基线零升件。
func (s *Sensor) probeFaceDiff(ctx context.Context, pairs []depPair) error {
	faces, err := s.faceInventory(ctx, pairs)
	if err != nil {
		return err
	}
	if len(faces) == 0 {
		return fmt.Errorf("面清单枚举空集——fail-closed（列不出面=枚举面病灶）")
	}
	var batch int64
	if err := s.pool.QueryRow(ctx,
		`SELECT nextval('ops_face_batch_seq')`).Scan(&batch); err != nil {
		return fmt.Errorf("批号派号: %w", err)
	}
	var prev *int64
	if err := s.pool.QueryRow(ctx,
		`SELECT max(batch) FROM ops_face_snapshot`).Scan(&prev); err != nil {
		return fmt.Errorf("上拍批读: %w", err)
	}
	prevSet := map[faceRow]bool{}
	if prev != nil {
		rows, err := s.pool.Query(ctx,
			`SELECT kind,face FROM ops_face_snapshot WHERE batch=$1`, *prev)
		if err != nil {
			return fmt.Errorf("上拍面集读: %w", err)
		}
		for rows.Next() {
			var r faceRow
			if err := rows.Scan(&r.kind, &r.face); err != nil {
				rows.Close()
				return fmt.Errorf("上拍面集读: %w", err)
			}
			prevSet[r] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("上拍面集读: %w", err)
		}
	}
	kinds := make([]string, 0, len(faces))
	faceNames := make([]string, 0, len(faces))
	var newFaces []faceRow
	for _, r := range faces {
		kinds = append(kinds, r.kind)
		faceNames = append(faceNames, r.face)
		if prev != nil && !prevSet[r] {
			newFaces = append(newFaces, r)
		}
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO ops_face_snapshot(batch,kind,face)
		SELECT $1,unnest($2::text[]),unnest($3::text[])`, batch, kinds, faceNames); err != nil {
		return fmt.Errorf("面清单落批: %w", err)
	}
	// 旧批清册——仅留最近两拍批（上拍批=本拍 diff 源，再旧者无供差义）。
	if prev != nil {
		if _, err := s.pool.Exec(ctx,
			`DELETE FROM ops_face_snapshot WHERE batch<$1`, *prev); err != nil {
			return fmt.Errorf("旧批清册: %w", err)
		}
	}
	sort.Slice(newFaces, func(i, j int) bool {
		if newFaces[i].kind != newFaces[j].kind {
			return newFaces[i].kind < newFaces[j].kind
		}
		return newFaces[i].face < newFaces[j].face
	})
	for _, r := range newFaces {
		if err := s.reportNewFace(ctx, r); err != nil {
			return fmt.Errorf("新面 %s:%s 落行: %w", r.kind, r.face, err)
		}
	}
	return nil
}

// faceBaselineExempt 基线册豁免判定（depBlindExempt 同式——精确面键
// 优先，'*' 尾前缀通配次；理由键随新面行留账）。
func faceBaselineExempt(scope string) (string, bool) {
	if r, ok := faceBaselineWhitelist[scope]; ok {
		return r, true
	}
	for k, r := range faceBaselineWhitelist {
		if strings.HasSuffix(k, "*") && strings.HasPrefix(scope, k[:len(k)-1]) {
			return r, true
		}
	}
	return "", false
}

// reportNewFace 单新面落账：「面清单新面」行+观测盲升件（esc_scope=
// face:<kind>:<face>，同窗 24h 至多一报——depblind 同款去重口径：
// 新面持续新于基线批只报一次，基线批前移后仍同窗吸）。基线册内面
// 新面行照落携理由键但升件免——人工过基线合法面窗摆不重复炸票。
func (s *Sensor) reportNewFace(ctx context.Context, r faceRow) error {
	scope := fmt.Sprintf("face:%s:%s", r.kind, r.face)
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		depblindEscClass, scope, depblindDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("同窗去重读败: %w", err)
	}
	if dup > 0 {
		return nil
	}
	reason, exempt := faceBaselineExempt(scope)
	note := fmt.Sprintf("kind=face_diff key=%s kind=%s face=%s", scope, r.kind, r.face)
	if exempt {
		note += " 基线册=" + reason
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'面清单新面',$2,$3)`,
		hungerSeat, faceTicket, note); err != nil {
		return fmt.Errorf("新面行: %w", err)
	}
	if exempt {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, faceTicket,
		fmt.Sprintf("面清单新面 %s：机器可列面 %s 类新增 %s——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）",
			scope, r.kind, r.face),
		depblindEscClass, scope); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}
