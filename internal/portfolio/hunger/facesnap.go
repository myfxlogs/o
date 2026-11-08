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
	// T-0450：user_intent 晋级请求面——fin_sub_request open 行在册即面
	// （depenum SubRequestArm 枚举本体零硬编码），请求闭/开摆动=间歇面形
	// （T-0448 册语义罩位），册外窗摆回重复升件炸起草拍（T-0393 链）。
	"face:dep_pair:subreq:user_intent:TRX-USDT-SWAP": "T-0450 入册——user_intent 晋级请求面：fin_sub_request open 行在册即面（SubRequestArm 枚举本体），请求闭/开=间歇面窗摆形；生产亲证 open 行 @2026-11-07 12:43Z（ops_event #3408 升件源）",
	// T-0453：ZEC 现货腿面——fin_strategy funding-hedge probe 行派生
	// （state<>'retired' 落 FundingHedgeSpotArm 谓词=ActiveSetUnion 枚举
	// 本体），提案/策略活性期进出=间歇面形（T-0448 册语义罩位），册外
	// 窗摆回重复升件炸起草拍（T-0393 链）。
	"face:dep_pair:leg:ZEC-USDT": "T-0453 入册——ZEC 现货腿面：fin_strategy funding-hedge probe 行（state<>'retired'）落 FundingHedgeSpotArm 谓词派生，提案/策略活性期进出=间歇面窗摆形；生产亲证 probe 行 @2026-11-08 00:13Z + b9/b10 快照批在册（ops_event #3454 升件源）",
	// T-0454：ZEC-USDT-SWAP 本键腿面——提案 fin_proposed/fin_armed 窗在册
	// （ActiveSetUnion 提案臂枚举本体），退役/probe 态期摆出、活性期摆回
	// =纯间歇面形（T-0448 册语义罩位），册外窗摆回重复升件炸起草拍
	// （T-0393 链）。
	"face:dep_pair:leg:ZEC-USDT-SWAP": "T-0454 入册——ZEC-USDT-SWAP 本键腿面：提案 fin_proposed/fin_armed 窗在册（提案臂枚举本体），退役/probe 态期摆出活性期摆回=纯间歇面窗摆形；生产亲证提案 2026-11-07 16:00Z 建/20:00Z retired（b9 在册→b10 摆出，ops_event #3456 升件源）",
	// T-0491：BTC-USD-270326 定日腿面——fin_proposal basis 族
	// （basis_quarter/perp_quarter_spread）fin_armed 窗在册（提案臂枚举
	// 本体零硬编码；legset.IsUSDTInst 定日形 -\d{6}$ 尾豁免=basis 域钦定
	// 数据源），armed_ttl_expired 周期 ~2h 间隙摆出摆回=间歇面窗摆形
	// （T-0448 册语义罩位），册外窗摆回重复升件炸起草拍（T-0393 链）。
	// CEO 裁 #4058：定日BTC腿新面合法性归草案基线审=本件。
	"face:dep_pair:leg:BTC-USD-270326": "T-0491 入册——BTC-USD-270326 定日腿面：basis 族提案 fin_armed 窗在册（ActiveSetUnion 提案臂枚举本体），armed_ttl_expired 周期间隙摆出摆回=间歇面形；生产亲证 52 提案循环 @2026-10-30 起+census 摆动 #3498 缺→#4035 回（ops_event #4039 升件源，CEO 裁 #4058 归本册基线审）",
	// T-0449：TRX-USDT-SWAP 本键腿面——fin_sub_request open 行经
	// SubRequestArm 入 ActiveSetUnion（depenum 枚举本体零硬编码；与
	// T-0450 册内 subreq:user_intent:TRX-USDT-SWAP 同一 open 行
	// user_intent/quote_watchlist），请求闭/开摆动=间歇面窗摆形
	// （T-0448 册语义罩位），册外窗摆回重复升件炸起草拍（T-0393 链）。
	"face:dep_pair:leg:TRX-USDT-SWAP": "T-0449 入册——TRX-USDT-SWAP 本键腿面：fin_sub_request open 行经 SubRequestArm 入 ActiveSetUnion（枚举本体零硬编码，同 T-0450 subreq 面同一 open 行 user_intent/quote_watchlist @2026-11-07 12:43Z 生产亲证仍 open），请求闭/开=间歇面窗摆形；生产亲证 b13/b14 快照批在册（ops_event #3406 升件源）",
	// T-0447 depSignalFamilies 钉定消费册登记对——合法新面（basis_ann→
	// basis 族活性对账在册，非「该建未建」残余）。
	"face:dep_pair:signal:basis_ann→quarter_basis_long":  "T-0447 basis_ann 钉定消费册登记对——depSignalFamilies 族消费补位合法新面（活性对账在册）",
	"face:dep_pair:signal:basis_ann→perp_quarter_spread": "T-0447 basis_ann 钉定消费册登记对——depSignalFamilies 族消费补位合法新面（活性对账在册）",
	// T-0471：CEO 无头班次席面——管理席席位（T-0456 业主令「CEO走无头」
	// 建席+T-0465 北极星账钦定面）非幽灵；ops_event 写径=ceo_tick.sh
	// O_SEAT=CEO→report.sh 并轨+ceo_northstar.sh 北极星账/议题行直插
	// （生产实证 2026-11-08 首行 00:25Z 起 169 行/日，ops_event #3925
	// 升件源）；班次驱动间歇写（kick+30min 兜底拍，队空=心跳即退零
	// ops_event 写）=间歇面形（T-0448 册语义罩位），静默 >7d 出集再
	// 活跃=再现新面再升件再起草循环（T-0393 链）。
	"face:service:CEO": "T-0471 入册——CEO 无头班次席（管理席，T-0456 业主令建席）：ceo_tick.sh O_SEAT=CEO→report.sh+ceo_northstar.sh 北极星账/议题行双写径合法，班次驱动间歇写 7d 活跃窗摆动登记不重复升；生产亲证 2026-11-08 首行 00:25Z 起 169 行/日（ops_event #3925 升件源）",
	// T-0472：COS上行=秘书席上行写径席位身份——O-DM pane 业主在席处置窗
	// ops_event 写径（呈批/移交/领号/升件/挂起/收口待审等；T-0456 双无头
	// 席建置 ceo_queue.sh 钦定席位集含 'COS上行'——席上行件归 CEO 方向层
	// 设计在案）；业主在才有秘书之义=间歇写面形（T-0448 册语义罩位），
	// 静默 >7d 出集再活跃=再现新面再升件再起草循环（T-0393 链）。
	"face:service:COS上行": "T-0472 入册——COS上行=COS 秘书席上行写径席位身份（O-DM pane 业主在席处置窗 ops_event 写径呈批/移交/领号/升件等；ceo_queue.sh T-0456 钦定席位集含 'COS上行' 席上行件归 CEO 方向层设计在案）；业主在才有秘书之义=间歇写面 7d 活跃窗摆动登记不重复升；生产亲证 2026-11-07 23:05Z→11-08 01:07Z 26 行五类事件（ops_event #3927 升件源）",
	// T-0474：席O-COS 面——COS 秘书席（业主令 2026-11-04 立席，纯交互席
	// O-COS tmux 会话）在会话内 report.sh tmux 回退径自报（无 O_SEAT、
	// 映射表无 席O-COS→COS 条目→seat='席O-COS'）；ops_event 写径=
	// report.sh 并轨合法（移交/审计中/外派/呈批/收口/裁决/通过/处置/
	// 派单/完成十类事件，生产实证 2026-11-08 00:18Z 首行 49 行，
	// ops_event #3931 升件源）；交互席队空零写=间歇面形（T-0448 册语义
	// 罩位），静默 >7d 出集再活跃=再现新面再升件再起草循环（T-0393
	// 链，T-0471 CEO 同型先例）。
	"face:service:席O-COS": "T-0474 入册——席O-COS=COS 秘书席交互席位（业主令 2026-11-04）：O-COS tmux 会话 report.sh 回退径自报（无 O_SEAT/映射表无 席O-COS→COS 条目），ops_event 写径合法；队空零写间歇面窗摆登记不重复升；生产亲证 2026-11-08 00:18Z 首行 49 行（ops_event #3931 升件源）",
	// T-0490：config:meta→DM 机批闸册 配对面——meta app_config 键域
	// =fin_0143 迁移种 meta.autoapprove.registry（T-0455 册本体，签名→
	// 钦定文件径白名单）在册即面；读者=depConfigReaders["meta"] 册载
	// 「DM 机批闸册」（T-0448 补册——dm_autoapprove.sh 实读同键=候选
	// 票批权径，本票自身 #4373 机批实证）；枚举对=depenum ④族键域×
	// 读面册本体零硬编码，域在册恒产非产消断——册外/重种回摆重复
	// 升件炸起草拍（T-0393 链，升件源 #4037）。
	"face:dep_pair:config:meta→DM 机批闸册": "T-0490 入册——meta 键域=fin_0143 种 meta.autoapprove.registry（T-0455 机批册本体）在册即面，读者册载 DM 机批闸册（dm_autoapprove.sh 实读同键，批权径在案）；枚举对 depConfigReaders 册内合法新面登记不重复升；生产亲证 app_config meta.autoapprove.registry 在册+升件源 ops_event #4037（机批已批 #4373）",
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
