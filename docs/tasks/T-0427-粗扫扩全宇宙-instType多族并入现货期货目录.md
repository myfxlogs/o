# T-0427 粗扫扩全宇宙——instType 多族并入现货/期货目录

- 状态：待验（外派施工收口 2026-11-07——分支已推，绿戳对口）
- 领取：外派施工（O-Dev 径——扫面扩族钦定全，不占 DM 钱径额度；支 t/0427-scan-universe @ /opt/o-wt/t0427）
  ——施工 session `zcode-t0427` @ 2026-11-07 领单开工
- 优先级：P1——发现域结构性缺（业主令根基定位）
- 收口移交归：决策席
- 关联：T-0401（两级制本件）、T-0416（UniverseCatalog 供给件
  当前 SWAP-only）、T-0412（选符域吃 md_tick_scan——本票落地
  即现货/期货可选可提名，前端零改动）、方向单数据面（另案）

## 病灶（实机三证在案）

- `md_tick_scan` 528 行清一色 SWAP（`NOT LIKE '%-SWAP'`=0）；
- `scan.go` REST 钦定 `instType=SWAP` 单请求；
- `UniverseCatalog` 拉 `/public/instruments?instType=SWAP`——
  目录供给件同盲；
- 后果链：自选报价/价格提醒搜不到未订阅现货符
  （BTC-USDT 靠持仓腿侥幸在采，ETH-USDT 类全盲）；
  期货/现货宇宙零发现——「全宇宙」实为「全 SWAP」。

## 范围

1. **UniverseCatalog 多族**：`instType` 参数化——
   `SWAP`/`SPOT`/`FUTURES` 三族目录（同供给件同 TTL 同
   fail-closed；OPTION/MARGIN 票面不收——无消费面）；目录
   行须携 inst_type 归位（消费面后续按族取子集）；
2. **okxscan 多族扫**：每拍按族各一发 REST（60s 节律不变，
   扫面 ×3 实机约 +800 行最新行——体积纪律不变）；谱域闸
   按族各判（该族目录读败=该族本拍让位不虚构，他族照常——
   fail-closed 粒度到族）；
3. **`md_tick_scan` 族标**：inst_id 形状可辨族（-USDT/-SWAP/
   -USD-YYMMDD），但钦定显式 > 键内隐式——加 `inst_type`
   列（迁移序号接续；存量行回填 'swap'）——防族判散落
   成 inst_id 正则复制件（单位错配同族病预防）；
4. **消费面对拍**：选符域 `candSets`/symDomain 谓词零改动
   应自动见新族符（断言行在验收）；晋升径 fin_sub_request
   对现货/期货符同事务提名→collector SubRequestArm 拾取
   →okxws 订阅该族符 tick——订阅臂族兼容实证（现货符
   WS 通道若与 SWAP 不同源，本票实接不假设）；
5. **MT5 谱系不沾**：MT5 目录=broker Symbols 另径
   （venue 注册面族），本票不并。

## 验收

- 红桩：三族 tickers fixture→md_tick_scan 携 inst_type 落行；
  某族目录读败=该族零行他族不受影响（fail-closed 到族断言）；
- 实机锚（部署后验）：SG `md_tick_scan` 三族行在（SPOT/
  FUTURES 非零）+选符域 candSets 见现货符（如 ETH-USDT
  非订阅态）+现货符建提醒同事务提名晋升→md_ticks 实采
  到 tick（全径实证非断言）；
- 绿戳对口+全链。

## 结果回填

- 状态：待验（施工 session `zcode-t0427` @ 2026-11-07 收口）

### 实现摘要

1. **UniverseCatalog 三族**（`internal/data/okxws/universe.go`）：refresh
   instType 参数化——SWAP 锚族（硬败上抛，原形）+SPOT 族（live -USDT
   对全集，USDC 键域滤除同律）+FUTURES 族（quarter 形沿用 T-0189 钉，
   禁猜零新方言）；族读败=族级记账（famErr）——FamSet 该族上抛让位、
   Symbols 面降级不连带照供；`FamSet(instType)` 族取子集出口=粗扫
   Gates 挂线+**T-0426 仿真层目录接口位**（票面②钦定留位）；订阅宇宙
   all 并入现货族（非 SWAP 基币现货对可晋升可订阅——现货符提名后
   tickGate valid 集须含之，否则僵尸提醒复病；基币派生臂保留兜底，
   T-0129 语义不回退）；SwapSet 保留=一行委托 FamSet("SWAP") 兼容面
   （生产调用已零——决策席可裁删）。
2. **okxscan 多族扫**（`internal/data/okxws/scan.go`）：扫族册
   scanFams 三族各一发（SWAP/SPOT/FUTURES，60s 节律不变）；族内环
   （谱域闸→REST→解析→键域滤→族标落行）任一败=该族让位他族照常
   （fail-closed 粒度到族，票面钦定）；全族败=本拍 loud 岗终（原
   fail-closed 形同守）；部分让位显式 log 不静默；行携 inst_type
   族标（**显式>键内隐式——禁猜钦定形：本件零 inst_id 形状族判**，
   族标来自发面原词小写 swap/spot/futures）；体积纪律不变——
   overwrite 最新行语义表型不变（同键 upsert+ts 守卫原样）。
3. **md_tick_scan 携族标**（迁移 md_1092+fin_0133 同形镜像+登记
   接续 {1092}/{133}）：`inst_type text NOT NULL DEFAULT 'swap'`——
   存量行清一色 SWAP 族（md_1091 起单族扫），DEFAULT 直填即回填零
   UPDATE；投影面 twotier SELECT/INSERT 列随行（SG 镜像同列归位）。
