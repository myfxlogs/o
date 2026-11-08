# T-0493 探针升件自动起草：观测盲@face:table:ops_seat_card

- 状态：收口待审（施工收口——回决策席常径裁）
- 领取：devin-o-worker-T-0493 @ 2026-11-09
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:table:ops_seat_card`，草案锚 fin_ticket_draft#32）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:table:ops_seat_card`（kind=探针类 `观测盲`，key=病灶键 `face:table:ops_seat_card`）

末行升件原文：
> 面清单新面 face:table:ops_seat_card：机器可列面 table 类新增 ops_seat_card——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#4047

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:table:ops_seat_card' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:table:ops_seat_card（T-0455 册内候选，DM 裁时可改判）；
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

- 新面本体=`ops_seat_card` public BASE TABLE——fin_0145 迁移建（T-0458
  飞书席位活卡：seat→message_id+chat_id+last_hash 卡册持久映射，展示面
  非真源真源=ops_event 链账；commit 29f6c5934 已合 master）。
- table 类面枚举=`information_schema.tables WHERE table_schema='public'
  AND table_type='BASE TABLE'` 全量本体零硬编码（facesnap.go
  faceInventory）——建表落地即面，无「该建未建」残余语义。
- 合法性核=**合法新面**：CEO 裁决 #4058 面清单新面五件终态明载
  「ops_seat_card表=席归一+T-0455部署落地合法新面」；写径=seatcard
  服务 FIN_PG_DSN 同 roster_rw 写册角色（internal/ops/seatcard+
  cmd/seatcard 在册，fin_0121 同族授权）；生产亲证 public 基表在册
  +ops_face_snapshot 批 14/15 在册。
- 升件源链=#4046 面清单新面行→#4047 升件→#4057 draft#32→#4058
  CEO 五件终态裁→#4365 自动起草落件→#4376 已批（机批）→#4426
  派单→#4485 dedup skip→#5075 拉起（漏拉/限流补派续跑）→本票。
- 病灶本体=T-0448 同型族：合法新面**无持久基线位**——面宇宙人工过
  基线后新增才值得看，但合法新增面需登记位续判，册外/表重建/快照
  册重置回摆重复升件炸起草拍（T-0393 链；dep_pair/service 先例
  T-0449/0450/0453/0454/0471/0472/0474/0490/0491 同径）。
- 疑似根域勘误：票面起草猜域「评估×信号链面（fin_opp_eval×
  md_signal——T-0384 族沉默探针域）」系 T-0393 模板猜域——病灶本体
  在 hunger 面清单基线册（facesnap table 类枚举），与 eval/signal
  链零涉；sigdist 面零触不越域衍（同 T-0474/0490 勘误先例口径）。

### 实现摘要（枝 t/0493-face-table-ops-seat-card）

- `facesnap.go` `faceBaselineWhitelist` 登记
  `face:table:ops_seat_card`（注册式单键精确匹配零通配——册内面差集
  照落「面清单新面」行携理由键留账但升件免；表重建/快照册重置回摆
  不重复炸票/起草，T-0393 链断点同 T-0448 族先例）。
- `depblind_test.go` 先红桩
  `TestFaceBaselineTableOpsSeatCardRegistered`（T-0388 修复类先红
  桩律：病灶存在断言修前红+同断言修后绿——签名面必须在册且理由键
  非空，注册式不静默）。
- 探针阈/签名面/类册机制本体零改动（红线③守——只修病灶本体）。

### 测试证据

- 先红桩双点：`TestFaceBaselineTableOpsSeatCardRegistered` 修前
  FAIL「病灶存在：face:table:ops_seat_card 未入人工基线册——回摆
  将重复升件（T-0493）」→入册后同断言 PASS；`TestFaceBaseline*`
  九先红桩（T-0449/0450/0453/0454/0491/0471/0472/0474+本票）同跑
  全绿。
- `go build`/`go vet`/`gofmt` 净；`go test
  ./internal/portfolio/hunger/` 全包绿（DB 桩测谱含
  TestProbeFaceDiffBaselineExempt 豁免对照面——册外面照升件不误伤
  断言在案）。
- check_all 绿戳对口见收口段（rebase 后本盒盖戳——收口终盖恒本盒
  律）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——本票仅新增
  `face:table:ops_seat_card` 一键；生产近 14d「观测盲 face:*」升件
  谱 29 签名逐键对拍：其余 28 键（service×16+dep_pair×7+table×4+
  gate_domain×1）零豁免面照升件等各票域裁决
  （MECH/判例闸/外审/外审席/演习/裁定席/起草拍/跑席/COS上行/
  gate_domain:meta/leg:BTC-USDT-SWAP/leg:BTC-USD-270326/
  table:md_eval_pulse/table:fin_exec_drill/table:fin_guard_suspend/
  table:fin_symbol_dialect/signal:basis_ann 族/配管/--take/业主/
  O-Dev交互席 等在案候派 T-0492~0497 族，本票不越权代判）。
- table 类同型面核查：information_schema 全量 BASE TABLE 枚举——
  `ops_seat_card` 为本票签名唯一登记键；余 table 面（md_eval_pulse
  等）各有己票域（T-0497 等），本票登记不触其升件径。
- 验收锚②「探针同窗不再报同签名病灶」：该面已入快照批 14/15
  （机制自愈——册=簿记基线位）；若表重建/快照册重置致面回摆，
  册内面新面行照落携「基线册=」理由键但升件免——同签名升件/起草
  链断（depblindDedupHours 同窗去重口径不变，探针阈/签名面/类册
  机制本体零改动）。

### 偏离/留案

- 兄弟票同批族在案候派——T-0492（face:gate_domain:meta，同 meta
  域姊妹面）/T-0494（leg:BTC-USDT-SWAP）/T-0495（signal:basis_ann）
  /T-0496（service:MECH）/T-0497（table:md_eval_pulse）各面合法性
  核+入册归各票域，本票不代判（facesnap.go 同图邻接行 merge 冲突
  解=条目并集全保，T-0471/0474/0490 先例在案）。
- CEO #4058 五件终态裁在案：ops_seat_card 表归「席归一+T-0455
  部署落地合法新面」档——本票登记=该裁落册执行，非新判。
- 合法性锚=fin_0145 迁移册本体在案+seatcard 写径在册+CEO #4058
  终态裁+生产基表/快照批双亲证（合法新面非幽灵面非断链面）。
