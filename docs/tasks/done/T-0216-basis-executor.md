# T-0216 basis 族 executor+short 腿能力缺口

> 状态：完成 | 领取：claude-glm-t0216r @ 2026-10-29（首交 devin-t0216 打回返修续任） | 优先级：P2（次于 T-0215——数据刚就位，executor 后一步）
> 立项：决策席 @ 2026-10-28

## 面况

- quarter_basis_long/perp_quarter_spread 两腿全在能力账
  （trade-order 覆盖 spot/perp/futures）——评估器修完（T-0213）即
  可评，executor 是开仓最后一块。
- quarter_basis_short 需 `spot_short` 腿=保证金借币卖出——能力账
  无 margin borrow 面（OKX /api/v5/account/borrow-repay 未在
  fin_capability_surface 在册）→条件缺口件，先立 leg 能力再谈
  executor。

## 钦定修法

- executor 钉版后接 funding-hedge 开闸同构（持仓/平仓/ledger 全链）。
- paper 径先行同 T-0215。
- spot_short 腿：margin borrow 能力面立项补齐（endpoint 在册+
  读面实测）→不入账=quarter_basis_short 恒缺腿评不了一条腿。


## 结果回填

**实现**（六件）：①`grammar.go`——quarter_basis_long/perp_quarter_spread
Executor 落钦定名；quarter_basis_short NeedFace 挂 margin-borrow-repay
（enabled=false→族不枚举=恒缺腿如实呈，不再产假想卡噪）；②evalBasis
proposed→fin_proposal 提案径（evidence 补 apr 键过 T-0205 硬闸，inst=季度
期货键 direction=basis_quarter）；③lifecycle paperTemplates+两族（short
不入——腿缺）；④`execution.BuildBasisLegs`——衍生腿=到期日期货 sell
（FUTURES 乐器域 ctVal 张数径）+多头腿 spot quote_ccy/perp 张数，futUSD
反推对齐律同 T-0062；⑤`cmd/core/basis_open.go` 派发——cap/冷却/闸/腿构/
Execute 原子双腿全链 funding-hedge 同构，loadMeta -YYMMDD→FUTURES；
⑥`md_1061_margin_borrow_capability.sql`——borrow-repay（write）+
borrow-repay-history（read）双行在册 enabled=false（登记表补行）。
测试：`TestBuildBasisLegs` 绿（USD 面值 10 万张 fut+spot quote_ccy $1000
对齐/perp 币本位 1.23 张）；假想卡断言随 executor 落位移锚 ≥3→≥2。

**返修**（2026-10-29 claude-glm-t0216r——三项按派单项+码面勘查落；打回单
原文未落档未获，逐项锚证见码面/测试，裁决模型两处自定待验收席裁）：

1. **basis paper 记账面补装**：basis 两族入 paperTemplates 而零记账径
   （paperAccounting 只吃 md_funding 行——期货键无 funding 行；
   dcdPaperAccounting 白名单不含 basis）→paper 恒零 fill、promote 三判据
   永假。补装三件：①`lifecycle/paper.go` `basisPaperAccounting`（dcd 同构
   日摊径）——期货腿价=md_mark_index mark-price 桥面行（paper 策略 inst
   在投影活跃集谓词内），现货腿价=md_ticks 本地行
   （`basisSpotLeg` 归一链 BTC-USD-261225→BTC-USDT 同 dcd/activeSet 律），
   exec.tick_max_age_s 鲜度闸双侧同律，缺/陈=fail-closed 让位不造零；
   溢价摊提 amt=notional×p/days_left×elapsed（p=(fut−spot)/spot 现价重算，
   线性衰减下 p/days_left 恒=R₀/T 跨拍 telescoping 收敛全溢价；elapsed 夹
   days_left=断窗至多补记全剩余溢价；sign 随结构如实）；perp_quarter_spread
   加计 perp 腿 funding=md_funding 行 −rate×notional（long_perp 付正租），
   fill inst_id=perp 键与溢价摊提分键互不污染 last-ts 基准；kind='funding'
   同收入族；②`okxws/subscribe.go` activeSet 补 basis 两族 paper/live 策略
   现货腿（T-0129 宇宙扩件含 USDT 现货对——valid 闸过）；③scan() 接线
   （dcd 同款非致命 log）；rejects 门注册 basis_mark_stale。
   【自定待裁】记账模型=线性衰减摊提（同 dcd apr 摊提家式）；perp 腿
   funding 计入——打回单若另有钦定模型，验收时对表裁。
2. **投影提案桥自修**：`fin_0083_basis_direction_defork.sql`（SG 侧）——
   direction 封闭词表 CHECK 扩 'perp_quarter_spread'（扩容前镜像 uEval/
   uProposal 携新词值落 SG=CHECK 拒批毒丸）+镜像行同构回填
   （eval/proposal/fin_strategy 按 template 锚定 UPDATE）——回填先于服务起
   则 uEval 自然键不分裂双行（SG 存量行旧词值不改=md 回填后同键异向分行）。
   闸函数 fin_0054 零改——(inst_id,direction) 键随词表去分叉自然归一。
