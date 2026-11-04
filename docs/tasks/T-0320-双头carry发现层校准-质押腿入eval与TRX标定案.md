# T-0320 双头 carry 发现层校准——质押腿入 eval 与 TRX 标定案

| 字段 | 值 |
|---|---|
| 状态 | **完成**（合 master·判词在案——[角色:决策] devin 无头拍 @ 2026-11-04） |
| 立项 | 业主直令 @ 2026-11-04（`[角色:讨论]` session 代笔——业主明令「直接立票进待办」；草案→已批仍归决策审批） |
| 领取 | devin-施工 @ 2026-11-04（o-worker t/0320-dual-carry-calib） |
| 优先级 | 高 |
| 涉及面 | `fin_opp_eval`（评估件）、`fin_opp_template`/grammar 枚举、`fin_capability_surface`（质押维度）、scout/carry_diff 数据面、`docs/数据流.md`（链路动则同 commit 更新） |

## 背景

业主自营盘实盘标定案：**TRX 现货质押收息 + CMC 空单收 swap（恒正）=双头 carry**——delta 中性两头吃，为 O 账面最佳单（经济案 canon `cmc-trx-vs-company` 在册）。但该单由业主人肉发现入账，**发现层未生同构提案**——实证三处口径盲：

1. **质押腿不入评分**——对冲模板 eval=「资金费−双腿费」，现货腿当裸持，质押/Earn 收益分量系统性蒸发，双头结构被按单头算分；
2. **venue 覆盖缺口**——CMC 类 venue 未入 capability surface，「TRX 空单 swap 恒正」对发现层隐形（`carry_diff` 无数据可差）；
3. **资产质押维度缺列**——「可质押资产×该资产空单付费 venue」二维积无枚举对象。

## 目标

发现管道自动产出 TRX 类双头 carry 假想——**修口径非建系统**：质押腿入 eval + 能力面加质押维度 + CMC 类 venue 喂数据。业主授权录（同档对话录 11-04）：已录入资金全部可用，目的唯一=盈利——授权为上限非免检，live 开闸仍走既有生命周期。

## 验收标准

- [ ] eval 对冲模板评分含现货腿收益分量（质押/Earn APR）——`fin_opp_eval` 行实证（分量字段非零且入总分）
- [ ] capability surface 增资产质押维度（stakable 旗+率源字段）——表行实证
- [ ] grammar 枚举产出 stake_carry（或对冲+质押变体）假想实例——`fin_opp_hypothesis` 行实证
- [ ] **TRX 标定**：枚举面自动产出 TRX×（现货质押+空单 carry）同构提案进 proposed——「系统能否复生业主已验证的机会」即本票及格线
- [ ] CMC venue 实测：经既有 MT5 桥/demo 凭据采 TRX swap 序列喂 `carry_diff`（凭据缺则如实记阻塞）——能力面/数据行实证

## 范围外

- 新结构不开 live——一律走既有 paper→probe→live 生命周期；
- 非 MT5 系 CFD 私有平台（IG/Plus500 类）集成不做——下期另议；
- 钱径闸/守仓面零动；业主 TRX 自营盘持仓不动（仅作标定参照非操作对象）；
- DCD 族不因此案启封（T-0288 回启三件套独立成立才可）。

## 参考

- 经济案 canon：`econ-cmc-trx-vs-company`（T-0314 裁定生成件）；
- T-0288 DCD 生死定性（类目级判死对照）；T-0313 scout 六 adapter（venue 接入先例）；
- `docs/数据流.md`（能力面/枚举面/桥面坐标）。

## 结果回填

**实现摘要**

1. **质押腿入 eval**：`funding.Evaluate` 增 `Yield` 分量（gross=资金费年化+质押 APR）；
   `spotyield.go` 率源读件——能力面 `product_family='staking'` 注册行（ccy→rate_source）
   闭集派发：`trx_onchain`（fin_trx_vote_apr 投票+fin_trx_txs/fin_trx_account 租金实收
   折算）/`md_earn_rate`；域缺席（42P01）/行缺/超龄（research.stake.stale_hours）=缺席
   如实分量置零不虚构；空 spot 反向腿方向闸置零。
2. **能力面质押维+空单付费维**（md_1086）：`tron/staking-TRX`（stakable+rate_source）
   左元；`cmc/okx/mt5 swap-carry-TRX` 右元——二维积=stakable×swap-carry on ccy join；
   mt5 TRXUSD 实测缺口 enabled=false 关面登记。
3. **grammar `stake_carry` 族**（LegPrim+`spot_long_staked`/`cfd_short`）→candidate
   模板+假想卡；**假想官 `stakeCarryScan` 臂**：二维积枚举→双头净年化→
   卡+评估+提案止 fin_proposed（direction 闭集扩 'stake_carry'，md_1086/fin_0105 双链）。
4. **率源闭集**：cmc_realized（fin_cmc_history 30d 日均实收÷空腿名义×365）/
   mt5_swap/funding_view；阈 research.signal.stake_carry_net_bps=300bps+
   buffer 50bps 热启。

**测试证据**

