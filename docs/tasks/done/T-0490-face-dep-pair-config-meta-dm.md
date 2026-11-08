# T-0490 探针升件自动起草：观测盲@face:dep_pair:config:meta→DM 机批闸册

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·取证=收口并master·机械轨票翻）
- 领取：devin-o-worker-T-0490 @ 2026-11-09
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:dep_pair:config:meta→DM 机批闸册`，草案锚 fin_ticket_draft#28）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:dep_pair:config:meta→DM 机批闸册`（kind=探针类 `观测盲`，key=病灶键 `face:dep_pair:config:meta→DM 机批闸册`）

末行升件原文：
> 面清单新面 face:dep_pair:config:meta→DM 机批闸册：机器可列面 dep_pair 类新增 config:meta→DM 机批闸册——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#4037

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:dep_pair:config:meta→DM 机批闸册' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:dep_pair:config:meta→DM 机批闸册（T-0455 册内候选，DM 裁时可改判）；
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

- 新面本体=depenum ④族配置对 `config:meta→DM 机批闸册`——枚举源=
  `configDomains`（app_config 首段键域谱）×`depConfigReaders` 读面册
  （depenum.go 本体零硬编码）：meta 域在册（键存在=产者已写）且读者
  册载→对签名入列→faceInventory dep_pair 面行→上拍快照批册外→
  「面清单新面」行+观测盲升件（facesnap.go probeFaceDiff 设计径）。
- meta 域来源=fin_0143 迁移种 `meta.autoapprove.registry`（T-0455 册
  本体——签名→钦定文件径白名单，dm_autoapprove.sh 实读同键=候选票
  批权径；生产亲证 SG app_config 该键在册 updated_at
  2026-11-08 14:24:50+00、册值含 `观测盲:*`→本票钦定 2 径）。
- 读者册载=`depConfigReaders["meta"]="DM 机批闸册"`（T-0448 bcd706f95
  补册——同 commit 立 faceBaselineWhitelist 机制本体；注释钉
  「fin_0143 未入册同族」depblind 存量红根修）。
- 合法性核=**合法新面非「该建未建」残余**：对=册内对（→DM 机批闸册
  非 →∅ orphan——产者域在册+读者册载双在案）；dm_autoapprove.sh
  line 41/60/107 三处实读 meta.autoapprove.registry（消费链活体）；
  本票自身即该闸产物（#4373 已批机批行=闸在跑实证）。
- 升件源链=#4036 面清单新面行→#4037 升件→draft#28→#4048 起草记→
  #4361 自动起草落件→#4373 已批（机批 T-0455/0456 批权归径）→
  #4476 dedup skip→#5051 领单→#5052 施工中→本票。
- 新面成因链：fin_0143 迁移随部署窗落 SG app_config（meta 域首现）
  +hunger 探针面 depenum ④族枚举→上拍批册外→diff 报新面。面已入
  快照批 14/15（机制自愈在案——册=簿记基线位+重种/重建回摆保险丝）。
- 病灶本体=T-0448 同型族：合法新面**无持久基线位**——面宇宙人工过
  基线后新增才值得看，但合法新增面需登记位续判，册外/重种回摆重复
  升件炸起草拍（T-0393 链；T-0450/0453/0454/0449/0491 dep_pair 先例
  +T-0471/0472/0474 service 先例同径）。
- 疑似根域勘误：票面起草猜域「评估×信号链面（fin_opp_eval×
  md_signal）」系 T-0393 模板猜域——病灶本体在 hunger 面清单基线册
  （facesnap/depenum dep_pair config 族），与 eval/signal 链零涉；
  sigdist 面零触不越域衍（同 T-0474 勘误先例口径）。

### 实现摘要（枝 t/0490-face-dep-pair-config-meta-dm）

- `facesnap.go` `faceBaselineWhitelist` 登记
  `face:dep_pair:config:meta→DM 机批闸册`（注册式单键精确匹配零通配
  ——册内面差集照落「面清单新面」行携理由键留账但升件免；窗摆/重种
  回摆不重复炸票/起草，T-0393 链断点同 T-0448/0450 族先例）。
