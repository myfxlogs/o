# T-0505 探针升件自动起草：观测盲@face:service:研究员哨兵

- 状态：收口待审（施工收口——回决策席常径裁）
- 领取：devin-o-worker-T-0505 @ 2026-11-09（ops_event #5374 领单行）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:service:研究员哨兵`，草案锚 fin_ticket_draft#43）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:service:研究员哨兵`（kind=探针类 `观测盲`，key=病灶键 `face:service:研究员哨兵`）

末行升件原文：
> 面清单新面 face:service:研究员哨兵：机器可列面 service 类新增 研究员哨兵——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#5357

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:service:研究员哨兵' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:service:研究员哨兵（T-0455 册内候选，DM 裁时可改判）；
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

- `研究员哨兵`=**T-0504 研究员触发消费拍席位**（方案A CTO 钦定，
  merge d167108af/票翻完成 4809189d2 在案）——`scripts/meta/
  researcher_tick.sh` 滤队消费 venue哨兵 `新品`/`达阈` 未终态行→
  paused 闸→systemd-run 瞬态拉起 `o-researcher-evt-*`→rc=0 逐事件
  落 `消`行 `seat='研究员哨兵'` `note='触发#<id> 已拉起 unit=…'`
  （钦定③）；`deploy/o-researcher-tick.{service,timer}` 30min 拍
  部署件在案。ops_event 写径=触发消费落账合法非幽灵席。
- 病灶行谱：seat='研究员哨兵' ops_event 全库**仅 2 行** #5345/
  #5346（2026-11-08 22:57:46Z，event=消 ticket=META-VENUEWATCH，
  note=「触发#1883/#1879 合成演练清场——T-0504 收口（2026-11-04
  VENUEWATCH_SYNTH_ANN 演练残留，非真实新品，不拉会话）」）——
  收口清场消费两条合成演练触发行的合法消行，首写=入集时点。
- 新面成因链：service 面=ops_event 近 7d `DISTINCT seat` 活跃
  投影（facesnap.go 设计口径「写者席即服务面投影」）——消行
  @22:57:46Z 首行入集→上拍批册外→「面清单新面」行 #5356
  （23:08:20Z）→观测盲升件 #5357 同窗→#5363 领号→#5367 CTO
  起草物料化→#5368 CEO 机批已批→#5373 CTO 派单→#5374 本 run
  领单。探针只报不修=设计内升件，合法性人工过基线=本票处置径。
- 病灶本体=T-0448/T-0471~T-0503 同型族变体：合法新面**无持久
  基线位**——研究员哨兵 ops_event 写仅在有 pending 触发消费时
  （空拍=hb ops_headless 零 ops_event 写，venue_watch 空拍同构），
  行龄出 7d 活跃窗摆出后任一触发再消费即再现新面再升件再起草
  循环（T-0393 链）；册=窗摆期保险丝非现报抑制（同 T-0473/0496
  语义，研究员哨兵已入 ops_face_snapshot 批 18=机制自愈在案）。
- 类属判定：合法在册写径面**非可疑新面**——写者/写径/时点/
  票源全谱亲证闭合（T-0504 钦定部署+收口清场语义+#5345/#5346
  行谱对拍），人工过基线结论=登记位续判不炸票。

### 实现摘要（枝 t/0505-face-service）

- `facesnap_whitelist.go` `faceBaselineWhitelist` 登记
  `face:service:研究员哨兵`（注册式单键精确匹配零通配——册内面
  差集照落「面清单新面」行携理由键留账但升件免；窗摆回不重复
  炸票/起草，T-0393 链断点同 T-0448/0450/0453/0454/0471/0472/
  0473/0474/0490/0492/0493/0494/0496/0497/0503 先例）。
- `depblind_test.go` 先红桩 `TestFaceBaselineService研究员哨兵
  Registered`（T-0388 修复类先红桩律：病灶存在断言修前红+同
  断言修后绿——签名面必须在册且理由键非空，注册式不静默）。
- sigdist 面零触——疑似根域（评估×信号链面 fin_opp_eval×
  md_signal，T-0384 族沉默探针域）系起草拍自动猜域，病灶本体
  在 hunger 面清单基线册（票面面内 depblind*/facesnap* 同族径），
  不越域衍。

### 测试证据

- 先红桩双点：`TestFaceBaselineService研究员哨兵Registered`
  修前红「病灶存在：face:service:研究员哨兵 未入人工基线册——
  窗摆回将重复升件（T-0505）」→入册后同断言绿；`TestFaceBaseline*`
  十七先红桩同跑全绿（`go test ./internal/portfolio/hunger` 全包
  绿）。
- `gofmt`/`go build`/`go vet` 净；`check_migrations` 七断言过；
  check_all 绿戳落本枝 HEAD commit（check-green-<SHA> 对口，
  mode 见戳 .mode 留痕）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——本票仅抑制 `face:service:研究员哨兵`
  一签名（生产近 14d「观测盲」升件 face:* 谱中本签名仅 #5357
  一报）；册外面照升件零误伤（depblind/facesnap 机制本体零改动）。
- service 册外新面（判例闸/起草拍/跑席/裁定席/演习/外审/O-Dev
  交互席/业主/配管/--take 等 T-0503 审计在案谱）归各自票域不越
  权代判。
- 研究员哨兵已入快照批 18（机制自愈在案）——册=窗摆期保险丝
  非现报抑制；面清单新面行照落留账（每 >7d 静默窗摆至多一行，
  drafter 只吃升件不误起草）。
- 验收锚②「探针同窗不再报同签名病灶」：修复后若研究员哨兵摆出
  再摆回，册内面新面行照落携「基线册=」理由键但升件免——同
  签名升件/起草链断（depblindDedupHours 同窗去重口径不变，
  探针阈/签名面/类册机制本体零改动，红线④守）。

### 偏离/留案

- `researcher_run.sh` 导出 `O_SEAT=Researcher`——真实触发会话
  若经 report.sh 自报将产 `face:service:Researcher` 新面（与本
  票 `研究员哨兵` 消费拍席位不同键）；该面尚未在 ops_event 出现
  （全库仅 #5345/#5346 消行在案），不预登记越签名面——若日后
  出现走同径升件→起草→机批→入册链。
- 疑似根域（评估×信号链面 fin_opp_eval×md_signal）系起草拍
  自动猜域——病灶本体在 hunger 面清单基线册（service 面枚举径），
  范围内修域正确不越域衍。
