# T-0470 探针升件自动起草：观测盲@MUU-USDT

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·取证=收口并master·机械轨票翻）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:MUU-USDT`，草案锚 fin_ticket_draft#21）

## 病灶（升件行证据引）

病灶签名：`观测盲:MUU-USDT`（kind=探针类 `观测盲`，key=病灶键 `MUU-USDT`）

末行升件原文：
> 腿行情覆盖失明 MUU-USDT：live/paper 策略腿所需行情键近 10min md_ticks 零新鲜行——归一链订阅面/镜像谓词/写入闸断点待查（探针只报不修）

证据引（ops_event 行 id）：#3497

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='MUU-USDT' ORDER BY id DESC LIMIT 20
```

## 疑似根域

行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:MUU-USDT（T-0455 册内候选，DM 裁时可改判）；
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
  1. **定性=同签名重复立单（病灶已愈在案）**：本票签名
     `观测盲:MUU-USDT` 与已完成 T-0445（docs/tasks/done/
     T-0445-muu-usdt.md，草案锚 fin_ticket_draft#4）同签名同病灶
     ——T-0445 已根修宇宙派生键目录真值闸（merge c2328c30，
     2026-11-08 01:29 UTC 在案）。本票证据升件 #3497
     （2026-11-07 22:24 UTC）落在修复在飞窗（merge/部署之前）——
     系同一病灶修复落地前的复发报告非新病灶；draft#21 物料化时
     起草拍未查同签名在飞/已完成票。
  2. **生产取证复核（2026-11-08 ~19:40–20:10 UTC，SG+OCI 双端）**：
     - 部署面：SG `o-core` + OCI 三臂（o-collecto/o-projector/
       o-research @149.118.58.98）全 @`6687044`——`git merge-base`
       实证含 T-0445 merge `c2328c30`；deploy-oci 17:24:19 rc=0、
       镜像 deploy 17:24:36 rc=0 ops_event 在案。
     - 宇宙面：OKX live 目录实读（SG 直拍 instruments SWAP/SPOT/
       FUTURES 三族）复算 `UniverseCatalog.all`=1004 键——
       `MUU-USDT`∉all（幽灵滤除）、`MUU-USDT-SWAP`∈all、
       `XMUU-USDT`∈all；collector 粗扫拍 swap=528/spot=466/
       futures=6 每分钟绿=SPOT 族读成→T-0445 兜底臂（famErr 全量
       放回派生键）不触发。
     - 链路面：o-collecto journal 17:22 重订起**零 error 帧**——
       `MUU-USDT` 若仍在宇宙∩活跃集，tickGate（valid∩isUSDTInst）
       必放行 tickers 订阅→OKX error 帧留痕（handle.go:23 日志面
       实证）；`fin_sub_request`/`fin_venue_untradeable` 无 MUU
       活行，`fin_proposal` MUU 行 fin_retired。
     - 探针面零误报审计（生产谱面对拍，判据同腿覆盖探针本体）：
       SG `ActiveSetUnion`∩宇宙∩`IsUSDTInst`=**required 71 键、
       fresh 71、盲 0**——`MUU-USDT` 不在 required 集（腿键仍由
       `FundingHedgeSpotArm` 派生入 union，但宇宙外被
       `legRequiredKeys` 结构性滤除）→探针对本签名**结构性不再
       报**（非宽限侥幸）。
  3. **代码面改动=零**：病灶本体在 `internal/data/okxws` 宇宙供给
     件（票面钦定域 sigdist+hunger 之外），已由 T-0445 同签名根修
     在案且部署生效；本枝复核码面确认探针判据（coverage.go
     `legRequiredKeys` 宇宙∩IsUSDTInst 闸）与订阅闸（tickGate
     valid∩isUSDTInst）同读一宇宙供给件——幽灵出宇宙后两面同愈，
     无第二处断点。sigdist 面 grep 实证与本签名无涉（basis/
     funding_view/xvenue 蒸馏域）。
- 测试证据：
  - 钉死断言全绿（既有锚，本票病灶形逐字 fixture）：
    `TestUniverseCatalogGhostSpotGate`（okxws——MUU-USDT-SWAP+
    XMUU-USDT 实挂形，修前红/修后绿双点 T-0445 在案）、
    `TestGhostInstFiltered`/`TestRefusedNotResent`（订阅面）、
    `TestUniverseCatalogFamFail`（兜底语义锚）全 PASS；
  - `TestProbeLegCoverage*` 七桩真库全绿（O_TEST_PG_DSN 租约
    5435 独占）——含 `TestProbeLegCoverageFictionalNotRequired`
    （宇宙外腿键不入所需集不判盲=本票判据面同义锚）；
  - `go build`+`go vet` okxws/hunger 两包净；check_all 走
    docs-only 面（变更=票面回填单件），绿戳绑收口 commit。
- 生产谱面零误报审计（范围④）：见上「探针面零误报审计」——required
  71 键全 fresh 零盲；`MUU-USDT` 不在 required=签名病灶结构性绝迹。
  24h 同键去重窗 2026-11-08 22:24 UTC 届满后首个 hungerTick（6h 拍
  ≈23:24 UTC）为最终观察点——彼时若仍报同签名则病灶另有根因回本
  票（当前证据链判不可能：required 集结构性不含该键）。
- 偏离/留案（范围外发现——不自行施工，呈决策席裁）：
  1. **起草器同签名去重缺口**：draft#21 与已完成 T-0445（draft#4）
     同签名 `观测盲:MUU-USDT`——升件 #3497 系修复在飞窗内复发
     报告，起草拍未查「同签名在飞/已完成票」即物料化候选票。同类
     签名「修复 merge→部署」在飞窗内的升件行会持续产重复候选票
     ——建议起草/派单面加同签名未结案查重（另票评估）。
  2. **MUU-USDT-SWAP live 策略腿无价实证**：o-core 日志持续
     「策略 3544 MUU-USDT-SWAP 开闸拒——breakeven_beyond_hold:
     毛收 0.0000」——`legset` 派生现货键 `MUU-USDT` 与 OKX 实挂
     现货对 `XMUU-USDT` 异基币币，幽灵键出宇宙后腿键永无
     tickers 供价。腿键归一正确性（MUU→XMUU 映射）与策略准入核
     现货对存在性属另域（universe-derived-spot-ghost-key 条目
     适用边界明载不在 T-0445 处置面）——呈决策席裁是否立票。
  3. **SPOT 族读败兜底放大面（记档不动件）**：T-0445 钦定「SPOT
     族读败=全量派生照旧」降级语义下，若 SPOT 族持续读败 ~305
     幽灵键回宇宙→观测盲升件批量复发（24h/键去重）。当前生产
     SPOT 读正常（粗扫 spot=466/min 实证）不触发。
