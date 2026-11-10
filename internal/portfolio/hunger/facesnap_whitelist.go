// facesnap_whitelist.go —— 面清单人工基线册（T-0494 返修1 拆件：
// faceBaselineWhitelist 册表自 facesnap.go 拆出同包新文件——check-lines
// 棘轮断根（facesnap.go 已贴 300 线，册表续增必撞）；package hunger 顶层
// var 拆文件编译等价，消费面 faceBaselineExempt/reportNewFace 原位不动）。
package hunger

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
	// T-0473：外审席=外派件归宗裁决道席位——业主令 2026-11-07「外派外审
	// 外部署，不去给DM做积压」成例（docs/纪元2归档/席位卡/外审席/SKILL.md 职卡在案；
	// ceo_queue.sh 钦定席位集含 '外审席'——席上行件非探针自产设计在案）；
	// ops_event 写径=归宗裁决/通过自报合法；班次形=瞬态 systemd-run
	// o-review-ext 扫到有件才拉起队列空即退无件不常驻=间歇写面形（T-0448
	// 册语义罩位），静默 >7d 出集再活跃=再现新面再升件再起草循环（T-0393
	// 链）。
	"face:service:外审席": "T-0473 入册——外审席=外派件归宗裁决道席位（业主令 2026-11-07 成例，职卡+ceo_queue.sh 钦定席位集在案）：瞬态 systemd-run o-review-ext 有件才拉起无件不常驻，归宗裁决/通过 ops_event 自报径合法；间歇写面 7d 活跃窗摆动登记不重复升；生产亲证 2026-11-08 02:36Z→04:55Z 9 行五件归宗裁决（ops_event #3929 升件源，CEO 裁 #3946 合法性终态在案）",
	// T-0490：config:meta→DM 机批闸册 配对面——meta app_config 键域
	// =fin_0143 迁移种 meta.autoapprove.registry（T-0455 册本体，签名→
	// 钦定文件径白名单）在册即面；读者=depConfigReaders["meta"] 册载
	// 「DM 机批闸册」（T-0448 补册——dm_autoapprove.sh 实读同键=候选
	// 票批权径，本票自身 #4373 机批实证）；枚举对=depenum ④族键域×
	// 读面册本体零硬编码，域在册恒产非产消断——册外/重种回摆重复
	// 升件炸起草拍（T-0393 链，升件源 #4037）。
	"face:dep_pair:config:meta→DM 机批闸册": "T-0490 入册——meta 键域=fin_0143 种 meta.autoapprove.registry（T-0455 机批册本体）在册即面，读者册载 DM 机批闸册（dm_autoapprove.sh 实读同键，批权径在案）；枚举对 depConfigReaders 册内合法新面登记不重复升；生产亲证 app_config meta.autoapprove.registry 在册+升件源 ops_event #4037（机批已批 #4373）",
	// T-0492：gate_domain:meta 面——meta app_config 键域=fin_0143 迁移种
	// meta.autoapprove.registry（T-0455 机批册本体在册即面；读者=
	// depConfigReaders["meta"] 册载「DM 机批闸册」T-0448 补册，
	// dm_autoapprove.sh 实读同键=候选票批权径本票自身 #4373 机批实证）；
	// gate_domain 枚举=split_part(key,'.',1) 本体零硬编码，域在册恒产
	// 非产消断——重种/库重建回摆重复升件炸起草拍（T-0393 链，升件源
	// #4043；姊妹面 dep_pair:config:meta→DM 机批闸册=T-0490 同根另册）。
	"face:gate_domain:meta": "T-0492 入册——meta 键域=fin_0143 种 meta.autoapprove.registry（T-0455 机批册本体）在册即面，读者册载 DM 机批闸册（dm_autoapprove.sh 实读同键批权径在案，写域册外 fail-closed 归 owner=类册面钦定判据设计在案）；枚举谓词 split_part 本体零硬编码合法新面登记不重复升；生产亲证 app_config meta.autoapprove.registry 在册+升件源 ops_event #4043（快照批 14/15 在册）",
	// T-0493：ops_seat_card 表面——fin_0145 迁移建 public 基表（T-0458
	// 飞书席位活卡：seat→message_id+chat_id 卡册持久映射，展示面非真源
	// 真源=ops_event 链账）在册即面；写径=seatcard 服务 FIN_PG_DSN 同
	// roster_rw 写册角色（internal/ops/seatcard+cmd/seatcard 在册，
	// fin_0121 同族授权）；table 类面=information_schema 全量 BASE TABLE
	// 枚举本体零硬编码——建表落地即面，表重建/快照册重置回摆重复升件
	// 炸起草拍（T-0393 链，升件源 #4047，机批已批 #4376）。
	"face:table:ops_seat_card": "T-0493 入册——ops_seat_card=fin_0145 迁移建 public 基表（T-0458 飞书席位活卡卡册：seat→message_id 持久映射，展示面非真源真源=ops_event）；seatcard 服务 FIN_PG_DSN 写径合法在册；table 类面=information_schema BASE TABLE 枚举本体零硬编码，建表落地即面；生产亲证 public 基表在册+b14/b15 快照批在册（ops_event #4047 升件源，机批已批 #4376）",
	// T-0496：MECH=GW-001 内循环机械轨席位（业主令 2026-11-08 钦定，
	// docs/infra/GW-001 在案）——ops_event 写径=drift_probe/drift_converge
	// （漂移/漂移消/升件）+verdict_reflow/case_writeback（归流）机械轨自报
	// 合法，o-mech.timer+事件帧驱动机械活批处理间歇写面形（T-0448 册语义
	// 罩位），静默 >7d 出集再活跃=再现新面再升件再起草循环（T-0393 链）。
	"face:service:MECH": "T-0496 入册——MECH=GW-001 内循环机械轨席位（业主令 2026-11-08 钦定，docs/infra/GW-001 在案）：drift_probe/drift_converge 漂移消+verdict_reflow/case_writeback 归流 ops_event 写径合法，o-mech.timer+事件帧驱动机械活批处理间歇写 7d 活跃窗摆动登记不重复升；生产亲证 2026-11-08 16:02Z 首行 92 行四类事件+ops_headless 心跳 alive（ops_event #4756 新面行→#4757 升件源，机批已批 #4783）",
	// T-0494：BTC-USDT-SWAP 本键腿面——fin_strategy state IN
	// ('live','degraded') 臂在册（ActiveSetUnion 策略臂枚举本体零硬编码；
	// depLegKeys ∩IsUSDTInst∩宇宙闸）——生产亲证 simdrill matrix 产跑
	// ephemeral 格行（T-0480 @2026-11-08 09:35:53Z，template_ref=
	// 'simdrill:cell:xvenue-carry@mt5xokx/*#usd=50'，state='degraded'
	// 哨戒降级联动设计态，fin_exec_drill 凭据 5/6 pass 在案）入集；
	// degraded_ttl_hours 默认 24h 自退役（retireStaleDegraded T-0098）
	// →出集摆回+下次产跑/真策略活性期摆回=间歇面窗摆形（T-0448 册语义
	// 罩位，T-0449/0453/0454/0491 同族先例），册外窗摆回重复升件炸
	// 起草拍（T-0393 链，升件源 #4467）。
	"face:dep_pair:leg:BTC-USDT-SWAP": "T-0494 入册——BTC-USDT-SWAP 本键腿面：fin_strategy live/degraded 臂在册（ActiveSetUnion 枚举本体），生产亲证 simdrill matrix ephemeral 格行 T-0480 @2026-11-08 09:35:53Z（degraded 设计态+24h TTL 自退役=间歇面窗摆形）；乐器=OKX 主流永续订阅宇宙内实键，腿覆盖供给无虞；册外窗摆回重复升件炸起草拍断（升件源 ops_event #4467）",
	// T-0497：md_eval_pulse 表面——fin_0148 迁移建 public 基表（T-0485
	// eval 活动脉冲桥：md fin_opp_eval per-template 聚合派生，projector
	// serve 拍尾+活性对账写 SG；domain.tsv bridge 入册，fin_0147→0148
	// 改号在案；SG 侧表+projector_rw SIUD/o_fin_ro SELECT 授权 T-0488
	// 直放对拍在案）在册即面；写径=projector refreshEvalPulse upsert
	// （internal/projector/pulse.go projector_rw 授权在案），读径=
	// hunger 族沉默探针 pulseTemplates+flow_pulse eval 段灯（o_fin_ro
	// SELECT 授权在案）；table 类面=information_schema BASE TABLE 枚举
	// 本体零硬编码——建表落地即面，表重建/快照册重置回摆重复升件炸起草
	// 拍（T-0393 链，升件源 #4759/新面行 #4758/快照批 14/15 在册；CEO
	// 终态裁 #4768 明载=部署落地合法新面，机批已批 #4784）。
	"face:table:md_eval_pulse": "T-0497 入册——md_eval_pulse=fin_0148 迁移建 public 基表（T-0485 eval 活动脉冲桥 per-template 聚合派生，domain.tsv bridge 入册；SG 表+授权 T-0488 直放对拍在案）；写径 projector refreshEvalPulse/读径 族沉默探针+eval 段灯授权在案；table 类面 information_schema BASE TABLE 枚举本体零硬编码，建表落地即面；生产亲证 public 基表在册+快照批 14/15 在册（ops_event #4759 升件源，CEO 终态裁 #4768 合法新面，机批已批 #4784）",
	// T-0503：TEST=键卫生闸实弹验证标记席——T-0499 ticket键卫生根修
	// 施工 run（ops_event #4990 领单→#5078 驳回返修1 在案）对
	// ops_event_key_gate.sh 伪键拒写径做生产实测，第三参 SEAT 显式
	// 传 'TEST' 标记件（伴生 #5005-#5008 同 burst 四连 MECH 默认席
	// 拒写行 @2026-11-08 20:05:33-39Z，ctx=event=领单 key=T-9001\nT-9002
	// h=95204335）——非在册席位=测试件产物面非幽灵席；service 面=
	// ops_event 近 7d DISTINCT seat 活跃投影（写者席即服务面投影，
	// facesnap.go 设计口径）入集→上拍批册外→面清单新面行 #5222→
	// 观测盲升件 #5223→draft#42→本票（机批已批 #5252）；TEST 行龄
	// 出 7d 活跃窗摆出，任一键闸复测再写 seat='TEST' 即再现新面
	// 再升件再起草循环（T-0393 链，T-0448 册语义罩位，T-0496 留案
	// face:service:--take 同族键卫生测试产物面先例在案）。
	"face:service:TEST": "T-0503 入册——TEST=T-0499 键卫生闸实弹验证标记席（ops_event_key_gate.sh SEAT 参显式传值，施工 run #4990-#5078 在案）：升件行 #5004 单发+伴生 #5005-#5008 四连拒写 @2026-11-08 20:05Z 测试件产物面非幽灵席；service 面 7d 活跃窗摆出摆回登记不重复升；生产亲证升件源 ops_event #5223（面清单新面行 #5222，机批已批 #5252）",
	// T-0505：研究员哨兵=研究员触发消费拍席位（T-0504 方案A CTO 钦定
	// ——scripts/meta/researcher_tick.sh 滤队消费 venue哨兵 新品/达阈
	// 触发→瞬态拉起 o-researcher-evt-*→rc=0 逐事件落 消行
	// seat='研究员哨兵' note='触发#<id> 已拉起 unit=…'+deploy/
	// o-researcher-tick.service 部署件在案）——ops_event 写径=触发
	// 消费落账合法非幽灵席；service 面=ops_event 近 7d DISTINCT seat
	// 活跃投影（写者席即服务面投影，facesnap.go 设计口径）入集→上拍
	// 批册外→面清单新面行→观测盲升件；消费拍间歇写（pending 空拍=
	// hb ops_headless 零 ops_event 写，venue_watch 空拍同构）静默
	// >7d 出集再活跃=再现新面再升件再起草循环（T-0393 链，T-0448
	// 册语义罩位，T-0496 MECH/T-0503 TEST 同型先例）。
	"face:service:研究员哨兵": "T-0505 入册——研究员哨兵=研究员触发消费拍席位（T-0504 方案A CTO 钦定，researcher_tick.sh 消费落账+o-researcher-tick 部署件在案）：venue哨兵 新品/达阈 触发滤队消费落 ops_event 消行 seat='研究员哨兵' 写径合法；消费拍间歇写（空拍零 ops_event 写）7d 活跃窗摆动登记不重复升；生产亲证 2026-11-08 22:57:46Z 首行 2 行 T-0504 收口合成演练清场消行（ops_event #5345/#5346→面清单新面行 #5356→升件 #5357，CEO 机批已批 #5368）",
	// ===== 纪元3 D-0045 五环席位批（2026-11-09 纪元3 部署落地合法新面
	// 成批入册——探测薄壳/唤知机/处置席/复审席/改进矿工/唤知实证=纪元3
	// 钦定五环写者席，ops_event 写径合法；service 面 7d 活跃窗摆动登记
	// 不重复升；生产亲证 2026-11-09 首拍面清单新面行/升件在案）。
	"face:service:探测席":                "纪元3 入册——探测席=hunger 探针面写者（面清单新面/依赖对清单/升件 写径合法，D-0045 感知环本体）；生产亲证 2026-11-09 面清单新面行 #8895→升件 #8896",
	"face:service:唤知机":                "纪元3 入册——唤知机=o-cognition.timer 唤知拍写者（唤起 行×5 在案，D-0045 认知环触发本体）；生产亲证 2026-11-09 面清单新面行 #8888→升件 #8889",
	"face:service:唤知实证":               "纪元3 入册——唤知实证=唤知环全环实证探针写者（升件 1 件在案）；生产亲证 2026-11-09 面清单新面行 #8886→升件 #8887",
	"face:service:审计机":                "纪元3 入册——审计机=o-audit.timer/o-audit-dispatch 审计臂写者（D-0046 不等信号主动扫不变量；run.sh COG_ROLE=audit→SEAT_MACHINE=审计机，唤起/升件写径合法）；生产亲证 2026-11-10 07:50Z 唤起行 #9307（unit=o-audit-1794297013 scope=探测面）→升件 #9323",
	"face:service:复审席":                "纪元3 入册——复审席=纪元3 复审写者（处置/裁决 写在案）；生产亲证 2026-11-09 面清单新面行 #8890→升件 #8891",
	"face:service:接市官":                "纪元3 入册——接市官=venue接入拍写者（T-0379 venue接入beat mt5/okx 配对枚举零命中如实记 写径合法）；生产亲证 2026-11-10 12:39Z venue接入行 #9494/#9495（key=mt5:3001157）→升件 #9497",
	"face:service:值守":                 "纪元3 入册——值守=live_dud 哑弹探针写席（sigfix 并入值守臂：fin_strategy live≥7d 零仓位→升件 beat 写径合法）；生产亲证 2026-11-10 12:40Z 集量行 #9500（live哑弹29/31 funding-hedge 自报）→升件 #9514",
	"face:service:research/dcd":       "纪元3 入册——research/dcd=o-research dcd 枚举臂写席（DCD 产品枚举 beat+连败自停/复役条件书升件自报 写径合法；fin_0162 授 review_rw ops_headless 权修通事务径）；生产亲证 2026-11-10 12:56Z 自停升件行 #9519 自落→升件 #9530",
	"face:service:改进矿工":               "纪元3 入册——改进矿工=hunger 教训矿工写者（升件提案 ×3 在案 R1/R2/R3）；生产亲证 2026-11-09 面清单新面行 #8935→升件 #8936",
	"face:service:处置席":                "纪元3 入册——处置席=纪元3 处置环写者（处置 920+ 行 2026-11-09 在案，信号处置落账本体）；生产亲证 面清单新面行 #6438→升件 #6439",
	"face:service:gateway":            "纪元3 入册——gateway=meta 网关席（升件路由40/心跳10/收敛9/席权2 写在案——纪元3 无头派发基建合法写面）；生产亲证 面清单新面行 #6436→升件 #6437",
	"face:service:hostmetrics":        "纪元3 入册——hostmetrics=宿主机量表探针写者（feed断更+升件 写在案——采集观测源宿主机面合法）；生产亲证 面清单新面行 #7703→升件 #7704",
	"face:table:ops_event_sem_matrix": "纪元3 入册——ops_event_sem_matrix=fin_0154 迁移建表（T-0506 ops_event (seat,event) 合法性矩阵白名单册，domain.tsv bridge 入册行在案）；建表落地即面；生产亲证 面清单新面行 #5759→升件 #5760",
	// ===== 纪元3 会话补册（2026-11-09 唤知会话处置面——同批语义）=====
	// repair 键域=app_config repair.feeds/repair.guard（采集臂修复注册表+
	// 闸参数，internal/repair/watch.go 实读）——depConfigReaders 册载
	// 「采集臂修复环」补位后枚举对=合法册内面；gate_domain 面=键域谱面
	// 本体零硬编码，域在册恒产非产消断（T-0490/0492 meta 同型先例）。
	"face:dep_pair:config:repair→采集臂修复环": "2026-11-09 入册——repair 键域=app_config repair.feeds/repair.guard 在册即面，读者册载采集臂修复环（internal/repair/watch.go cfg.Get 实读两键，修复环配额闸+md_repair_log 落账径在案）；depConfigReaders 补册后枚举对合法新面登记不重复升；生产亲证 app_config 两键在册+升件源 ops_event #9011",
	"face:gate_domain:repair":            "2026-11-09 入册——repair app_config 键域=采集臂修复注册表/闸参数在册即面（写域册 dm 类基建自愈面，读者=采集臂修复环实读在案）；gate_domain 枚举=split_part 本体零硬编码合法新面登记不重复升；生产亲证 app_config repair.feeds+repair.guard 在册+升件源 ops_event #9015",
	// 研究员=能力面扫描机件席位（D-0045①½立项产出——capability scan
	// 假想卡 ops_event 写径合法）；扫描拍间歇写（信号面净+7d 无扫描才
	// 触发）静默 >7d 出集再活跃=再现新面再升件再起草循环（T-0393 链，
	// T-0448 册语义罩位，纪元3 批同型先例）。
	"face:service:研究员": "2026-11-09 入册——研究员=能力面扫描机件席位（D-0045①½立项，唤知会话代行产出 cap-scan+hypo 卡 ops_event 写径合法）：扫描拍间歇写 7d 活跃窗摆动登记不重复升；生产亲证 2026-11-09 cap-scan-2026-11-09+三假想卡行 #8975-8976（面清单新面行→升件 #8990）",
	// 决策席-DM写径=T-0376 DM自治写径席位（internal/platform/config/
	// dmwrite.go dmActor——业主令「如何去赚」全参数域归 DMSet 正径）：
	// friction_bps 实绩回灌等参数调整账行按校准拍间歇写，静默 >7d
	// 出窗摆回=间歇面形（T-0448 册语义罩位），登记不重复升。
	"face:service:决策席-DM写径": "2026-11-10 入册——决策席-DM写径=T-0376 DM自治写径席位（dmwrite.go dmActor，业主令「如何去赚」参数域归 DMSet 正径）：friction_bps 实绩回灌等参数调整行按校准拍间歇写 7d 活跃窗摆动登记不重复升；生产亲证 #9090 lifecycle.friction_bps.okx=7.317 30d 实绩回灌行（面清单新面行→升件 #9092）",
	"face:service:审计席":      "2026-11-10 入册——审计席=D-0046 审计臂裁决写席（audit 会话处置裁词/复证行 ops_event 写径合法——audit_stall/spawn_fail 自体活性探针同席）：间歇写=审计拍驱动 7d 活跃窗摆动登记不重复升；生产亲证 2026-11-10 审计升件处置批 #9572-9577 裁决行（面清单新面行→升件 #9593）",
}
