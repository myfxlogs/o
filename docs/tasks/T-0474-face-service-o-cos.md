# T-0474 探针升件自动起草：观测盲@face:service:席O-COS

- 状态：收口待审（施工收口——回决策席常径裁）
- 领取：devin-o-worker-T-0474 @ 2026-11-09
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:service:席O-COS`，草案锚 fin_ticket_draft#25）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:service:席O-COS`（kind=探针类 `观测盲`，key=病灶键 `face:service:席O-COS`）

末行升件原文：
> 面清单新面 face:service:席O-COS：机器可列面 service 类新增 席O-COS——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#3931

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:service:席O-COS' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:service:席O-COS（T-0455 册内候选，DM 裁时可改判）；
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

- 席O-COS=COS 秘书席交互席位本体——业主令 2026-11-04 立席（「COS 席=
  Chief of Staff 秘书席——业主办公室：能讨论能派单。纯交互席」），
  O-COS tmux 会话（2026-11-03 建）内 devin 会话在跑；合法席位非幽灵。
- ops_event 写径=report.sh tmux 回退径：O-COS 会话内无 O_SEAT、席位
  映射表仅 `席DM|席O-DM→COS`、`席Dev|席O-Dev→施工席` 无 席O-COS 条目
  →WHO='席O-COS'→seat='席O-COS'；事件谱=移交/审计中/外派/呈批/收口/
  裁决/通过/处置/派单/完成 十类 COS 域工作流事件（含「业主授权COS
  直裁」自报行 relay 在案）。
- 生产账：seat='席O-COS' ops_event 首行 2026-11-08 00:18:12Z，至
  19:26Z 共 49 行/10 类事件；升件源链=#3930 面清单新面行→#3931
  升件→draft#25→#4358 起草→#4371 已批（CEO 机批 T-0455/0456 批权
  归径）→#4415 CTO 派单→本票。
- 新面成因链：service 面=ops_event 近 7d `DISTINCT seat` 活跃投影
  （facesnap.go 设计口径「写者席即服务面投影」）——COS 席 11-08 起在
  O-COS 会话自报入集→上拍批册外→「面清单新面」行+观测盲升件。
  席O-COS 已入快照批 14/15（机制自愈在案——册=窗摆期保险丝）。
- 病灶本体=T-0448 同型：合法间歇写席**无持久基线位**——交互席队空
  零写，静默 >7d 出集→再活跃=再现新面→再升件再起草无限循环
  （「该建未建」残余差集的册外盲区——面宇宙人工过基线后新增才值得
  看，间歇面需要登记位续判；T-0471 CEO 同签名族先例同径）。

### 实现摘要（枝 t/0474-face-service-o-cos）

- `facesnap.go` `faceBaselineWhitelist` 登记 `face:service:席O-COS`
  （注册式单键精确匹配零通配——册内面差集照落「面清单新面」行携理由
  键留账但升件免；窗摆回不重复炸票/起草，T-0393 链断点同
  T-0448/0450/0471 先例）。
- `depblind_test.go` 先红桩 `TestFaceBaselineServiceSeatOCOSRegistered`
  （T-0388 修复类先红桩律：病灶存在断言修前红+同断言修后绿——签名面
  必须在册且理由键非空，注册式不静默）。
- sigdist 面零触——疑似根域（评估×信号链面）系起草拍自动猜域，
  病灶本体在 hunger 面清单基线册（票面面内 depblind*/facesnap*
  同族径），不越域衍。

### 测试证据

- 先红桩双点：`TestFaceBaselineServiceSeatOCOSRegistered` 修前 FAIL
  「病灶存在：face:service:席O-COS 未入人工基线册——窗摆回将重复
  升件（T-0474）」→入册后同断言 PASS；`TestFaceBaseline*` 七先红桩
  （T-0449/0450/0453/0454/0491/0471+本票）同跑全绿。
- `gofmt`/`go build`/`go vet` 净；`go test ./internal/portfolio/hunger/`
  绿（DB 桩测谱含 TestProbeFaceDiffBaselineExempt 豁免对照面——
  check_all 租约 O_TEST_PG_DSN 口径同跑）。
- check_all 分段全绿（mode=scoped）：绿戳落本枝 HEAD commit
  （check-green-<SHA> 对口）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——本票仅新增 `face:service:席O-COS` 一键，
  对生产近 3d「观测盲 face:*」升件谱其余 28 签名（service×15+
  dep_pair×7+table×5+gate_domain×1）零豁免面，各键照升件等各票域
  裁决（MECH/跑席/裁定席/演习/判例闸/起草拍/COS上行/外审席 等在案
  候派 T-0472/0473 族，本票不越权代判）。
- 席O-COS 已入快照批 14/15（机制自愈在案）——册=窗摆期保险丝非现报
  抑制；面清单新面行照落留账（每 >7d 静默窗摆至多一行，drafter 只吃
  升件不误起草）。
- 验收锚②「探针同窗不再报同签名病灶」：修复后若席O-COS 静默出集
  摆回，册内面新面行照落携「基线册=」理由键但升件免——同签名
  升件/起草链断（depblindDedupHours 同窗去重口径不变，探针阈/
  签名面/类册机制本体零改动）。

### 偏离/留案

- 兄弟票 T-0472（face:service:COS上行）/T-0473（外审席）同签名族在案
  候派——各面合法性核+入册归各票域，本票不代判（facesnap.go 同图
  邻接行 merge 冲突解=条目并集全保，T-0471 先例在案）。
- **范围外发现（不动手呈裁）**：席位名分裂——COS 秘书席自报落
  seat='席O-COS' 与既有 seat='COS'（11-04 起 209 行）双名并存；
  根在宿主面 `~/bin/report.sh` 席位映射表缺 `席O-COS→COS` 条目
  （映射现仅 席DM|席O-DM→COS、席Dev|席O-Dev→施工席）。report.sh
  非本仓件且在票面钦定文件径外——归宿主面/决策席裁是否补映射或
  令 O-COS 会话设 O_SEAT；本票册内登记不依赖该裁决（service 面=写者
  席投影，名分裂不影响合法性判定，理由键已载双名事实留账）。
- 合法性锚=业主令 2026-11-04 立席+O-COS tmux 会话在跑+report.sh
  写径十类 COS 域事件+ops_event 49 行生产实证（COS 席非
  fin_employee_registry 在册员工——该册=员工域非席位域，同 T-0471
  CEO 先例口径）。
