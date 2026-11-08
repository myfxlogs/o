# T-0494 探针升件自动起草：观测盲@face:dep_pair:leg:BTC-USDT-SWAP

- 状态：收口待审（施工收口——回决策席常径裁）
- 领取：devin-o-worker-T-0494 @ 2026-11-09
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:dep_pair:leg:BTC-USDT-SWAP`，草案锚 fin_ticket_draft#33）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:dep_pair:leg:BTC-USDT-SWAP`（kind=探针类 `观测盲`，key=病灶键 `face:dep_pair:leg:BTC-USDT-SWAP`）

末行升件原文：
> 面清单新面 face:dep_pair:leg:BTC-USDT-SWAP：机器可列面 dep_pair 类新增 leg:BTC-USDT-SWAP——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#4467

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:dep_pair:leg:BTC-USDT-SWAP' ORDER BY id DESC LIMIT 20
```

## 疑似根域

行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:dep_pair:leg:BTC-USDT-SWAP（T-0455 册内候选，DM 裁时可改判）；
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

- 新面本体=dep_pair 面 `leg:BTC-USDT-SWAP`——枚举源=depenum
  `depLegKeys`：`SELECT DISTINCT inst_id FROM
  (legset.ActiveSetUnion) t` ∩ `legset.IsUSDTInst` ∩ 订阅宇宙
  （facesnap `faceInventory` kind=dep_pair 直投 `pairs[].sig`，
  本体零硬编码）；本键腿=策略臂 `fin_strategy state IN
  ('live','degraded')`（legset.go ActiveSetUnion 第二臂）产物——
  非三派生臂（Basis/DCD/FundingHedge 皆产现货键 -USDT/-USD 形）。
- 生产亲证（SG o-postgres 直读）：`fin_strategy` id=287290/287291
  inst_id=BTC-USDT-SWAP state=degraded
  template_ref='simdrill:cell:xvenue-carry@mt5xokx/
  {long_perp_short_cfd,short_perp_long_cfd}#usd=50' venue=mt5xokx
  entered/created=2026-11-08 09:35:53.7/.9Z——simdrill matrix 产跑
  ephemeral 格行（T-0480 收口在案；fin_exec_drill id=5/6
  verdict=pass rules_version=simvenue-matrix/1 同刻落账，矩阵六格
  同批 strategy_ref=287285/287287-287291 全谱）。全库 active（live/degraded）
  BTC-USDT-SWAP 策略恰此二行——提案臂/晋级臂同键零在册（亲查
  fin_proposal/fin_sub_request 无 active 行）。
- degraded=演习设计态（matrix.go:131 注记「state='degraded'
  哨戒降级联动必平径免判据链依赖」）；24h TTL 自退役=
  retireStaleDegraded（run.go:220——strategy.degraded_ttl_hours
  键缺默认 24h，T-0098 方案A）→~2026-11-09 09:35Z 出集摆回。
- 合法性核=**合法新面非「该建未建」残余**：枚举谓词零硬编码本体
  枚举 fin_strategy 实态；格行=设计内 ephemeral（template_ref
  simdrill: 前缀=excell 占用排除+级联清理同律在案）；乐器
  BTC-USDT-SWAP=OKX 主流永续订阅宇宙内实键（depLegKeys universe
  闸已过=供给侧在订阅），腿覆盖供给无虞（本键即采集面主力键，
  md_ticks 常新鲜——非 T-0445 幽灵键形）。
- 面形=间歇窗摆形（T-0448 册语义罩位）：TTL 退役出集→下次
  matrix 产跑/真策略活性期摆回；且 BTC-USDT-SWAP 真策略史=
  2026-10-24~10-28 连环 retired（funding-hedge 族活性期曾在册
  亲查）——摆面非 simdrill 独有，登记罩两源摆回。
- 升件源链=#4466「面清单新面」行→#4467 升件→#4500 领号→#4542
  起草（draft#33）→#4549 已批（CEO 机批）→#4558 派单→#4559
  候水→#5101 拉起→本票。
- 病灶本体=T-0448 同型族：合法新面无持久基线位——面宇宙人工过
  基线后新增才值得看，但合法新增面需登记位续判，册外/出入摆回
  重复升件炸起草拍（T-0393 链；T-0449/0453/0454/0491 dep_pair leg
  先例+T-0490/0492 dep_pair/config/gate_domain 姊妹票同径）。
- 疑似根域勘误：票面起草猜域「行情订阅覆盖面（okxws activeSet×
  md_ticks——T-0392 腿覆盖探针域）」半对——腿键枚举确走
  ActiveSetUnion 同源链，但病灶本体在面清单基线册（facesnap
  dep_pair leg 枚举族）非订阅覆盖供给面；sigdist 面零触不越域衍
  （同 T-0490/T-0492 勘误先例口径）。

### 实现摘要（枝 t/0494-dep-pair-leg-btc-usdt-swap）

- `facesnap_whitelist.go`（返修1 拆件新档）`faceBaselineWhitelist`
  册表整体迁入+登记 `face:dep_pair:leg:BTC-USDT-SWAP`（注册式单键
  精确匹配零通配零前缀——册内面差集照落「面清单新面」行携理由键
  留账但升件免；出入摆回不重复炸票/起草，T-0393 链断点同
  T-0448/0450 族先例）。
- `depblind_test.go` 先红桩
  `TestFaceBaselineLegBTCUSDTSwapRegistered`（T-0388 修复类先红
  桩律：病灶存在断言修前红+同断言修后绿——签名面必须在册且理由
  键非空，注册式不静默）。
- 探针阈/签名面/类册机制本体零改动（红线③守——只修病灶本体）。

### 返修1（棘轮新超标驳回→册表拆件断根）

- 驳回因（决策席裁决 #5142/#5143 在案）：首收 merge 撞
  check-lines 棘轮——facesnap.go 294+11=305>300 新超标硬阻断
  （基线册外首越线=硬阻断非棘轮内缩行问题；死法类=码面）。
- 改法（返修1 件钦定二径取拆分优径）：`faceBaselineWhitelist`
  册表整体迁同包新档 `facesnap_whitelist.go`（package hunger 顶层
  var 拆文件编译等价——消费面 faceBaselineExempt/reportNewFace
  原位不动）；facesnap.go 瘦身 300→199、册表档 117 行双双远低
  于棘轮——后续兄弟面票（T-0497 等）同族入册不再撞线=断根。
- rebase 新 master（a0be73f9f——含 T-0473/T-0496 同文件入册行，
  depblind_test.go EOF 双追加冲突两断言并存解）。
- 新双点（终态档面重取证）：红=`go test -run
  TestFaceBaselineLegBTCUSDTSwapRegistered`（拆件后册表尚无本键）
  →`depblind_test.go:901: 病灶存在：face:dep_pair:leg:BTC-USDT-SWAP
  未入人工基线册——窗摆回将重复升件（T-0494）` FAIL；绿=同断言
  入册后 PASS（TestFaceBaseline* 十四桩同跑全绿）。首轮双点
  （depblind_test.go:836 FAIL→PASS）在案同断言本体不变。

### 验收锚对账

- ① 桩测断言全绿（红/绿双点）: PASS——终态档面双点见返修1 节
  （depblind_test.go:901 修前红→入册修后绿，十四桩全绿）；首轮
  双点（:836）在案同断言。豁免机制面另有
  TestProbeFaceDiffBaselineExempt（T-0448 册语义桩）同拍护。
- ② 探针同窗不再报同签名病灶: PASS（机制面）——面键入册后
  reportNewFace 豁免径生效（升件免/「面清单新面」行照落携基线册
  理由键）；生产谱态亲证：leg:BTC-USDT-SWAP 已入快照批 14/15
  （机制自愈在案——册=簿记基线位）；TTL ~09:35Z 出集摆回时册内
  面新面行照落携「基线册=」理由键但升件免——同签名升件/起草链
  断（depblindDedupHours 同窗去重口径不变，探针阈/签名面/类册机
  制本体零改动）。产线行随停=决策席合并+发窗后随拍生效（遵令未
  部署）。
- ③ check_all 绿戳对口: 见机检段（收口 commit 后 check_all 实跑
  落戳——收口终盖恒本盒律）。

### 机检

`gofmt -l` 净✓ `go build ./internal/portfolio/hunger/`✓
`go vet ./internal/portfolio/hunger/`✓｜`go test
./internal/portfolio/hunger/ -run TestFaceBaseline` 十四桩全绿✓
（终态档面双点见返修1 节）｜行数自证：facesnap.go=199/
facesnap_whitelist.go=117 双双<300 棘轮✓｜check_all：返修收口
commit 后实跑（戳 `.git/check-green-<sha>` 落公共仓戳面，
pre-push 验）。

### 自检 A–I

A 复用现有册件零新造（faceBaselineWhitelist=T-0448 既有登记位）✓；
B 直击病灶本体（根因=合法新面未入册非探针错）✓；C 零死码/TODO✓；
D 红桩断言修前红修后绿+对照面=册外面照升
（TestProbeFaceDiffBaselineExempt）✓；E 红线逐条：只碰本票签名
面/未部署未并/探针阈签名面类册零动/钱径凭据零碰✓；F 票面回填同
步✓；G 三锚逐条上对账✓；H 生产亲证五径：ops_event #4466 新面行+
#4467 升件行在案（psql 亲读）/fin_strategy 287290-287291 degraded
格行亲读（SG o-postgres 直读）/fin_exec_drill id=5/6 pass 凭据
同刻落账亲读/ops_face_snapshot 批 14/15 leg:BTC-USDT-SWAP 在册
亲读/ActiveSetUnion+depenum depLegKeys+retireStaleDegraded 码面
亲读——非听闻✓；I 非展示值面 N/A✓。

### 涉及面与范围外确认

实际触碰：`internal/portfolio/hunger/facesnap_whitelist.go`
（返修1 拆件新档——faceBaselineWhitelist 册表整体迁入+本键 +1
登记）、`internal/portfolio/hunger/facesnap.go`（册表迁出瘦身
300→199 行，机制本体零改动）、`internal/portfolio/hunger/
depblind_test.go`（先红桩 +1 测试）、本票面。钦定径
`internal/research/sigdist` 未触——根因不在信号蒸馏域（腿键枚举
属 hunger 面清单基线册族，票面疑似根域系起草模板猜域已勘误半对），
册内候选径为白名单非必触清单。范围外发现（呈裁量不代修）：

- simdrill ephemeral 策略行（state='degraded'，template_ref=
  'simdrill:%'）经 ActiveSetUnion 同源链同时喂 okxws 订阅集+腿
  覆盖探针+面清单 dep_pair 枚举——演习 TTL 期内其 inst 键占
  订阅/覆盖域（同批 MUU-USDT-SWAP leg 面在册同型——旧面因 live
  funding-hedge #3544 在册未升；BTC-USD-270326=T-0491 册内；
  BTC-USDT-261127-67000-P 宇宙闸滤除不入集）。枚举语义=钦定同源
  表达式（T-0392 票面钦「归一链复用 activeSet 同源表达式」禁第二
  套）+degraded 态=哨戒降级联动设计态——演习行入集是否应豁属
  类册/签名面设计裁量（改枚举=本票红线③禁碰面），本票只登记簿记
  基线位消重复升件。
- 同批新面 service×4（外审/演习/裁定席/跑席，#4468-#4474）→
  T-0496 face-service-mech 兄弟票域，各有己域不代判（T-0493 移交
  注记「兄弟面票 T-0492~0497 各有己域不代判」同口径）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配零前缀——本票仅新增
  `face:dep_pair:leg:BTC-USDT-SWAP` 一键；复核查询式亲跑：本签名
  升件谱唯一=#4467（24h 去重窗内零他键）——无掩盖并发病灶。
- dep_pair leg 面类同型面核查：ops_face_snapshot 批 14/15 leg:*
  面 73/71 键全谱在册——BTC-USDT-SWAP 本拍唯一册外新 leg 面
  （同批 simdrill 余格 inst 皆旧面/册内/宇宙闸滤除，见上）；
  「该建未建」残余差集口径下本面=合法枚举产物非残余缺口。
- 余各「观测盲 face:*」升件键照升件等各票域裁决（T-0495~0497
  族在案候派，本票不越权代判）。