3. **提案去重语叉回填**：根因=quarter_basis_long 与 perp_quarter_spread
   共用 direction='basis_quarter' 且同 inst（季度期货键）——md_1054 闸键
   (inst_id,direction) 双族撞车，BEFORE INSERT RETURN NULL **静默吞**第二族
   提案（零错零痕）；Go 预查 inst 单键同病更宽。修：①`grammar.go`
   perp_quarter_spread Direction→'perp_quarter_spread'（词表去分叉，同模板
   名 dcd_call/gap_carry 同构式）；②`md_1074_basis_direction_defork.sql`
   （md 侧）CHECK 扩容+存量 eval/proposal 按 template 回填；③run.go 预查键
   对齐闸键（inst+direction+双在途态 fin_proposed/fin_armed）——
   预查先行=吞显形为预查静默跳过、假「落卡」日志根除。
   迁移已登记 pgstore.migrationTable（md 1074/fin 83——fin_0080 号已被占）。

测试：`TestBasisDeforkDualFamilyProposals`（同 inst 双族双提案并存+direction
配对词表锚）/`TestBasisPaperFillConvergence`（摊提 fill 真生成+detail 价源
同源+陈 tick fail-closed 恒 1 行）/`TestBasisPaperFillFailClosedSpotMissing`
（现货腿缺不造零）/`TestBasisPerpFundingNegativeSign`（负号记账+分键）/
`TestBasisSpotLeg`（归一链纯件）全绿；check_all 全谱见机检行。
坑条目：`工程经验库/条目/proposal-dedup-key-cross-family-fork.md`（去重键
跨族分叉+双层键对齐律）+kbgen 重建；数据流.md paper 记账段同 commit 更新。


## 审计敌意核验要点（管理令 20261029-211316-MANAGE-62——本票=业主捕获假完工种子#1）

- **六件逐条实态核验**：结果回填宣称件逐条验产物——grammar 落名实查/
  evalBasis 提案径实跑/paperTemplates 实存；锚上没有的按不存在计，
  不许『大体相符』；
- **margin borrow 缺口重点核**：回填所述处置是否真实（short 腿恒缺根因，
  最可能藏假）——borrow 能力面/fin_capability_surface 在册行/读面实测；
- **armed 双族真 capability 缺口清点入票**：驳回不丢人——打回即重立钦定界，
  缺口如实清单化；
- 审计过程留痕=清债排查校准样本。

## 审计判决（审计席：headless @ 2026-10-29）

逐锚实态对账（码面+测试实例+生产 PG-md/PG-fin 三面亲证）：

- ① grammar 落名实证——quarter_basis_long/perp_quarter_spread Executor 在
  位（钦定名 *_executor）；short NeedFace 挂 margin-borrow-repay 且
  executor 缺省——PG-md 实拍该面 enabled=f→Enumerate 不枚举：产上 short
  eval 止 10-28 14:33Z（返修部署前），提案零行，恒缺腿如实呈未藏。
- ② evalBasis 提案径实跑——evidence 补 apr 键过 T-0205 闸；产上
  fin_opp_eval BTC-USD-261225 双族 proposed（11:43Z）+fin_proposal
  406218/406219 同 inst 双族并存 fin_armed（各持分叉 direction）——
  去重语叉根修产上直证，md_1054 共键吞件病灶实证消。
- ③ paperTemplates 两族实存（short 不在）——SG fin_strategy
  286990(quarter_basis_long)/287004(perp_quarter_spread) paper $50 实拍。
- ④ BuildBasisLegs 衍生腿=到期日期货 sell（loadMeta -YYMMDD→FUTURES 元
  数据域）+多头腿 spot TgtQuote/perp 张数，futUSD 反推对齐；⑤ basis_open
  cap/冷却/闸/腿构/Execute 原子双腿全链同 funding-hedge 构；⑥ md_1061
  双行（borrow-repay write=f + borrow-history read=f）PG-md 实拍在册，
  md_1074/fin_0083 schema_migrations 双侧已应用。
- 测试六件（defork 双提案并存/摊提收敛/缺腿 fail-closed/负号分键/
  SpotLeg 归一/BuildBasisLegs）独立实例实跑全绿非 skip；CI 绿。
- armed 双族真 capability 缺口清点：面级零缺（market-instruments/
  tickers/funding-rate/trade-order 全 enabled；executor 在）；spot_short
  margin 缺口=入册禁启用如实呈，无藏假。
- 【残差入票·部署面非码面】OCI o-collecto 未随返修重启
  （ActiveEnter=10-28 06:14Z，早于返修 commit）→activeSet 现货腿补丁
  未生效→md_ticks BTC-USDT 零行。fill 锚（286990 首摊≈15:46Z/
  287004≈17:39Z）到期将 basis_mark_stale fail-closed——与决策档挂起锚
  同因，须先重启 collector 补订现货腿再验 fill；paper fill 现零行属
  节拍内正常（elapsed<24h），非病灶。
- 【残差核销·2026-10-30 决策席实证（MANAGE-114①）】o-core 10-29 17:02Z
  重建（o-core:a6e8964cd15d）已携带本票 collector 侧 activeSet 现货腿
  补丁；决策席亲证 md_ticks BTC-USDT 现货行新鲜在流（10-30 09:4x CST
  age<2s）。ETH-USDT 现货行滞 3h+=T-0250 projector 侧归一臂未部署所致
  （另票残差，非本票）。
- 小注（不阻通过）：basis.go 结构卫语句（非到期键/过期/节拍未到）静默
  continue 未挂 rej——记账让位语义非拒单语义；detail JSON 拼接沿
  paper.go 同例；frows 中途 scan 败 break 留部分行，下拍自愈。

