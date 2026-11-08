# T-0472 探针升件自动起草：观测盲@face:service:COS上行

- 状态：收口待审（施工收口——回决策席常径裁）
- 领取：devin-o-worker-T-0472 @ 2026-11-09（补派续跑 session）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:service:COS上行`，草案锚 fin_ticket_draft#23）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:service:COS上行`（kind=探针类 `观测盲`，key=病灶键 `face:service:COS上行`）

末行升件原文：
> 面清单新面 face:service:COS上行：机器可列面 service 类新增 COS上行——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#3927

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:service:COS上行' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:service:COS上行（T-0455 册内候选，DM 裁时可改判）；
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

- `COS上行`=COS 秘书席**上行写径席位身份**——非幽灵席：O-DM pane
  （席O-COS 交互席本体）业主在席处置窗以 seat='COS上行' 落
  ops_event（呈批/移交/领号/升件/挂起/收口待审/派单等自报与台账
  行）。合法性双锚：①T-0456 业主令双无头席建置（commit 285825a43，
  [角色:COS]）`scripts/meta/ceo_queue.sh` 钦定席位集显式含
  `'COS上行'`——「席位上行件归 CEO 方向层」设计在案（同集另有
  'CEO上行'/'管理席' 同式上行身份）；②COS 职卡写面白名单
  （skills/COS/SKILL.md）上行通道=ops_event 行+飞书径。
- 生产账：seat='COS上行' ops_event 共 26 行，集中
  2026-11-07 23:05Z→11-08 01:07Z（业主令密集处置窗——T-0455/0456
  移交裁、T-0443~0454 草案呈批批、T-0458/0459 移交、幽灵票升件、
  META-DM 挂起令、T-0145 收口待审）；领号行证 `next_ticket.sh
  COS上行` 取号径同席名。此后零写=间歇面实证。
- 新面成因链：service 面=ops_event 近 7d `DISTINCT seat` 活跃投影
  （facesnap.go 设计口径「写者席即服务面投影」）——COS上行 11-07/08
  写入入集→上拍批 13 册外→「面清单新面」行 #3926→观测盲升件
  #3927→draft#23→#4356 起草→#4368 CEO 机批（T-0455/0456 批权归径）
  →#4409 派单→本票。
- 病灶本体=T-0471/T-0448 同型：合法间歇写席**无持久基线位**——
  COS 上行写径随业主在席节奏间歇（「业主在才有秘书之义」纯交互席
  形态钉死），静默 >7d 出集→再活跃=再现新面→再升件再起草无限
  循环（「该建未建」残余差集的册外盲区——面宇宙人工过基线后新增
  才值得看，间歇面需要登记位续判）。

### 实现摘要（枝 t/0472-face-service-cos）

- `facesnap.go` `faceBaselineWhitelist` 登记
  `face:service:COS上行`（注册式单键精确匹配零通配——册内面差集照落
  「面清单新面」行携理由键留账但升件免；窗摆回不重复炸票/起草，
  T-0393 链断点同 T-0448/0450/0453/0454/0471 先例）。
- `depblind_test.go` 先红桩
  `TestFaceBaselineServiceCOS上行Registered`（T-0388 修复类先红桩律：
  病灶存在断言修前红+同断言修后绿——签名面必须在册且理由键非空，
  注册式不静默）。
- sigdist 面零触——疑似根域（评估×信号链面）系起草拍自动猜域，
  病灶本体在 hunger 面清单基线册（票面面内 depblind*/facesnap*
  同族径），不越域衍。

### 测试证据

- 先红桩双点：`TestFaceBaselineServiceCOS上行Registered` 修前红
  「病灶存在：face:service:COS上行 未入人工基线册——窗摆回将重
  复升件（T-0472）」→入册后同断言绿；`TestFaceBaseline*` 七先红桩
  （T-0450/0453/0454/0491/0471/0449+本票）同跑全绿。
- `gofmt`/`go build`/`go vet` 净；`go test
  ./internal/portfolio/hunger/` 全包绿 48.6s（DB 桩测谱含
  TestProbeFaceDiffBaselineExempt 豁免对照面全过——check_all
  租约 O_TEST_PG_DSN 口径同跑）。
- check_all 分段全绿（mode=scoped）：build/vet/import/gofmt/
  env-reads/rejects/config-write-domain/migrations/bridge-audit/
  tolerate-report/ops-chain/seat-writes + scoped 定向 go test
  谱全 ok——绿戳落本枝 HEAD commit（check-green-<SHA> 对口）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——生产近 3d「观测盲」升件 face:* 谱
  （service×11+dep_pair×5+table×3+gate_domain×1 各键）与册零交集，
  册外面照升件零误伤；service 册外新面（判例闸/起草拍/席O-COS/
  外审席/MECH/跑席/裁定席/演习/外审 等）各有己票域（T-0473/0474/
  0490~0497 在案）不越权代判。
- COS上行 已入快照批 14/15（机制自愈在案）——册=窗摆期保险丝非
  现报抑制；面清单新面行照落留账（每 >7d 静默窗摆至多一行，
  drafter 只吃升件不误起草）。
- 验收锚②「探针同窗不再报同签名病灶」：修复后若 COS上行 静默出
  集摆回，册内面新面行照落携「基线册=」理由键但升件免——同签名
  升件/起草链断（depblindDedupHours 同窗去重口径不变，探针阈/
  签名面/类册机制本体零改动）。

### 偏离/留案

- 兄弟票 T-0473（face:service:外审席）/T-0474（face:service:席O-COS）
  及 T-0490~0497 面族在案候派——各面合法性核+入册归各票域，本票
  不代判（facesnap.go 同图邻接行 merge 冲突解=条目并集全保，
  T-0447/T-0453/T-0454/T-0471 合入先例在案）。
- COS上行 写径非脚本自动件（O-DM pane 会话内 report.sh/next_ticket.sh
  席名实参）——合法性锚=ceo_queue.sh 钦定席位集设计在案+26 行生产
  实证+COS 职卡上行通道白名单，理由键已载；若后续 COS 侧统一席名
  归一（如全归 'COS'），本册键成死键无害（精确匹配不误伤他面），
  册面清理归后续面册治理票域。
