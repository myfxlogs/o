# T-0471 探针升件自动起草：观测盲@face:service:CEO

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·取证=收口并master·机械轨票翻）
- 领取：devin-o-worker-T-0471 @ 2026-11-08
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:service:CEO`，草案锚 fin_ticket_draft#22）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:service:CEO`（kind=探针类 `观测盲`，key=病灶键 `face:service:CEO`）

末行升件原文：
> 面清单新面 face:service:CEO：机器可列面 service 类新增 CEO——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#3925

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:service:CEO' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:face:service:CEO（T-0455 册内候选，DM 裁时可改判）；
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

### 根因（生产取证 2026-11-08）

- CEO=管理席无头班次席——T-0456 业主令「CEO走无头——交互态与COS
  重复」建席（O组织席谱对齐 CEO/COS/CTO/施工 语系归一）+T-0465
  北极星账钦定面；合法席位非幽灵。
- ops_event 写径双件实证：①`scripts/meta/ceo_tick.sh` 班次壳
  `export O_SEAT=CEO`→report.sh 并轨（管理令/处置/移交/已批等
  自报事件）；②`scripts/meta/ceo_northstar.sh` 直插
  `seat='CEO'`（北极星账快照行+背离议题行，24h 同指标去重）。
- 生产账：seat='CEO' ops_event 首行 2026-11-08 00:25:17Z，至
  12:23Z 共 169 行（北极星账/管理令/处置/移交/已批五类事件）；
  升件源链=#3924 面清单新面行→#3925 升件→draft#22→#4355 起草→
  #4368 已批（T-0455/0456 批权归径机批）→本票。
- 新面成因链：service 面=ops_event 近 7d `DISTINCT seat` 活跃
  投影（facesnap.go 设计口径「写者席即服务面投影」）——CEO 席
  2026-11-07 建席首班 11-08 00:25Z 起写，入集拍=批 11（b11/b12
  快照均在册自愈在案）→上拍批 10 册外→「面清单新面」行+观测盲
  升件 #3925。
- 病灶本体=T-0448 同型：合法间歇写席**无持久基线位**——CEO 班次
  驱动写（kick+30min 兜底拍，队空=心跳即退零 ops_event 写），
  静默 >7d 出集→再活跃=再现新面→再升件再起草无限循环（「该建
  未建」残余差集的册外盲区——面宇宙人工过基线后新增才值得看，
  间歇面需要登记位续判）。

### 实现摘要（枝 t/0471-face-service-ceo）

- `facesnap.go` `faceBaselineWhitelist` 登记
  `face:service:CEO`（注册式单键精确匹配零通配——册内面差集照落
  「面清单新面」行携理由键留账但升件免；窗摆回不重复炸票/起草，
  T-0393 链断点同 T-0448/0450/0453/0454 先例）。
- `depblind_test.go` 先红桩
  `TestFaceBaselineServiceCEORegistered`（T-0388 修复类先红桩律：
  病灶存在断言修前红+同断言修后绿——签名面必须在册且理由键非空，
  注册式不静默）。
- sigdist 面零触——疑似根域（评估×信号链面）系起草拍自动猜域，
  病灶本体在 hunger 面清单基线册（票面面内 depblind*/facesnap*
  同族径），不越域衍。

### 测试证据

- 先红桩双点：`TestFaceBaselineServiceCEORegistered` 修前红
  「病灶存在：face:service:CEO 未入人工基线册——窗摆回将重复
  升件（T-0471）」→入册后同断言绿；`TestFaceBaseline*` 四先红桩
  （T-0450/0453/0454+本票）同跑全绿。
- `gofmt`/`go build`/`go vet` 净；`go test
  ./internal/portfolio/hunger/` 绿（DB 桩测谱含
  TestProbeFaceDiffBaselineExempt 豁免对照面全过——check_all
  租约 O_TEST_PG_DSN 口径同跑）。
- check_all 分段全绿（mode=scoped）：build/vet/import/gofmt/
  env-reads/rejects/config-write-domain/migrations/bridge-audit/
  tolerate-report/ops-chain/seat-writes + scoped 定向 go test
  谱（hunger+反向闭包+恒测册 20 包）全 ok——绿戳落本枝 HEAD
  commit（check-green-<SHA> 对口）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——生产近 24h「面清单新面」20 行键谱
  （dep_pair leg/config/signal×5+service×4+table×1+gate_domain×1
  等）与册零交集，册外面照升件零误伤；service 册外新面
  （判例闸/起草拍/席O-COS/外审席/COS上行 等）各有己票域
  （T-0472/0473/0474 在案）不越权代判。
- CEO 已入快照批 11/12（机制自愈在案）——册=窗摆期保险丝非现报
  抑制；面清单新面行照落留账（每 >7d 静默窗摆至多一行，drafter
  只吃升件不误起草）。
- 验收锚②「探针同窗不再报同签名病灶」：修复后若 CEO 静默出集
  摆回，册内面新面行照落携「基线册=」理由键但升件免——同签名
  升件/起草链断（depblindDedupHours 同窗去重口径不变，探针阈/
  签名面/类册机制本体零改动）。

### 偏离/留案

- 兄弟票 T-0472（face:service:COS上行）/T-0473（外审席）/
  T-0474（席O-COS）同签名族在案候派——各面合法性核+入册归各票
  域，本票不代判（facesnap.go 同图邻接行 merge 冲突解=条目并集
  全保，T-0447/T-0453/T-0454 合入先例在案）。
- CEO 席非 fin_employee_registry 在册员工（该册=员工域非席位
  域）——合法性锚=T-0456 业主令建席+ceo_tick/ceo_northstar
  双写径+ops_event 169 行/日生产实证，理由键已载。