4. **消费面对拍**（票面④）：选符域 candDomainSQL/symDomain 谓词零
   改动——族符落 md_tick_scan 即自动入域（桩测
   TestIntentPromoteFamilySpot 断言：SPOT/FUTURES 符入候选域+现货
   目录符建单同事务提名 user_intent/open 晋升行落）；collector
   SubRequestArm/tickGate 族兼容=谓词族无关（纯 inst 谓词）+宇宙含
   族符（universe 桩测断言）——现货符 tickers WS 通道不假设之实证
   =生产在跑（dcd 现货腿 T-0171 同通道同 face 采集中）；MT5 谱系
   零沾（票面⑤）。

### 测试证据

- **红桩先行**（票面验收首锚）：三族 fixture 测试+族级 fail-closed
  断言先落，跑出编译红证（ScanGate/Gates/inst_type 全缺）后实现
  转绿——红→绿全程在案；
- 三族落行携 inst_type：`TestScanFamiliesLand` PASS（swap/spot/
  futures 三族标逐行对拍）；
- 族级 fail-closed：`TestScanFamilyFailClosed` PASS（SPOT 族读败=
  该族零行他族落行拍不炸；全族败 loud 上抛）；
- 目录族语义：`TestUniverseCatalogFamSets/FamFail/FailClosed/
  Parse/TTL` PASS（族集归位/USDC 滤/族败让位他族照常/Symbols 降级
  不连带/基币派生兜底/TTL 三族零重读）；
- 消费面对拍：`TestIntentPromoteFamilySpot` PASS（候选域零改动见
  族符+现货符晋升提名）；
- 机检：`bash scripts/check_all.sh` **全链绿 mode=full**（go build✓
  vet✓ import✓ gofmt✓ env✓ rejects✓ config-write-domain✓
  migrations 七断言✓ bridge-audit 四列✓ tolerate✓ ops-chain✓ go
  test 全谱✓）——log /tmp/checkall_t0427.log；
- 测试实例隔离：专库 o_test_t0427（KB parallel-checkall-shared-
  testdb-migrate-hang 律）；全谱首跑 internal/platform/events
  TestListenSocketStarvationRootFix 红——与本案面零交集，单跑即绿
  （兄弟票 t0426/t0430 全谱并行 socket 争用假红，KB 坑域），避峰
  重跑全链净绿。

### 偏离说明

- 订阅宇宙 all 并入 SPOT 族为必要延伸（票面未明写）：现货符晋升
  后 tickGate valid 集=订阅宇宙——族符不在宇宙=提名后永 pending
  （僵尸提醒复病），不并则票面④「晋升径…okxws 订阅该族符 tick」
  不成立；桥面 FullSub index-tickers 订阅面随 +~130 对（T-0137
  「index-tickers 现货全量」设计内），SG 瘦身面零变化（现货形
  宇宙内零订——T-0171 原语义）。
- FUTURES 族谱域=quarter 形（T-0189 验证形沿用非全谱期货）——
  扫面行数对位票面「+800」量级（SPOT-USDT+quarter）；周度期货
  不入（无消费面+体积纪律）。

### 验收锚对账

- 红桩①三族 fixture 落行携 inst_type：**PASS**（TestScanFamiliesLand，
  O_TEST_PG_DSN 专库真跑非 skip）；
- 红桩②族读败零行他族不受影响：**PASS**（TestScanFamilyFailClosed）；
- 绿戳对口+全链：**PASS**（check_all mode=full 全绿戳落 worktree）；
- 实机锚（SG 三族行在/candSets 现货符/提醒晋升→md_ticks 实采）：
  **部署后验**——本票不合并不部署（票面钦定），留决策席裁决后
  部署窗验。

### 自检 A–I

- A 架构：同一供给件扩族（FamSet 子集出口）非第二套目录；复用
  ScanGet/rejects/pgx.Batch/pgstore 迁移链——零新框架；
- B 实现：族循环直击票面钦定形，无多余间接层；
- C 洁净：无死码/调试残渣；SwapSet 一行委托兼容面（注释明标，
  删留归决策席）；
- D 正确：红桩+族 fail-closed+全族败 loud+回戳守卫——对抗面测试
  在案（删全族败 Join 行/闸滤行/族标列均必有测试红）；
- E 合规：写面只 md_tick_scan（fin_* 零触）；OPTION/MARGIN 不收；
  零新 config 键；迁移域白名单/八桥同形机检过；
- F 文档：数据流.md 粗扫段+宇宙扩件段同 commit 更新；票面本回填；
- G 验收锚对账：见上节（实机锚=部署后验明提）；
- H 生产现实对拍：instId 方言全走已验证形（T-0189 quarter/
  T-0129 基币现货/USDT 键域律），零新形状猜测；实机行数锚留部署
  后验（票面钦定后验位）；
- I 展示值与判定取值同源：选符域仍单源 md_ticks∪md_tick_scan
  （本次零谓词改动——同源性不受影响）。

### 涉及面

- migrations/md_1092_scan_universe.sql（新）+fin_0133_scan_universe.sql
  （新）+internal/store/pgstore/migrate_table.go（登记）
- internal/data/okxws/{universe,scan}.go+各自 _test
- internal/projector/twotier.go（投影列随行）
- cmd/collector/main.go（Gates 三族挂线）
- cmd/core/quotes_face_test.go（消费面对拍测试）
- docs/数据流.md（链路变更同 commit）

### 范围外确认

未越单：MT5 谱系零沾；fin_sub_request 语义零改（user_intent 晋升
径原样）；fin_proposal/评估臂零触；proto/前端零改动（T-0412 钦定
零改动成立）；部署零动作。
