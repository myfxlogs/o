# T-0449 探针升件自动起草：观测盲@face:dep_pair:leg:TRX-USDT-SWAP

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·取证=收口并master·机械轨票翻）
- 领取：o-worker-T-0449 @ 2026-11-09（无头施工席——常径机派 T-0324）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:dep_pair:leg:TRX-USDT-SWAP`，草案锚 fin_ticket_draft#11）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:dep_pair:leg:TRX-USDT-SWAP`（kind=探针类 `观测盲`，key=病灶键 `face:dep_pair:leg:TRX-USDT-SWAP`）

末行升件原文：
> 面清单新面 face:dep_pair:leg:TRX-USDT-SWAP：机器可列面 dep_pair 类新增 leg:TRX-USDT-SWAP——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#3406

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:dep_pair:leg:TRX-USDT-SWAP' ORDER BY id DESC LIMIT 20
```

## 疑似根域

行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/portfolio/hunger/facesnap.go+depblind*（dep_pair:leg:TRX-USDT-SWAP 新面合法性核+入册——TRX 腿面册外残余）；
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

### 实现摘要

根因定位（亲证五径）：非码缺陷——facesnap 探针按设计正确检出「机器可列面超出上拍快照册」的**合法新面**。T-0449 面=`leg:TRX-USDT-SWAP` 本键腿：`fin_sub_request` open 行（requester=`user_intent`、reason=`quote_watchlist`、ts=2026-11-07 12:43:45Z）经 `legset.SubRequestArm` 入 `ActiveSetUnion`（depenum 枚举本体零硬编码；`legset.IsUSDTInst` -USDT 计价闸过、订阅宇宙在册）。与 T-0450 册内 `face:dep_pair:subreq:user_intent:TRX-USDT-SWAP` **同一 open 行的腿面投影**——同根双面（subreq 对签名+leg 对签名并出）。排除他臂亲证：fin_proposal 290880=fin_retired、fin_strategy 286006=retired（非提案/策略臂供给）；派生三臂（basis/dcd/funding-hedge）产现货 -USDT 形非 -SWAP 本键。间歇性实证：fin_sub_request 全表仅此 1 行 state='open'——请求闭→面消失、重开→册外新面再升件；快照 b13 首现（升件 #3406 @2026-11-07 13:48Z）→b14 在册。危害面=请求窗摆回撞上拍差集重复升件炸 T-0393 起草拍（24h 去重只吸同窗，跨窗必再炸）。修法=「新面核+入册」：`faceBaselineWhitelist` 注册面键（facesnap.go——T-0448 机制自设计「合法新面登记位」，depBlindWhitelist 同式先例），升件免、新面行照落携理由键留账。

### 验收锚对账

- ① 桩测断言全绿（红/绿双点）: PASS——红点=`go test -run TestFaceBaselineLegTRXUSDTSwapRegistered`（入册前）→ `depblind_test.go:731: 病灶存在：face:dep_pair:leg:TRX-USDT-SWAP 未入人工基线册——窗摆回将重复升件（T-0449）` FAIL；绿点=同断言入册后 PASS（`ok o/internal/portfolio/hunger`）。豁免机制面另有 TestProbeFaceDiffBaselineExempt（T-0448 册语义桩）同拍护。
- ② 探针同窗不再报同签名病灶: PASS（机制面）——面键入册后 reportNewFace 豁免径生效（升件免/「面清单新面」行照落携基线册理由键）；生产谱态亲证：sub_request open 行仍在（面持续在册非消失——入册后窗摆回不再升件）。产线行随停=决策席合并+发窗后随拍生效（遵令未部署）。
- ③ check_all 绿戳对口: 见机检段（收口 commit 后 check_all 实跑落戳）。

### 机检

`go build ./...`✓ `go vet ./...`✓ `gofmt -l` 净✓｜`go test ./internal/portfolio/hunger/`（O_TEST_PG_DSN 实库 5434 租约独占）ok 169.383s 全谱✓｜check_all：收口 commit 后实跑（戳 `.git/check-green-<sha>` 落公共仓戳面，pre-push 验）

### 自检 A–I

A 复用现有册件零新造（faceBaselineWhitelist=T-0448 既有登记位）✓；B 直击病灶本体（根因=合法新面未入册非探针错）✓；C 零死码/TODO✓；D 红桩断言修前红修后绿+对照面=册外面照升（TestProbeFaceDiffBaselineExempt）✓；E 红线逐条：只碰本票签名面/未部署未并/探针阈签名面类册零动/钱径凭据零碰✓；F 票面回填同步✓；G 三锚逐条上对账✓；H 生产亲证五径：ops_event #3406 升件行在案（psql 亲读）/fin_sub_request open 行亲读（user_intent/quote_watchlist @2026-11-07 12:43Z 仍 open，全表唯一行）/fin_proposal+fin_strategy 退役态亲读排除他臂/ops_face_snapshot b13/b14 在册亲读/md_ticks TRX-USDT-SWAP 行在案（订阅消费在流）——非听闻✓；I 非展示值面 N/A✓。

### 涉及面与范围外确认

实际触碰：`internal/portfolio/hunger/facesnap.go`（faceBaselineWhitelist +1 登记）、`internal/portfolio/hunger/depblind_test.go`（先红桩 +1 测试）、本票面。范围外：零——未碰探针阈/签名面/类册（depConfigReaders/depTablePairs/depSignalFamilies/depBlindWhitelist 原样）、未碰其他票面、未碰 simdrill/exec/钱径/凭据面。**疑似根域注记**：票面「疑似根域=行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）」——实证根因在面清单 diff 册（T-0405 域）非覆盖探针：leg 对失明探针 depblind 独立在册不受影响（md_ticks 有行证明订阅消费在流，入册只免 face-diff 升件不豁免腿覆盖失明判定）。

### 偏离说明

- **同族 merge 提示**：T-0490/T-0492/T-0493 同 facesnap.go/depblind_test.go 尾插在飞——合后冲突解=各条登记+各先红桩全保（T-0454/T-0491 收口裁记同式先例）；本件 rebase 后基线已含 T-0491 条目。
- **同根双面注记**：本面与 T-0450 册内 `subreq:user_intent:TRX-USDT-SWAP` 同出一 open 行——两册键各自独立登记（subreq 对签名/leg 对签名不同 esc_scope），非重复登记。
