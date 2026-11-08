# T-0503 探针升件自动起草：观测盲@face:service:TEST

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·取证=收口并master·机械轨票翻）
- 领取：devin-o-worker-T-0503 @ 2026-11-09（ops_event #5259 领单行）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:service:TEST`，草案锚 fin_ticket_draft#42）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:service:TEST`（kind=探针类 `观测盲`，key=病灶键 `face:service:TEST`）

末行升件原文：
> 面清单新面 face:service:TEST：机器可列面 service 类新增 TEST——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#5223

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:service:TEST' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:service:TEST（T-0455 册内候选，DM 裁时可改判）；
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

- `TEST`=**T-0499 ticket键卫生根修施工 run 的实弹验证标记席**——非
  幽灵席非在册席位：T-0499 施工席领单 ops_event #4990（19:53:37Z）
  →驳回返修1 #5078（20:28:45Z）施工窗内，对
  `scripts/meta/ops_event_key_gate.sh` 伪键拒写径做生产实测——
  第三参 `SEAT` 显式传 `'TEST'` 作标记件（同脚本缺省 `MECH`）。
- 病灶行谱：seat='TEST' ops_event 全库**仅 1 行** #5004
  （2026-11-08 20:05:33.59Z，event=升件 ticket='' note=
  「ticket键卫生拒写 event=领单 h=95204335——key=T-9001\nT-9002
  （写面落库前拒…T-0499）」）；伴生 #5005-#5008 同 burst 四连
  MECH 默认席拒写行（ctx=event=x，键分隔符 ·/，/|/空格 变体谱，
  20:05:33-39Z 六秒窗）
  ——单发测试件产物面，写径=键闸 loud 拒写末环尽力落账合法
  （脚本注释钦定「闸自身写径全尽力而为」）。
- 新面成因链：service 面=ops_event 近 7d `DISTINCT seat` 活跃投影
  （facesnap.go 设计口径「写者席即服务面投影」）——TEST @20:05Z
  首行入集→上拍批册外→「面清单新面」行 #5222→观测盲升件 #5223
  （22:03:12Z）→draft#42（#5224）→#5251 CTO 起草物料化→#5252
  CEO 机批已批→#5259 本 run 领单。
- 病灶本体=T-0448/T-0471~T-0497 同型族变体：合法/已知面**无持久
  基线位**——TEST 行龄出 7d 活跃窗（~2026-11-15 20:05Z）摆出，
  任一键闸复测再写 seat='TEST' 即再现新面再升件再起草循环
  （T-0393 链）；册=窗摆期保险丝非现报抑制（同 T-0473/0496 语义，
  TEST 已入 ops_face_snapshot 批 16=机制自愈在案）。
- 类属判定：测试件产物面**非可疑新面**——写者/写径/时点/伴生行
  全谱亲证闭合（T-0499 施工窗+键闸拒写语义+h=95204335 键摘要
  对拍），人工过基线结论=登记位续判不炸票。

### 实现摘要（枝 t/0503-face-service-test）

- `facesnap_whitelist.go` `faceBaselineWhitelist` 登记
  `face:service:TEST`（注册式单键精确匹配零通配——册内面差集照落
  「面清单新面」行携理由键留账但升件免；窗摆回不重复炸票/起草，
  T-0393 链断点同 T-0448/0450/0453/0454/0471/0472/0473/0474/0490/
  0492/0493/0494/0496/0497 先例）。
- `depblind_test.go` 先红桩 `TestFaceBaselineServiceTESTRegistered`
  （T-0388 修复类先红桩律：病灶存在断言修前红+同断言修后绿——
  签名面必须在册且理由键非空，注册式不静默）。
- sigdist 面零触——疑似根域（评估×信号链面 fin_opp_eval×md_signal）
  系起草拍自动猜域，病灶本体在 hunger 面清单基线册（票面面内
  depblind*/facesnap* 同族径），不越域衍。

### 测试证据

- 先红桩双点：`TestFaceBaselineServiceTESTRegistered` 修前红
  「病灶存在：face:service:TEST 未入人工基线册——窗摆回将重复升件
  （T-0503）」→入册后同断言绿；`TestFaceBaseline*` 十六先红桩
  同跑全绿。
- `gofmt`/`go build`/`go vet` 净；`go test ./internal/portfolio/hunger`
  全包绿；check_all 绿戳落本枝 HEAD commit（check-green-<SHA> 对口，
  mode 见戳 .mode 留痕）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——生产近 14d「观测盲」升件 face:*
  谱 30 键与册零交集新增抑制面（册内各键均对应己票已处理签名，
  本票 TEST 仅 #5223 一报）；册外面照升件零误伤。
- service 册外新面（判例闸/起草拍/跑席/裁定席/演习/外审/O-Dev交互席/
  业主/配管/--take）与 table 册外面（fin_exec_drill/fin_guard_suspend/
  fin_symbol_dialect）归各自票域不越权代判（T-0496 留案在案，本票
  不代判）。
- TEST 已入快照批 16（机制自愈在案）——册=窗摆期保险丝非现报
  抑制；面清单新面行照落留账（每 >7d 静默窗摆至多一行，drafter
  只吃升件不误起草）。
- 验收锚②「探针同窗不再报同签名病灶」：修复后若 TEST 摆出再摆回，
  册内面新面行照落携「基线册=」理由键但升件免——同签名升件/起草
  链断（depblindDedupHours 同窗去重口径不变，探针阈/签名面/类册
  机制本体零改动）。

### 偏离/留案

- `face:service:--take` 孤例（T-0496 留案在案）与本票 TEST 同族
  ——均系键卫生/测试写径产物面；类级根治候选=键闸 SEAT 参域约束
  （白名单席位集/测试席固定径）或 service 面枚举对测试标记席豁免，
  均越本票签名面呈裁量不代修。
- 疑似根域（评估×信号链面 fin_opp_eval×md_signal）系起草拍
  自动猜域——病灶本体在 hunger 面清单基线册（service 面枚举径），
  范围内修域正确不越域衍。
