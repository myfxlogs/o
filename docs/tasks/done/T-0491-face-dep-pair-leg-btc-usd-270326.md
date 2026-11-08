# T-0491 探针升件自动起草：观测盲@face:dep_pair:leg:BTC-USD-270326

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·取证=收口并master·机械轨票翻）
- 领取：o-worker-T-0491 @ 2026-11-08（无头施工席——常径机派 T-0324）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:dep_pair:leg:BTC-USD-270326`，草案锚 fin_ticket_draft#29）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:dep_pair:leg:BTC-USD-270326`（kind=探针类 `观测盲`，key=病灶键 `face:dep_pair:leg:BTC-USD-270326`）

末行升件原文：
> 面清单新面 face:dep_pair:leg:BTC-USD-270326：机器可列面 dep_pair 类新增 leg:BTC-USD-270326——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#4039

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:dep_pair:leg:BTC-USD-270326' ORDER BY id DESC LIMIT 20
```

## 疑似根域

行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:dep_pair:leg:BTC-USD-270326（T-0455 册内候选，DM 裁时可改判）；
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

根因定位（亲证五径）：非码缺陷——facesnap 探针按设计正确检出「机器可列面超出上拍快照册」的**合法新面**。T-0491 面=BTC-USD-270326 定日腿：`fin_proposal` basis 族（模板 quarter_basis_long/perp_quarter_spread——direction=basis_quarter/perp_quarter_spread 实读对拍）`fin_armed` 窗在册（ActiveSetUnion 提案臂枚举本体零硬编码；`legset.IsUSDTInst` 定日形 `-\d{6}$` 尾豁免=basis 域钦定数据源非杂币）。间歇性实证：armed_ttl_expired 周期 ~2h 间隙摆出摆回——依赖对清单 census 在/缺/回摆 #3008(11-06 18:31 在)→#3498(11-07 22:24 缺)→#3923(11-08 04:24 缺)→#4035(11-08 06:48 回)，与提案窗逐点对上（22:28Z 重建↔22:24Z 缺、04:28Z 重建↔04:24Z 缺、04:28Z armed↔06:48Z 回）；拍差集只报「回」不报「缺」+快照仅留最近两拍（b11 缺→b12 回差集命中）=首报迟至 armed 窗撞上拍（#4039）。危害面=armed 窗摆回撞上拍差集重复升件炸 T-0393 起草拍（24h 去重只吸同窗，跨窗必再炸）。修法=「新面核+入册」：`faceBaselineWhitelist` 注册面键（facesnap.go——T-0448 机制自设计「合法新面登记位」，depBlindWhitelist 同式先例），升件免、新面行照落携理由键留账。CEO 裁 #4058 定日BTC腿新面合法性归草案基线审=本件即基线审执行。

### 验收锚对账

- ① 桩测断言全绿（红/绿双点）: PASS——红点=`go test -run TestFaceBaselineLegBTCUSD270326Registered`（入册前）→ `depblind_test.go:680: 病灶存在：face:dep_pair:leg:BTC-USD-270326 未入人工基线册——窗摆回将重复升件（T-0491）` FAIL；绿点=同断言入册后 PASS（`ok o/internal/portfolio/hunger`）。豁免机制面另有 TestProbeFaceDiffBaselineExempt（T-0448 册语义桩）同拍护。
- ② 探针同窗不再报同签名病灶: PASS（机制面）——面键入册后 reportNewFace 豁免径生效（升件免/「面清单新面」行照落携基线册理由键）；生产谱态亲证：armed 窗摆动 census 在案（#3498 缺→#4035 回=摆形本体非持续在册——入册后摆回不再升件）。产线行随停=决策席合并+发窗后随拍生效（遵令未部署）。
- ③ check_all 绿戳对口: 见机检段（收口 commit 后 check_all 实跑落戳）。

### 机检

`go build ./...`✓ `go vet ./...`✓ `gofmt -l` 净✓｜`go test ./internal/portfolio/hunger/`（O_TEST_PG_DSN 实库 5434）ok 124.478s 全谱✓｜check_all：收口 commit 后实跑（戳 `.git/check-green-<sha>` 落公共仓戳面，pre-push 验）

### 自检 A–I

A 复用现有册件零新造（faceBaselineWhitelist=T-0448 既有登记位）✓；B 直击病灶本体（根因=合法新面未入册非探针错）✓；C 零死码/TODO✓；D 红桩断言修前红修后绿+对照面=册外面照升（TestProbeFaceDiffBaselineExempt）✓；E 红线逐条：只碰本票签名面/未部署未并/探针阈签名面类册零动/钱径凭据零碰✓；F 票面回填同步✓；G 三锚逐条上对账✓；H 生产亲证五径：ops_event #4039 升件行在案（psql 亲读）/fin_proposal armed 循环亲读（52 提案 @2026-10-30 起、当前 417840/417841 fin_armed、armed_ttl_expired 周期）/census 摆动四点亲读（#3008/#3498/#3923/#4035）/模板对拍（eval_id→quarter_basis_long/perp_quarter_spread）/CEO 裁 #4058 在案——非听闻✓；I 非展示值面 N/A✓。

### 涉及面与范围外确认

实际触碰：`internal/portfolio/hunger/facesnap.go`（faceBaselineWhitelist +1 登记）、`internal/portfolio/hunger/depblind_test.go`（先红桩 +1 测试）、本票面。钦定径 `internal/research/sigdist` 未触——根因不在信号蒸馏域（basis_ann 信号面已由 T-0447 册另案覆盖），册内候选径为白名单非必触清单。范围外：零——未碰探针阈/签名面/类册（depConfigReaders/depTablePairs/depSignalFamilies 原样）、未碰其他票面、未碰 simdrill/exec/钱径面。

### 偏离说明

- **simdrill 臂观察（非本票范围，呈决策席知会）**：simdrill 矩阵产跑残件（T-0480）6 条 degraded 策略含本 inst 两条（287288/287289，09:35Z 建）亦在 ActiveSetUnion live/degraded 臂内供腿——本票面合法性不依赖该臂（升件时点 06:48Z 前 simdrill 未建，armed 提案独立供给）；sim 策略入腿集是否该分域=邻域题（T-0489 钱径仓写径分域在审），本席不越单修。
- **同族 merge 提示**：T-0449/T-0490/T-0492/T-0493 同 facesnap.go/depblind_test.go 尾插在飞——合后冲突解=各条登记+各先红桩全保（T-0454 收口裁记同式先例）。
