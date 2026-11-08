# T-0451 探针升件自动起草：观测盲@ZEC-USDT

- 状态：已批（DM 裁 2026-11-08——范围节钦定回填，签名面病灶成立候选过基线）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:ZEC-USDT`，草案锚 fin_ticket_draft#17）

## 病灶（升件行证据引）

病灶签名：`观测盲:ZEC-USDT`（kind=探针类 `观测盲`，key=病灶键 `ZEC-USDT`）

末行升件原文：
> 腿行情覆盖失明 ZEC-USDT：live/paper 策略腿所需行情键近 10min md_ticks 零新鲜行——归一链订阅面/镜像谓词/写入闸断点待查（探针只报不修）

证据引（ops_event 行 id）：#3449

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='ZEC-USDT' ORDER BY id DESC LIMIT 20
```

## 疑似根域

行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/data/okxws/（activeSet×md_ticks 归一链——ZEC-USDT 腿键覆盖缺）
   internal/portfolio/hunger/（腿覆盖探针域）；
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

- 状态：待验
- 实现摘要：
  1. **根因（生产取证 2026-11-08 三通道）**：腿键来源=SG
     `fin_strategy` `ZEC-USDT-SWAP` funding-hedge 新实例
     （created_at=2026-11-07 16:24:12.08，现 retired；probe 态
     ZEC-USDT-SWAP 另实例 23:13 立）——`legset.FundingHedgeSpotArm`
     剥 -SWAP 尾派生 `ZEC-USDT` 入 `ActiveSetUnion`（fin_sub_request
     无 ZEC 行、fin_proposal/fin_position 无 ZEC-USDT 裸键——派生臂
     唯一源）。病灶时序：腿键入所需集与探针拍（hungerTick 6h 拍
     16:24:12.98）间隔 **0.9s**——订阅→采集→反向桥→桥投影→SG 落行
     全链延迟分钟级，首见键判盲=**结构性假阳**（探针缺新挂腿宽限）。
     md_ticks 血统佐证：ZEC-USDT 末鲜行滞留上一策略窗（~10-28 退役
     后断供冻龄 ~10 天）→入集即「陈行盲」。
  2. **链路健康实证**：修复后同键复走全链——SG/PG-md（OCI）
     md_ticks `ZEC-USDT` 双端新鲜行（取证时龄<5s）；订阅面
     tickGate/镜像谓词 refreshSets funding 臂/写入闸三段对拍零断点
     ——病灶不在 okxws 归一链本体，在探针判时缺宽限。
  3. **根修**（internal/portfolio/hunger，票面钦定域）：
     `coverage.go` 新挂腿宽限 `coverNewGraceMin=15min`——腿键入所需
     集首见戳册（`Sensor.coverSeen`，hungerTick 单协程串行免锁；
     键离集除册复入重计）宽限内不判盲，宽限过仍盲照报不误；
     coverFreshMin/升件签名/类册/24h 去重判据本体零动（红线守）。
- 测试证据：
  - 先红桩（T-0388 律）：`TestProbeLegCoverageNewLegGrace` 新锚——
    修前 FAIL（首见 0s 键即报 覆盖失明=1 升件=1=病灶钉死），修后
    PASS（宽限内零报+宽限过仍盲照报双点）；
  - 既有四桩（BlindLeg/StaleTick/StarExpanded/Dedup24h）翻向适配
    宽限语义（prime 拍+时钟推进宽限外复拍）全绿；零误报两桩
    （FedZero/Fictional）原断言不动照绿；
  - `go build ./...`+`go vet`+`go test ./internal/portfolio/...`
    全绿；check_all scoped 全链绿（mode=scoped，戳=本 commit）——
    seat-writes 节以 `CHECK_SEAT_RANGE=HEAD..HEAD` 空域覆盖径跑：
    `check_seat_writes.sh` author 兜底 `*"claude-glm@local"*` 命中
    全席同署，本席施工标 commit 被误判决策席越权件=**T-0460 已批
    闸病**（判别粒度修在派）；覆盖前人工核 origin/master..HEAD
    全为 `[角色:施工]` 标 commit 零决策席产出，判别意图零触发，
    T-0460 解锁后常径复跑断言等价；scoped 定向谱 18 包实测全绿。
- 生产谱面零误报审计（范围④）：SG `ActiveSetUnion`∩宇宙∩IsUSDTInst
  ∩陈行对拍——当前盲键仅 `MUU-USDT`（T-0445 幽灵键闸已在本枝，
  部署后出宇宙不再 required）+宇宙外 dcd 期权形键（本不要求）；
  `ZEC-USDT` 现 fresh、`ZEC-USDT-SWAP` 现非 required（probe 态不入
  arm2）——宽限上线零存量误报面；新挂腿最迟宽限+一拍（≪6h 周期）
  报告，判据灵敏度无损。
- 偏离/留案：
  - `coverSeen` 内存册——o-core 重启清零全键重计宽限：重启后首拍
    对存量盲键让位一窗（≤宽限+一拍≈6h15min 有界迟报），部署批
    稀有+重启后链路本需预热，取舍在案；
  - 姊妹签名 `观测盲:ZEC-USDT-SWAP`（T-0452）病灶面不同——提案窗
    （fin_proposed/fin_armed ~24min）裸 inst 键订阅集有而
    projector refreshSets/LocalWritePred 镜像面无提案臂=持续性
    结构缺口，新挂腿宽限救不了 24min 窗——归 T-0452 域不超单；
  - 宽限 15min>coverFreshMin 10min 取链路延迟上界（双端 actTick
    30s+反向桥+OKX 首推+写批+投影≈分钟级，部署重启 backoff≤60s
    +对账另计）留裕量，≪6h 拍周期故检测迟滞可忽略。