- `TestSpotLegYield`/`TestScanSpotYieldE2E`：TRX-USDT-SWAP eval 行
  spot_leg_yield_apr=0.0333 非零入 gross 0.1428（0.1095+0.0333）；反向置零。
- `TestStakeCarryTrx`：staking×swap-carry 二维积→`cmc/stake_carry-TRX` 卡
  proposed+eval proposed+fin_proposed 提案链（净≈1283bps）。
- dev 实证（o-postgres@5433 实数据）：`stakeCarryScan` 产出 cmc:TRX
  卡+评估 proposed net=0.1283+fin_proposed 提案；grammar 出 `stake_carry`
  candidate 模板+`stake_carry_template` 假想卡；业主实盘口径 ≈13%/年对拍符。
- `fin_capability_surface` 4 行实证（staking×1+swap-carry×3）。
- go build/vet/gofmt/test(`internal/research/...` 全绿)/check_migrations/
  rejects/import 方向（hypothesis→funding 白名单加边——SpotYield 共用读件
  同 EarnRate 先例）全过。

**偏离说明**

- **CMC/MT5 实测如实记阻塞**：EXR 3001157（既有 MT5 桥凭据）TRXUSD 在目录但
  SwapType=SwapNone+TradeMode_Disabled——该 venue TRX swap=0 且不可交易，
  `carry_diff` 无数据可差（探针实采 2026-11-04，mutating 零触）；缺口
  以 `mt5/swap-carry-TRX` enabled=false 登记入册。TRX 空单付费实际数据源
  =CMC 业主盘实收隔夜结算（~10%/年，fin_cmc_history 实数）。
- dev 库 md_funding 陈 >25h——funding 扫在此库不产出新 eval 行属新鲜度门
  正常行为；行实证以集成测试锚行+机制在案（生产鲜数下自然带分量）。
- fin_trx_*/fin_cmc_* 归 fin 域（SG 侧正本）——研究臂在 OCI 盒 PG-md 读不到
  属域缺席如实（42P01 路径已埋），数据面如需上盒走反向桥扩列另立票。

审批注记（决策交互形态 @2026-11-04）：**已批**。口径修三件锚全可机检（eval分量/能力面维度/枚举实例行实证）；红线零碰（不开live不动钱径）；授权录上限非免检在案；标定判据「系统复生业主已验证机会」=发现层最硬验收式。施工提示词另立。

## 决策席验收注记（无头拍 devin @ 2026-11-04）

裁词=**通过**，合 master@（merge 件见 log）。

- 范围红线过：diff 16 件 ⊆ 涉及面+机械随件（migrate_table 登记+import
  白名单加边 hypothesis→funding——SpotYield 共用读件 EarnRate 先例，
  闸件自文档放行径+回填明注非偷渡）。
- 门禁亲跑全绿（本席 worktree 实拍）：build/vet/gofmt/env-reads/
  rejects/check_migrations 七断言/bridge-audit 四列/import 129/138 边
  全过；go test 触面包 funding/hypothesis/grammar 全 ok；仅 cmd/core
  TestGetPipeline*+pgstore TestMigrateChainSurface 红=T-0321 在案
  master 断言漂移（种子行/桥表归属——本件 diff 零触断言册，先例不罚）。
- 验收锚实迹（非口报）：
  ① `TestSpotLegYield`/`TestScanSpotYieldE2E` O_TEST_PG_DSN 下真跑
    PASS——TRX-USDT-SWAP 评估行 spot_leg_yield_apr=0.0333 非零入 gross
    0.1428，反向分量置零（方向闸实证）；
  ② md_1086 能力面 4 行（staking×1+swap-carry×3）静态在案+dev 实查在册；
  ③ dev 库 fin_opp_hypothesis 双行亲查：stake_carry_template+
    stake_carry-TRX 均 proposed；
  ④ dev 库 fin_proposal 亲查：cmc:TRX direction=stake_carry
    net=0.1283 state=fin_proposed——业主已验证结构系统复生，标定线过；
    `TestStakeCarryTrx` 同构链 PASS（净 1280bps≈业主 ~13% 实盘口径）；
  ⑤ CMC/MT5 实测：EXR 3001157 TRXUSD=SwapNone+TradeMode_Disabled——
    venue 产品缺非凭据缺，mt5/swap-carry-TRX enabled=false 缺口登记=
    「凭据缺则如实记阻塞」同构处置，偏离如实回填可接受。
- fixture 生产真形核过：fin_trx_vote_apr/account/txs、fin_cmc_*
  positions/history 字段列与 fin_0046 schema 逐字对（margin 案教训钉）。
- 红线三件零碰：提案止 fin_proposed（paper→probe→live 生命周期不动）、
  无 venue 注册请求、钱径/守仓面零动、业主 TRX 实盘仅作标定参照。
- 部署后验项（随下次 o-core/research 部署链销，非本件红）：fin_0105/
  md_1086 双链迁移上 SG+OCI 库；stakeCarryScan 生产一拍实产对拍；
  md_funding 鲜数下 funding 评估行带 spot_leg_yield_apr 分量；fin_trx_*/
  fin_cmc_* OCI 侧域缺席（42P01 如实——如需上盒走反向桥扩列另立票）。
