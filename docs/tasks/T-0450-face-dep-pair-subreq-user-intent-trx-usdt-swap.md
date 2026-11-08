# T-0450 探针升件自动起草：观测盲@face:dep_pair:subreq:user_intent:TRX-USDT-SWAP

- 状态：待验（业主手动外派 2026-11-08——号序串行 T-0450→0453→0454；证据移交外审席复审）
- 领取：zcode-facepair @ 2026-11-08（业主手动外派令——窄口豁免径，收口照常回决策席）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:dep_pair:subreq:user_intent:TRX-USDT-SWAP`，草案锚 fin_ticket_draft#12）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:dep_pair:subreq:user_intent:TRX-USDT-SWAP`（kind=探针类 `观测盲`，key=病灶键 `face:dep_pair:subreq:user_intent:TRX-USDT-SWAP`）

末行升件原文：
> 面清单新面 face:dep_pair:subreq:user_intent:TRX-USDT-SWAP：机器可列面 dep_pair 类新增 subreq:user_intent:TRX-USDT-SWAP——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#3408

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:dep_pair:subreq:user_intent:TRX-USDT-SWAP' ORDER BY id DESC LIMIT 20
```

## 疑似根域

行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/portfolio/hunger/facesnap.go+depblind*（dep_pair:subreq:user_intent:TRX-USDT-SWAP 新面核+入册）；
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

根因定位（亲证三径）：非码缺陷——facesnap 探针按设计正确检出「机器可列面超出上拍快照册」的**合法新面**。T-0450 面=fin_sub_request open 行（requester=user_intent，inst_id=TRX-USDT-SWAP，2026-11-07 12:43:45Z open，仍 open）——depenum.go SubRequestArm 枚举本体在册即面（零硬编码），请求闭/开摆动=间歇面形（T-0448 faceBaselineWhitelist 册语义罩位正例）。危害面=窗摆回重复升件炸 T-0393 起草拍。修法=DM 钦定「新面核+入册」：faceBaselineWhitelist 注册该面键（facesnap.go——机制自设计「合法新面登记位」），升件免、新面行照落携理由键留账，窗摆回不重复炸票。

### 验收锚对账

- ① 桩测断言全绿（红/绿双点）: PASS——红点=`go test -run TestFaceBaselineSubreqUserIntentTRXRegistered`（入册前）→ `depblind_test.go:553: 病灶存在：face:dep_pair:subreq:user_intent:TRX-USDT-SWAP 未入人工基线册——窗摆回将重复升件（T-0450）` FAIL；绿点=同断言入册后 PASS（`ok o/internal/portfolio/hunger`）。豁免机制面另有 TestProbeFaceDiffBaselineExempt（T-0448 册语义桩）同拍护。
- ② 探针同窗不再报同签名病灶: PASS（机制面）——面键入册后 reportNewFace 豁免径生效（升件免/行照落）；生产谱态亲证：面已入 b9+b10 快照批（谱稳拍差集空不重报），窗摆回由入册径免升。产线行随停=决策席合并+发窗后随拍生效（遵令未部署）。
- ③ check_all 绿戳对口: **未达（缺口=非本票病灶）**——T-0460 已批闸病：`check_seat_writes` author 兜底 `claude-glm@local` 误判 [角色:施工] commit 为决策席越权（本票触 internal/ 码面被咬；T-0460 钦定修法=角色标 commit 出判别集，尚未实施）。非本票范围不修闸——T-0445/T-0459 同判先例：枝留本地移交留案。实证：check_all 跑至 seat-writes 步前全绿（env-reads/migrations/rejects/import/gofmt/build/config-write-domain/bridge-audit/tolerate-report/ops-chain 逐闸 PASS 在 log），**go test 全谱手动补跑 PASS**（`go test ./... -timeout=1500s` → 63 包 ok 零败 EXIT=0）。「并入发窗批」=决策席合并后按镜像面判据入窗（本票 facesnap.go 入 o-core 镜像面）。

### 机检

gofmt✓ vet✓ build✓（hunger 包+全仓）｜go test 全谱 63 包 ok✓（手动补跑，上锚③）｜check_all: seat-writes 步前全绿，seat-writes 存量红（T-0460 已批在修，非本票面）

### 自检 A–I

A 复用现有册件零新造（faceBaselineWhitelist=T-0448 既有登记位）✓；B 直击病灶本体（根因=合法新面未入册非探针错）✓；C 零死码/TODO✓；D 红桩断言修前红修后绿+对抗面=册外面照升（TestProbeFaceDiffBaselineExempt 对照面）✓；E 红线逐条：只碰本票签名面/未部署未并/探针阈签名面类册零动/钱径凭据零碰✓；F 票面回填同步✓（facesnap 无模块页面）；G 三锚逐条上对账✓；H 生产亲证三径：ops_event #3408 行在案（psql 亲读）/fin_sub_request open 行亲读（SG）/b9+b10 快照批亲读——非听闻✓；I 非展示值面 N/A✓。

### 涉及面与范围外确认

实际触碰：`internal/portfolio/hunger/facesnap.go`（faceBaselineWhitelist +1 登记）、`internal/portfolio/hunger/depblind_test.go`（先红桩 +1 测试）、本票面。范围外：零——未碰探针阈/签名面/类册（depConfigReaders/depTablePairs 原样）、未碰其他票面。

### 偏离说明

- 推送留案：领单 commit（锁）已推远端；impl+回填 commit 本地滞留——push 被 pre-push 绿戳闸挡（check_all 止于 seat-writes 存量闸病 T-0460，上锚③）。T-0445/T-0459 先例径：决策席/外审可经共享 git dir 直 merge 本枝（`t/0450-face-dep-pair-subreq-user-intent-trx-usdt-swap`@本收口 hash），或候 T-0460 落地后补推。
- 同族提示（非本票范围）：T-0453/T-0454 同 facesnap.go 入册件，三枝独立自 2f302586——建议按号序串行 merge，faceBaselineWhitelist map 尾插三处登记文本冲突=保留全部三条。

分支: t/0450-face-dep-pair-subreq-user-intent-trx-usdt-swap（领单锁已推；impl/回填本地留案——闸病 T-0460 上锚③）
移交: 决策席（业主令：证据报外审席复审）
[施工收口:T-0450] @b3f376d4
