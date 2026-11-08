# T-0492 探针升件自动起草：观测盲@face:gate_domain:meta

- 状态：收口待审（施工收口——回决策席常径裁）
- 领取：devin-o-worker-T-0492 @ 2026-11-09
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:gate_domain:meta`，草案锚 fin_ticket_draft#30）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:gate_domain:meta`（kind=探针类 `观测盲`，key=病灶键 `face:gate_domain:meta`）

末行升件原文：
> 面清单新面 face:gate_domain:meta：机器可列面 gate_domain 类新增 meta——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#4043

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:gate_domain:meta' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:gate_domain:meta（T-0455 册内候选，DM 裁时可改判）；
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

- 新面本体=gate_domain 面 `meta`——枚举源=`SELECT DISTINCT
  split_part(key,'.',1) FROM app_config`（facesnap.go faceInventory
  本体零硬编码）：`meta.autoapprove.registry` 键在册即产 meta 域面。
- meta 域来源=fin_0143 迁移种 `meta.autoapprove.registry`（T-0455
  册本体——签名→钦定文件径白名单；迁移 commit a0be2ee88 @2026-11-08
  07:51+0800，生产亲证 SG app_config 该键在册 updated_at
  2026-11-08 14:24:50 UTC=fin_0149 af9b4e583 合入同拍更新，册值含
  `观测盲:*`→本票钦定 2 径）。
- 读者册载=`depConfigReaders["meta"]="DM 机批闸册"`（depenum.go
  T-0448 bcd706f95 补册——dm_autoapprove.sh line 41/60/107 三处
  实读同键=候选票批权径消费链活体；本票自身 #4373 已批机批行=
  闸在跑实证）。
- 写域面核=config-write-domain.tsv `meta.*` 未入册→未命中
  fail-closed 归 owner——类册面钦定判据设计在案（册修=决策席面，
  T-0455 票面裁定节），非登记缺口非病灶。
- 合法性核=**合法新面非「该建未建」残余**：域=册本体键域在册恒产
  （非产消断非间歇摆面——app_config 键持久在册）；枚举谓词
  split_part 本体零硬编码；dep_pair 姊妹面
  `config:meta→DM 机批闸册`=T-0490 同根另册在飞（同 meta 域首现
  一事件双面投影——gate_domain/dep_pair 两独立面类各投一影）。
- 升件源链=#4042「面清单新面」行→#4043 升件→#4342 领号→#4363
  起草（draft#30）→#4375 已批（CEO 机批）→#4422 派单→#4423 候水
  →#5065 拉起→#5068/#5069 领单施工中→本票。
- 病灶本体=T-0448 同型族：合法新面**无持久基线位**——面宇宙人工
  过基线后新增才值得看，但合法新增面需登记位续判，册外/重种回摆
  重复升件炸起草拍（T-0393 链；T-0450/0453/0454/0449/0491 dep_pair
  先例+T-0471/0472/0474 service 先例+T-0490 同根姊妹票同径）。
- 疑似根域勘误：票面起草猜域「评估×信号链面（fin_opp_eval×
  md_signal——T-0384 族沉默探针域）」系 T-0393 模板猜域——病灶本
  体在 hunger 面清单基线册（facesnap gate_domain 枚举族），与
  eval/signal 链零涉；sigdist 面零触不越域衍（同 T-0474/T-0490 勘误
  先例口径）。

### 实现摘要（枝 t/0492-face-gate-domain-meta）

- `facesnap.go` `faceBaselineWhitelist` 登记
  `face:gate_domain:meta`（注册式单键精确匹配零通配——册内面差集
  照落「面清单新面」行携理由键留账但升件免；重种/库重建回摆不
  重复炸票/起草，T-0393 链断点同 T-0448/0450 族先例）。
- `depblind_test.go` 先红桩
  `TestFaceBaselineGateDomainMetaRegistered`（T-0388 修复类先红
  桩律：病灶存在断言修前红+同断言修后绿——签名面必须在册且理由
  键非空，注册式不静默）。
- 探针阈/签名面/类册机制本体零改动（红线③守——只修病灶本体）。

### 验收锚对账

- ① 桩测断言全绿（红/绿双点）: PASS——红点=`go test -run
  TestFaceBaselineGateDomainMetaRegistered`（入册前）→
  `depblind_test.go:792: 病灶存在：face:gate_domain:meta 未入人工
  基线册——窗摆回将重复升件（T-0492）` FAIL；绿点=同断言入册后
  PASS（`ok o/internal/portfolio/hunger`，TestFaceBaseline* 九桩
  同跑全绿）。豁免机制面另有 TestProbeFaceDiffBaselineExempt
  （T-0448 册语义桩）同拍护。
