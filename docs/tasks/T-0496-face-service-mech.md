# T-0496 探针升件自动起草：观测盲@face:service:MECH

- 状态：收口待审（施工收口——回决策席常径裁）
- 领取：devin-o-worker-T-0496 @ 2026-11-09（漏拉/限流补派续跑 session，ops_event #5129）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:service:MECH`，草案锚 fin_ticket_draft#39）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:service:MECH`（kind=探针类 `观测盲`，key=病灶键 `face:service:MECH`）

末行升件原文：
> 面清单新面 face:service:MECH：机器可列面 service 类新增 MECH——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#4757

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:service:MECH' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:service:MECH（T-0455 册内候选，DM 裁时可改判）；
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

- `MECH`=**GW-001 内循环机械轨席位**——非幽灵席：业主令 2026-11-08
  钦定席位与例外授权（`docs/infra/GW-001-内循环机械轨-o-mech.md`
  文头留证防翻案件在案）；机械轨=案→草案→裁→票→已批→派单→拉起→
  收口→裁决→merge→绿戳→push→部署→闭环回写案卷全链机械活唯一
  执行者（o-mech.service/timer 5min 兜底拍+事件帧双径，单例 PG
  advisory lock 见 mech_tick.sh）。
- ops_event 写径四件全在册：`drift_probe.sh`（漂移行+升件经
  ops_esc.sh MECH）、`drift_converge.sh`（漂移消）、`verdict_reflow.sh`
  +`case_writeback.sh`（归流）；活性锚=ops_headless seat='MECH'
  心跳（gauges.tsv `mech_hb` 量表锚——last_activity 2026-11-08
  21:01:33Z alive 亲读）。
- 合法性锚：ceo_queue.sh 注释钦定「探针/壳轨机械升件留 CTO 壳轨」
  ——MECH 属壳轨机械席设计在案（其 ops_event 行=机械轨自报非席位
  上行件，故不入 CEO 待裁席位集=设计内非漏列）。
- 生产账：seat='MECH' ops_event 共 **92 行**，集中
  2026-11-08 16:02:28Z→20:29:03Z——GW-001 部署后 drift_probe 首跑
  落漂移行→converge 批量漂移消→reflow/writeback 归流；事件谱
  归流44/漂移消38/漂移6/升件4（升件4行=ticket键卫生拒写落账，
  T-0499 域在案非本票面）。
- 新面成因链：service 面=ops_event 近 7d `DISTINCT seat` 活跃投影
  （facesnap.go 设计口径「写者席即服务面投影」）——MECH 16:02Z
  首行入集→上拍批 13 册外→「面清单新面」行 #4756→观测盲升件
  #4757→draft#39（#4761）→#4781 CTO 起草物料化→#4783 CEO 机批已批
  →#4785 派单→#5129 漏拉/限流补派拉起本 run。
- 病灶本体=T-0448/T-0471/T-0472/T-0473/T-0474 同型：合法间歇写席
  **无持久基线位**——机械轨「有活才写」静默 >7d 出集→再活跃=再现
  新面→再升件再起草无限循环（「该建未建」残余差集的册外盲区——
  面宇宙人工过基线后新增才值得看，间歇面需要登记位续判）。
- 快照自愈在案：face:service:MECH 已入 ops_face_snapshot 批 14/15
  ——册=窗摆期保险丝非现报抑制（同 T-0473 语义）。

### 实现摘要（枝 t/0496-face-service-mech）

- `facesnap.go` `faceBaselineWhitelist` 登记 `face:service:MECH`
  （注册式单键精确匹配零通配——册内面差集照落「面清单新面」行携
  理由键留账但升件免；窗摆回不重复炸票/起草，T-0393 链断点同
  T-0448/0450/0453/0454/0471/0472/0473/0474/0490/0492/0493 先例）。
- `depblind_test.go` 先红桩 `TestFaceBaselineServiceMECHRegistered`
  （T-0388 修复类先红桩律：病灶存在断言修前红+同断言修后绿——签名
  面必须在册且理由键非空，注册式不静默）。
- sigdist 面零触——疑似根域（评估×信号链面 fin_opp_eval×md_signal）
  系起草拍自动猜域，病灶本体在 hunger 面清单基线册（票面面内
  depblind*/facesnap* 同族径），不越域衍。

### 测试证据

- 先红桩双点：`TestFaceBaselineServiceMECHRegistered` 修前红
  「病灶存在：face:service:MECH 未入人工基线册——窗摆回将重复升件
  （T-0496）」→入册后同断言绿；`TestFaceBaseline*` 十三先红桩
  （T-0450/0453/0454/0491/0449/0471/0474/0472/0490/0492/0493/0473
  +本票）同跑全绿。
- `gofmt`/`go build`/`go vet` 净；check_all 分段全绿（mode=scoped）
  ——绿戳落本枝 HEAD commit（check-green-<SHA> 对口）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——生产近 7d「观测盲」升件 face:* 谱
  29 键（service×16+dep_pair×7+table×5+gate_domain×1+其他）与册
  零交集，册外面照升件零误伤；册内各键均对应己票已处理签名
  （本票 MECH 仅 #4757 一报），登记不新增抑制面。
- service 册外新面（判例闸/起草拍/跑席/裁定席/演习/外审/O-Dev交互席/
  业主/配管/--take 等）与 table 册外面（fin_exec_drill/
  fin_guard_suspend/fin_symbol_dialect/md_eval_pulse）归各自票域
  不越权代判（详见偏离/留案——T-0494/T-0497 兄弟票在案）。
- MECH 已入快照批 14/15（机制自愈在案）——册=窗摆期保险丝非现报
  抑制；面清单新面行照落留账（每 >7d 静默窗摆至多一行，drafter
  只吃升件不误起草）。
- 验收锚②「探针同窗不再报同签名病灶」：修复后若 MECH 静默出集
  摆回，册内面新面行照落携「基线册=」理由键但升件免——同签名
  升件/起草链断（depblindDedupHours 同窗去重口径不变，探针阈/
  签名面/类册机制本体零改动）。

### 偏离/留案

- 兄弟面票各有己域不代判：T-0494（dep_pair leg）/T-0497
  （face:table:md_eval_pulse）票件在案；册外 service 面余键
  （判例闸/起草拍/跑席/裁定席/演习/外审/O-Dev交互席/业主/配管/
  --take）与 table 余面合法性核+入册归后续票域，本票不代判
  （facesnap.go 同图邻接行 merge 冲突解=条目并集全保，T-0447/
  T-0453/T-0454/T-0471/T-0472/T-0473/T-0474/T-0490/T-0492/T-0493
  合入先例在案）。
- `face:service:--take` 孤例留案：seat='--take' 单行 @2026-11-07
  12:34Z（领号 T-0440 伴生）——疑 CLI 参数误作 seat 写径的键卫生
  面病灶，与 T-0499 ticket键卫生同族嫌疑；范围外不代修，呈裁量。
- MECH 席位 ops_event「升件」4 行=ticket键卫生拒写落账（T-0499
  域在案）非面册病灶；MECH 写径本身合法在册（drift_*/verdict_
  reflow/case_writeback 四脚本亲读）。
- 疑似根域（评估×信号链面 fin_opp_eval×md_signal）系起草拍
  自动猜域——病灶本体在 hunger 面清单基线册（service 面枚举径），
  范围内修域正确不越域衍。
