# T-0497 探针升件自动起草：观测盲@face:table:md_eval_pulse

- 状态：收口待审（施工收口——回决策席常径裁）
- 领取：devin-o-worker-T-0497 @ 2026-11-09
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:table:md_eval_pulse`，草案锚 fin_ticket_draft#40）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:table:md_eval_pulse`（kind=探针类 `观测盲`，key=病灶键 `face:table:md_eval_pulse`）

末行升件原文：
> 面清单新面 face:table:md_eval_pulse：机器可列面 table 类新增 md_eval_pulse——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#4759

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:table:md_eval_pulse' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:table:md_eval_pulse（T-0455 册内候选，DM 裁时可改判）；
2. 病灶根因定位与根修（签名面病灶本体——不超签名外衍）；
3. 桩测钉死断言（病灶存在断言修前红+同断言修后绿——T-0388 修复类先红桩律）；
4. 生产谱面零误报审计。

## 验收锚（骨架——DM 裁时补足）

① 桩测断言全绿（红/绿双点证据落结果回填）；
② 探针同窗不再报同签名病灶（修复后探针行随停）；
③ check_all 绿戳对口；并入发窗批。

## 红线（继承模板——不改不删）

- 只修本票签名病灶——不超单施工、不碰范围外文件；
- 不部署、不并 master——收口回决策席常径裁；
- 探针只报不修——本票修病灶本体，禁顺手改探针阈/签名面/类册；
- 钱径/凭据域/真单径语义禁碰。

## 结果回填

### 根因（生产取证 2026-11-09）

- 新面本体=`md_eval_pulse` public BASE TABLE——fin_0148 迁移建（T-0485
  eval 活动脉冲桥：md 正本 fin_opp_eval per-template 聚合派生
  max(ts)/count(24h)，projector serve 拍尾+活性对账写 SG；domain.tsv
  bridge 入册；原 fin_0147 与 GW-001 fin_0147_ops_ticket_phase 撞号改
  0148 在案——CTO 驳回行 #4528/返修件在案）。SG 侧表+授权=T-0488 直放
  对拍在案（ops_event #4511：projector_rw SIUD+o_fin_ro SELECT 齐）。
- table 类面枚举=`information_schema.tables WHERE table_schema='public'
  AND table_type='BASE TABLE'` 全量本体零硬编码（facesnap.go
  faceInventory）——建表落地即面，无「该建未建」残余语义。
- 合法性核=**合法新面**：CEO 终态裁 #4768 明载「面清单新面二件终态——
  table:md_eval_pulse(T-0485 eval脉冲桥落地表)+service:MECH=部署落地
  合法新面」（#4770 同裁摘要在案）；写径=projector `refreshEvalPulse`
  upsert（internal/projector/pulse.go——md 侧聚合→SG 同键 upsert，
  projector_rw 授权在案）；读径=hunger 族沉默探针 `pulseTemplates`
  （silence.go）+`cmd/core/flow_pulse.go` eval 段灯（o_fin_ro SELECT
  授权在案，生产直查 role_table_grants 在案）。
- 生产亲证（FIN_PG_RO_DSN=o_fin_ro 直查 SG o 库）：`md_eval_pulse`
  public BASE TABLE 在册；表内 13 模板行 `refreshed_at`=
  2026-11-08 21:09:30Z 秒级新鲜（projector 活写在案）；
  `ops_face_snapshot` 批 14/15 在册（max batch=15）；o_fin_ro SELECT
  授权直查在案。
- 升件源链=#4758「面清单新面」行→#4759 升件（票面证据引）→#4765
  draft#40→#4768/#4770 CEO 终态裁→#4782 自动起草落件→#4784 机批已批
  →#4909 dedup skip→本票（领单 #5140/#5141 在案；#4795 派单→#4796 候水
  →#5139 漏拉补派续跑）。
- 病灶本体=T-0448 同型族：合法新面**无持久基线位**——面宇宙人工过
  基线后新增才值得看，但合法新增面需登记位续判，册外/表重建/快照
  册重置回摆重复升件炸起草拍（T-0393 链；dep_pair/service/table 先例
  T-0449/0450/0453/0454/0471/0472/0473/0474/0490/0491/0492/0493 同径）。
- 疑似根域勘误：票面起草猜域「评估×信号链面（fin_opp_eval×md_signal
  ——T-0384 族沉默探针域）」系 T-0393 模板猜域——病灶本体在 hunger
  面清单基线册（facesnap table 类枚举），非 eval/signal 链 bug；
  `md_eval_pulse` 表本体即 T-0485 观测域错位修复件产物（合法新面），
  sigdist 面零触不越域衍（同 T-0473/0490/0493 勘误先例口径）。

### 实现摘要（枝 t/0497-face-md-eval-pulse）

- `facesnap_whitelist.go`（返修1 拆件档——T-0494 册表整体迁入新档
  已并 master）`faceBaselineWhitelist` 册表登记
  `face:table:md_eval_pulse`（注册式单键精确匹配零通配——册内面差集
  照落「面清单新面」行携理由键留账但升件免；表重建/快照册重置回摆
  不重复炸票/起草，T-0393 链断点同 T-0448 族先例）。
- `depblind_test.go` 先红桩
  `TestFaceBaselineTableMdEvalPulseRegistered`（T-0388 修复类先红
  桩律：病灶存在断言修前红+同断言修后绿——签名面必须在册且理由键
  非空，注册式不静默）。