- ② 探针同窗不再报同签名病灶: PASS（机制面）——面键入册后
  reportNewFace 豁免径生效（升件免/「面清单新面」行照落携基线册
  理由键）；生产谱态亲证：meta 域已入快照批 14/15（机制自愈在案
  ——册=簿记基线位）；若 meta.* 键重种/库重建致面出入摆回，册内
  面新面行照落携「基线册=」理由键但升件免——同签名升件/起草链
  断（depblindDedupHours 同窗去重口径不变，探针阈/签名面/类册机
  制本体零改动）。产线行随停=决策席合并+发窗后随拍生效（遵令未
  部署）。
- ③ check_all 绿戳对口: 见机检段（收口 commit 后 check_all 实跑
  落戳——收口终盖恒本盒律）。

### 机检

`gofmt -l` 净✓ `go build ./internal/portfolio/hunger/`✓
`go vet ./internal/portfolio/hunger/`✓｜`go test
./internal/portfolio/hunger/`（O_TEST_PG_DSN=testdb 租约 5434）
ok 49.043s 全谱✓｜check_all：收口 commit 后实跑（戳
`.git/check-green-<sha>` 落公共仓戳面，pre-push 验）。

### 自检 A–I

A 复用现有册件零新造（faceBaselineWhitelist=T-0448 既有登记位）✓；
B 直击病灶本体（根因=合法新面未入册非探针错）✓；C 零死码/TODO✓；
D 红桩断言修前红修后绿+对照面=册外面照升
（TestProbeFaceDiffBaselineExempt）✓；E 红线逐条：只碰本票签名
面/未部署未并/探针阈签名面类册零动/钱径凭据零碰✓；F 票面回填同
步✓；G 三锚逐条上对账✓；H 生产亲证五径：ops_event #4042 新面行+
#4043 升件行在案（psql 亲读）/app_config meta.autoapprove.registry
在册亲读（册值含 观测盲:* 钦定径）/ops_face_snapshot 批 14/15
gate_domain:meta 在册亲读/dm_autoapprove.sh 三处实读+depenum
meta 册载亲读/fin_0143+fin_0149 迁移 commit 在案——非听闻✓；
I 非展示值面 N/A✓。

### 涉及面与范围外确认

实际触碰：`internal/portfolio/hunger/facesnap.go`
（faceBaselineWhitelist +1 登记）、`internal/portfolio/hunger/
depblind_test.go`（先红桩 +1 测试）、本票面。钦定径
`internal/research/sigdist` 未触——根因不在信号蒸馏域（eval/
signal 链与本面零涉——票面疑似根域系起草模板猜域已勘误），册内
候选径为白名单非必触清单。范围外：零——未碰探针阈/签名面/类册
（faceInventory 枚举谓词/depConfigReaders/depTablePairs/
depSignalFamilies/depBlindWhitelist 原样）、未碰其他票面、未碰
simdrill/exec/钱径/凭据面。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——本票仅新增 `face:gate_domain:meta`
  一键；生产近 14d「观测盲 face:*」升件谱逐键对拍：其余各键
  （service×15+dep_pair×8+table×5+gate_domain 余量）零豁免面照
  升件等各票域裁决（T-0490~0497 族在案候派，本票不越权代判）。
- gate_domain 面类同型面核查：app_config 键域谱枚举全域 24 域
  （bridge/calendar/cmc/data/dm/earn/exec/feishu/gate/guard/
  invest/ledger/lifecycle/meta/mt5/net/notify/okx/readcred/
  research/risk/scout/trx）——meta 为唯一未册域（余域皆上拍快照
  在册旧面），`meta.*` 后续新键同映 face=meta 已被本键罩盖；
  `config:*→∅` orphan 形由 depConfigReaders 册外判定独立路径，
  本票登记不触 orphan 升件径（TestProbeDepsConfigOrphanDomain
  桩在案零误伤）。
- 豁免不误伤断言在案：TestProbeFaceDiffBaselineExempt 册内面升
  件免+册外面照升对照面同拍绿。

### 偏离/留案

- **范围外发现（不动手呈裁）**：`face:gate_domain:meta` 与 T-0490
  dep_pair 面同根（meta 域首现 app_config 一事件双面投影——
  gate_domain 枚举=split_part(key,'.',1)、dep_pair config 族=同
  键域×读者册两独立面类）——姊妹票各在其域，本票仅登记
  gate_domain 面键不代判 dep_pair 面（T-0490 已另册在飞）。
- **同族 merge 提示**：T-0490/T-0493~T-0497 同 facesnap.go/
  depblind_test.go 尾插在飞——合后冲突解=各条登记+各先红桩全保
  （T-0454/T-0471/T-0474 收口裁记同式先例）。
- 合法性锚=fin_0143 迁移册本体在案+depConfigReaders 册载读者在案
  +dm_autoapprove.sh 三处实读消费链活体+本票 #4373 机批实证+
  写域 fail-closed owner 归属设计在案（产消双端在册=合法新面，
  非幽灵面非断链面）。