- `depblind_test.go` 先红桩
  `TestFaceBaselineDepPairConfigMetaDMRegistered`（T-0388 修复类先红
  桩律：病灶存在断言修前红+同断言修后绿——签名面必须在册且理由键
  非空，注册式不静默）。
- 探针阈/签名面/类册机制本体零改动（红线③守——只修病灶本体）。

### 测试证据

- 先红桩双点：`TestFaceBaselineDepPairConfigMetaDMRegistered` 修前
  FAIL「病灶存在：face:dep_pair:config:meta→DM 机批闸册 未入人工基
  线册——窗摆回将重复升件（T-0490）」→入册后同断言 PASS；
  `TestFaceBaseline*` 九先红桩（T-0449/0450/0453/0454/0491/0471/
  0472/0474+本票）同跑全绿。
- `gofmt -l` 净/`go build`/`go vet` 净；`go test
  ./internal/portfolio/hunger/` 全包绿 119s（O_TEST_PG_DSN=testdb
  租约口径——DB 桩测谱含 TestProbeFaceDiffBaselineExempt 豁免对照
  面+TestProbeDepsConfigOrphanDomain ④族死键检出面）。
- check_all 绿戳对口见收口段（rebase 后本盒盖戳——收口终盖恒本盒
  律）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——本票仅新增
  `face:dep_pair:config:meta→DM 机批闸册` 一键；生产近 14d「观测盲
  face:*」升件谱 29 签名逐键对拍：其余各键（service×15+dep_pair×
  8+table×5+gate_domain×1 余量）零豁免面照升件等各票域裁决
  （MECH/判例闸/外审/外审席/演习/裁定席/起草拍/跑席/COS上行/
  gate_domain:meta/leg:BTC-USDT-SWAP/table:ops_seat_card/
  table:md_eval_pulse/signal:basis_ann 族/配管/--take/业主/
  O-Dev交互席/fin_exec_drill/fin_guard_suspend/fin_symbol_dialect
  等在案候派 T-0492~0497 族，本票不越权代判）。
- dep_pair config 族同型面核查：app_config 键域谱枚举全域——meta
  为唯一新增域对（余域皆册内旧对在册）；`config:*→∅` orphan 形
  由 depConfigReaders 册外判定独立路径，本票登记不触 orphan 升件
  径（TestProbeDepsConfigOrphanDomain 桩在案零误伤）。
- 验收锚②「探针同窗不再报同签名病灶」：该面已入快照批 14/15
  （机制自愈——册=簿记基线位）；若 meta 域键重种/库重建致面出
  入摆回，册内面新面行照落携「基线册=」理由键但升件免——同签名
  升件/起草链断（depblindDedupHours 同窗去重口径不变，探针阈/
  签名面/类册机制本体零改动）。

### 偏离/留案

- 兄弟票同批族在案候派——T-0492（face:gate_domain:meta，同 meta
  域姊妹面）/T-0493（table:ops_seat_card）/T-0494
  （leg:BTC-USDT-SWAP）/T-0495（signal:basis_ann）/T-0496
  （service:MECH）/T-0497（table:md_eval_pulse）各面合法性核+入册
  归各票域，本票不代判（facesnap.go 同图邻接行 merge 冲突解=条目
  并集全保，T-0471/0474 先例在案）。
- **范围外发现（不动手呈裁）**：`face:gate_domain:meta` 与本票
  dep_pair 面同根（meta 域首现 app_config 一事件双面投影——
  gate_domain 枚举=同 `split_part(key,'.',1)` 谓词独立面类）——
  姊妹票 T-0492 域内裁，本票仅登记 dep_pair 面键不代判。
- 合法性锚=fin_0143 迁移册本体在案+depConfigReaders 册载读者在案
  +dm_autoapprove.sh 三处实读消费链活体+本票 #4373 机批实证
  （产消对双端在册=合法新面，非幽灵面非断链面）。