- 探针阈/签名面/类册机制本体零改动（红线③守——只修病灶本体）。

### 返修1（棘轮新超标驳回→册表落位新档）

- 驳回因（决策席裁决 #5171/#5172 在案）：首收 merge 撞
  check-lines 棘轮——主干含 T-0496 入册行后 facesnap.go=300，
  本件 +6=306>300 新超标硬阻断 merge 不可提交（T-0494 返修1 同判
  先例，其预言兄弟面撞线实证；死法类=码面）。
- 改法（返修1 件钦定径）：rebase 新 master（8b2202f4c——已含
  T-0494 册表拆件 `facesnap_whitelist.go`+T-0496 service:MECH
  入册行，本键直接落新档不再自拆）；depblind_test.go EOF 双追加
  冲突两断言并存解（union 消解同 T-0494 返修1 律）；facesnap.go
  取主干版零触（199 行）。
- 新双点（终态档面重取证）：红=`go test -run
  TestFaceBaselineTableMdEvalPulseRegistered`（新档册表尚无本键
  ——以 master 版 facesnap_whitelist.go 换入实证）→
  `depblind_test.go:925: 病灶存在：face:table:md_eval_pulse
  未入人工基线册——回摆将重复升件（T-0497）` FAIL；绿=同断言
  入册后 PASS（TestFaceBaseline* 十五桩同跑全绿）。首轮双点
  （depblind_test.go:880 FAIL→PASS，facesnap.go 册内档面）在案
  同断言本体不变。

### 测试证据

- 先红桩双点（终态档面）：`TestFaceBaselineTableMdEvalPulseRegistered`
  修前 FAIL「病灶存在：face:table:md_eval_pulse 未入人工基线册——
  回摆将重复升件（T-0497）」（depblind_test.go:925）→入册后同断言
  PASS；`TestFaceBaseline*` 十五先红桩（T-0449/0450/0453/0454/0471/
  0472/0473/0474/0490/0491/0492/0493/0494/0496+本票）同跑全绿。
  首轮双点（:880，册内档面）在案同断言本体不变。
- `go build`/`go vet`/`gofmt` 净；`go test
  ./internal/portfolio/hunger/` 全包绿（DB 桩测谱随
  O_TEST_PG_DSN 租约径——check_all scoped 绿戳对口见收口段）。
- 行数自证：facesnap.go=199（主干版零触）/facesnap_whitelist.go
  129 双双<300 棘轮✓；check_all 绿戳对口见收口段（rebase 后本盒
  盖戳——收口终盖恒本盒律）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——本票仅新增
  `face:table:md_eval_pulse` 一键；生产近 14d「观测盲 face:*」升件
  谱 29 签名逐键对拍：其余 28 键（service×16+dep_pair×8+table×3+
  gate_domain×1——其中已入册面系登记前历史升件行留账，新面行照落
  携「基线册=」理由键升件免机制在案）零豁免面照升件各归己票域裁决
  （MECH/判例闸/外审/演习/裁定席/起草拍/跑席/配管/--take/
  O-Dev交互席/业主 等——同批族 T-0494/0495/0496 已并 master 在案，
  本票不越权代判）。
- table 类同型面核查：information_schema 全量 BASE TABLE 枚举——
  `md_eval_pulse` 为本票签名唯一登记键；余 table 面
  （fin_exec_drill/fin_guard_suspend/fin_symbol_dialect）对应草案
  dismissed 在案未入册，回摆径见偏离/留案段，本票登记不触其升件径。
- 验收锚②「探针同窗不再报同签名病灶」：该面已入快照批 14/15
  （机制自愈——册=簿记基线位）；若表重建/快照册重置致面回摆，
  册内面新面行照落携「基线册=」理由键但升件免——同签名升件/起草
  链断（depblindDedupHours 同窗去重口径不变，探针阈/签名面/类册
  机制本体零改动）。

### 偏离/留案

- 兄弟票同批族已全部并 master——T-0494（dep_pair leg:BTC-USDT-SWAP，
  返修1 册表拆件先例=本件改法出处）/T-0495（signal:basis_ann）/
  T-0496（service:MECH——CEO #4768 同裁合法新面在案）各面合法性核+
  入册已落；本件=同族末件，facesnap_whitelist.go 册表档面续登
  （merge 冲突解=条目并集全保先例在案）。
- CEO #4768 终态裁在案：md_eval_pulse 表归「T-0485 eval脉冲桥落地
  表=部署落地合法新面」档——本票登记=该裁落册执行，非新判。
- 合法性锚=fin_0148 迁移册本体在案+projector 写径/o_fin_ro 读径
  授权在案+CEO #4768 终态裁+生产基表/快照批/活写三亲证（合法新面
  非幽灵面非断链面）。
- 范围外发现（不越权处置，留案呈裁量）：生产近 14d face:* 升件谱
  中 `face:table:fin_exec_drill`/`face:table:fin_guard_suspend`/
  `face:table:fin_symbol_dialect` 三 table 面对应草案
  fin_ticket_draft#8/#9/#10 现态=`dismissed`（在案处置非漏单），
  但三面均未入 `faceBaselineWhitelist`——表重建/快照册重置回摆将
  再报升件再起草（T-0393 链同型）；是否补入册归决策席/COS 裁量，
  本席不代判。
