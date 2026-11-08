# T-0445 探针升件自动起草：观测盲@MUU-USDT

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·机械轨票翻）
- 领取：devin-o-worker-T-0445 @ 2026-11-08
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:MUU-USDT`，草案锚 fin_ticket_draft#4）

## 病灶（升件行证据引）

病灶签名：`观测盲:MUU-USDT`（kind=探针类 `观测盲`，key=病灶键 `MUU-USDT`）

末行升件原文：
> 腿行情覆盖失明 MUU-USDT：live/paper 策略腿所需行情键近 10min md_ticks 零新鲜行——归一链订阅面/镜像谓词/写入闸断点待查（探针只报不修）

证据引（ops_event 行 id）：#3007

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='MUU-USDT' ORDER BY id DESC LIMIT 20
```

## 疑似根域

行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/data/okxws/（activeSet×md_ticks 归一链订阅面——MUU-USDT 腿键覆盖缺）
   internal/portfolio/hunger/（腿覆盖探针域 T-0392）；
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

（施工写：实现摘要 / 测试证据 / 偏离说明）

### 根因（生产取证 2026-11-08）

- 腿键来源实证：SG `fin_strategy` `MUU-USDT-SWAP` state=live
  template_ref=funding-hedge——`legset.FundingHedgeSpotArm` 剥 -SWAP
  尾派生 `MUU-USDT` 入 `ActiveSetUnion`（fin_sub_request 无 MUU 行，
  提案 fin_retired——派生臂唯一源）。
- 幽灵键实证：OKX `/public/instruments` 实扫——`MUU-USDT` 现货对
  **不存在**（SPOT 实挂=`XMUU-USDT` 异基币币对；SWAP=MUU-USDT-SWAP
  实挂）。宇宙全谱对拍：525 个 live -USDT-SWAP 中 **305 个基币现货
  对不存在**（股票/商品/新币 perp 为主）——派生臂原语义假设
  「SWAP 基币=现货基币」系统性不成立。
- 断点链：`UniverseCatalog.refresh` 基币现货派生臂**无条件**入
  `all`（原注记自承「兜底」语义但实现直入）→ `MUU-USDT` 入订阅宇宙
  → 腿覆盖探针 required=腿键∩宇宙 命中 → collector tickGate valid
  放行 → tickers 订阅 OKX error 帧→refuseSub→md_ticks 零行→
  `观测盲:MUU-USDT` 升件。探针判据本身正确——**虚构键本应宇宙外**
  （coverage.go 注记「虚构键宇宙外不求供价不炸」），宇宙含幽灵=上游病灶。

### 实现摘要（枝 t/0445-muu-usdt）

- `internal/data/okxws/universe.go` 单件根修：SWAP 基币现货派生键
  改**暂存**（`derivedSpot` 不直入 `all`）——SPOT 族读成=live 目录
  真值闸：实证在的派生键经 SPOT 族环同键覆盖（不重不漏），族外
  派生键=幽灵滤除；**SPOT 族读败=兜底径全量派生照旧**（族级降级
  语义不动——FamFail 桩原断言照绿）。
- 效应面：collector `valid`/探针 `subUniverse`/提名 catalog 闸同读
  一供给件——幽灵键出宇宙后三面同愈（订阅 error 帧绝、腿键不再
  required、假符提名拒）。legset 派生臂/探针阈/签名面/类册零触
  （红线守——病灶在宇宙面不在腿键面）。

### 测试证据

- 红桩先红（T-0388 律）：`TestUniverseCatalogGhostSpotGate` 新锚
  （MUU-USDT-SWAP+XMUU-USDT 实挂形——修前 `MUU-USDT` 幽灵入集红）
  +`TestUniverseCatalogParse` 断言翻向（BBB-USDT 幽灵滤除）——
  修前双红实证病灶存在，修后全绿。
- 兜底语义守：`TestUniverseCatalogFamFail`（SPOT 读败→AAA-USDT
  派生兜底仍入集）原断言不动照绿。
- `go build ./...`+`go vet`+`go test ./...` 全谱 61 包绿。
- check_all 分段：build/vet/import/gofmt/env/rejects/config-write/
  migrations/bridge-audit/ops-chain 全绿；**tolerate-report+
  seat-writes 两红=master 存量**（主树 /opt/o 同跑同红实证——今日
  COS/决策席提交未同步容错册/越权档在案，与本 diff 无关，呈决策席）。

### 生产谱面零误报审计（范围④）

- SG `ActiveSetUnion` 全式实跑=83 腿键；对拍 OKX live 目录：
  36 现货形键中**仅 MUU-USDT 不在 SPOT live 目录**（唯一幽灵），
  余 35 全实证在册经 SPOT 族环照常入宇宙——零误伤；SWAP/期货/uly
  键全实证不受影响；BTC 期权形键本宇宙外（探针原不要求）照旧。
- 验收②生产锚待验：部署后探针同窗不再报 `观测盲:MUU-USDT`
  （uni TTL 10min 自愈——新实例首拍即真值）。

### 偏离/留案

- `MUU-USDT-SWAP` funding-hedge 策略 live 但无对应现货对可对冲
  ——腿键归一正确性属策略准入域（exec `BuildLegs` meta load 对
  幽灵键 fail-closed 已挡真钱径）；「funding-hedge 准入前核现货对
  存在性」若需立另票，非本票签名面。
- check_all 两存量红件（tolerate-registry 未同步+seat_writes 越权
  档）系今日他席提交遗留，决策席发窗前需先行消化；连带 pre-push
  绿戳闸不可达——本枝 HEAD commit 暂留本地
  （/opt/o 共享 git dir 决策席可直 merge），推送待闸消后补。
