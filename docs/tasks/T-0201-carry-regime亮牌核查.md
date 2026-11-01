# T-0201 carry regime 亮牌核查——carry_diff 1095bps 真实性对拍

> 状态：审计通过 | 收口commit：69586c237e8988ca85c3383c818c24efe2e51aed
> 领取：devin-施工席（无头单飞）@ 2026-11-01 06:2xZ
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**已批**——carry-regime亮牌核查——同支半产未并，T-0145升派前置证未产
> 域：评估面/venue 对拍（D-0034 行动②）
> 性质：取证+定性，禁改码；**先证后派**——若实证亮牌则 T-0145 升派

## 病灶/疑点（决策席 2026-10-27 实拍）

`carry_diff` 现报 BTC/ETH **1095bps≈10.95%净年化**——
= funding_ann(0.0001×3×365×10000=1095bps) − mt5_swap_ann(0)。
两疑问必须先钉死，否则「亮牌」可能是口径伪影：

1. **MT5 swap=0 真实性**：mt5_swap 行 BTCUSDm/ETHUSDm val=0
   （swap_type=1）——venue 侧该标的 swap_long/swap_short 是否真的
   免息？对拍 MT5 contract spec（symbol properties 面：
   swap_mode/swap_long/swap_short 原值）——零是真零还是读径
   没取到值（val=0 与「字段缺席」必须区分）。
2. **funding 侧符号方向**：OKX funding_rate>0 时多头付空头收——
   carry_diff 表达式（funding−short 端）的符号/方向经济学解释
   复核一遍（1095bps 是「做空 OKX perp 收 funding+做多 MT5 免 swap」
   的净 carry，对否）。

## 施工面

1. MT5 contract spec 实拍：BTCUSDm/ETHUSDm 的 swap_long/swap_short/
   swap_mode 原值（readcred 取凭据后 venue 读面）——若 spec 本就 0
   →免息实证；若非 0→信号计算链取错字段立案。
2. funding 方向核对：xvenue.go carry_diff 表达式 vs 双腿经济学
   （OKX short perp 收 funding vs MT5 long CFD 付 swap——方向、
   年化换算、单位三审）。
3. **持续亮牌判**：carry_diff 行史分布（近 7d val 序列）——
   是稳定 >8% 还是噪声尖峰；T-0145 候场阈「净年化 ≥8% 持续」的
   持续窗拍数实证。

## 产出要求

- 结论行：亮牌真/伪+证据指针
- 若伪：坑位定性（哪个环节出伪——spec 读径/换算/方向）
- 禁动生产语义/代码/配置

## 验收判据

- venue spec 原值实拍入票
- 方向/年化/单位三审结论在案
- carry_diff 7d 分布钉值在案

## 决策席验收（2026-10-27 深夜）——亮牌判伪收口

施工三审+我自拍的 mt5_swap meta 双腿合判：
- spec 面：链报 swap_long=-1786.5/swap_short=0 是 venue 原值级
  （meta 直出 venue 读件）——「免息」只在空腿侧为真。
- **判伪**：正 funding 收割向=short perp+long CFD，成本腿=
  swap_long(-816bps)——carry_diff 用 swap_short(0) 虚高了 816bps。
  真净≈+279bps<8% 阈——**T-0145 候场阈未达，不升派**。
- 口径伪影根治转 T-0203（方向腿自洽化）。销号。

## 二期补拍（管理席 2026-11-01 03:4xZ·T-0203 修复后生产复测）

判伪结论生产面验证成立——伪影已死：
- carry_diff 现值：BTC **+258.5bps** / ETH **−807.5bps**（md_signal
  11-01 02:55 实拍）——伪 1095bps 绝迹，真净低于 8% 候场阈一致。
- meta 腿向已正：`direction=short_perp_long_cfd`——swap_long 成本腿
  入算（T-0203 修复生效实证）。
- 漏斗诚实度改善：近 2d eval=proposed 4/rejected 91——伪 carry
  灌出的虚提案已不产。
- 结论：**亮牌判伪终验通过**，本票收口证据齐——T-0145 不升派
  裁定维持；验收债归 T-0249 在案。

## 结果回填（无头施工·三期复拍 2026-11-01 ~06:2xZ——SG `o-postgres` 实拍）

**结论行：亮牌判伪——三锚复证全数维持。** 伪影根因（成本腿误取
swap_short=0 致虚高 ~830bps）T-0203 已根治；T-0145 候场阈
（净年化 ≥8% 持续）7d 窗内**任何时点均未触及**，不升派裁定成立。

### 锚① venue spec 原值实拍（`md_signal` kind=`mt5_swap`，11-01 02:55Z 帧，meta 直出 venue 读件）

| 标的 | swap_long(点) | swap_short | swap_long_ann_bps | swap_short_ann_bps | mid | ann_ok |
|---|---|---|---|---|---|---|
| BTCUSDm | −1778.7 | 0 | −836.54 | 0 | 77608.16 | true |
| ETHUSDm | −51.6 | 0 | −832.20 | 0 | 2263.17 | true |
| SOLUSDm | 0 | 0 | 0 | 0 | 95.62 | false |

「免息」只在空腿侧为真（swap_short=0）；多腿成本 swap_long≠0——与决策席
10-27 实拍（BTC −1786.5/0）同构（点值随 venue 日更漂移属正常口径）。
支上旧回填「spec 未钉」未决锚至此钉死：**是 venue 证实值，非读径缺席**。

### 锚② 方向/年化/单位三审（`internal/research/sigdist/xvenue.go:87–117`，T-0203 后形态）

- **方向**：funding>0 → `direction=short_perp_long_cfd`——short OKX perp
  收 funding＋long MT5 CFD 付 swap_long，成本腿=swap_long（经济学正解）；
  funding<0 → long_perp_short_cfd 用 swap_short。旧式不分方向恒取
  swap_short=空腿=伪影根因，票面判伪完全成立。
- **年化**：funding_ann=rate×3×365×1e4（8h 期→bps）；swap 由采集器按
  points/合约规格年化（meta `swap_long_ann_bps`）。
- **算式自洽实拍**：BTC 1095.00+(−836.54)=**+258.46bps**=val 实写；
  ETH 24.74+(−832.20)=**−807.45bps**=val 实写（02:55Z 帧逐分位对平）。

### 锚③ 7d 分布钉值——持续亮牌判

`md_signal` 当值重写语义（T-0199）下 carry_diff 仅 2 行（BTC/ETH 各一），
历史自输入侧重建：`md_funding` 10-25~10-30 逐日 funding_ann(bps)
日均[min,max]（各 16 拍，3/日结算频）：

| 日 | BTC | ETH |
|---|---|---|
| 10-25 | 821 [609,1034] | 263 [142,384] |
| 10-26 | 977 [764,1095] | 557 [200,1095] |
| 10-27 | 1095 [1095,1095] | 1095 [1095,1095] |
| 10-28 | 1095 [1095,1095] | 992 [948,1072] |
| 10-29 | 727 [368,1095] | 844 [659,1082] |
| 10-30(至08:00) | 1076 [1058,1095] | 181 [−330,692] |
| 11-01 现值 | 1095（funding_view 06:19Z 实拍，next 16:00） | 24.7 同帧 |

- **旧伪口径下**（funding−swap_short(0)≈funding_ann）：亮牌 ≥800bps
  **持续 4+ 日**（BTC 10-26~30、ETH 10-27~29）——非噪声尖峰而是
  **持续性伪影**（更危险：足以长期武装 T-0145 误升派）。
- **修正净 carry**（funding+swap_long_ann≈−833~−836）：BTC 逐日约
  −109~+259bps、ETH −651~+263bps——**7d 窗内峰值 +263bps≪800bps 阈**，
  任何时点未达；funding 钉帽 1095bps（rate=+0.01%/8h 上限）时
  理论顶亦仅 +259bps。持续窗拍数：mt5_swap 4h 拍面（≤6 拍/日）×
  当值重写——抽样密度本就不足以证「持续」，而真值连单日阈值都未到，
  判不升派无歧义。

### 偏离/伴生说明

- **同支半产未并簿记债**：支 `t/0200-0201-audit`（7874bad8）回填内容
  已被 master 后续段（决策验收/管理二期/本拍）全面覆盖——该支不复并
  （并则回滚新证据），簿记债以本回填清；其「spec 未钉」疑点已由
  锚①销案。
- **伴生疑点（非本票界，供决策席裁）**：`md_funding` 对
  BTC-USDT-SWAP/ETH-USDT-SWAP 结算行止于 10-30 08:00Z（缺 6 个结算拍），
  同期他标的（ETHW/PEOPLE 等）照写——WS funding-rate 帧到件疑似漏
  两旗舰标的；carry_diff 上游 `funding_view`（REST 蒸馏径）活体不受影响
  （06:19Z 帧鲜、next_funding 16:00 当值）。
- **零码/零配置改动**（票面钦定取证+定性）；钱径面零触碰；
  全文证据径=SSH sg → `docker exec o-postgres psql -U o -d o` 实拍，
  码面锚=worktree master 同源行号。

审计席：headless @ 2026-11-01

**审计注记**（收口链 300207bb→6bc4cb00→69586c23 全量 diff 亲核；
三锚经 SSH sg→`o-postgres` 独立实拍复证出账，非引回填）：

- 锚① venue spec 原值：`md_signal` kind=`mt5_swap`（11-01 02:55Z 帧）
  与票面逐位一致——BTCUSDm swap_long=−1778.7点/swap_short=0/
  swap_long_ann=−836.5428/mid=77608.16/ann_ok=true；ETHUSDm
  −51.6/−832.1955；SOLUSDm ann_ok=false。「零=venue 真零非读径缺席」
  实证成立（长侧非零值在案即读径工作之反证）——**过**
- 锚② 方向/年化/单位三审：码面亲核 `internal/research/sigdist/
  xvenue.go:87–117`——fundingAnn=funding×3×365×1e4（l.101）、
  funding>0→`short_perp_long_cfd` 成本腿=swap_long_ann（l.112–118，
  T-0203 自洽形态在码）、carry_diff `ON CONFLICT` 当值重写
  （l.133–136，T-0199 语义→仅 2 行实证）。生产 carry_diff val
  逐分位对平：BTC 258.4572≡1095.0000+(−836.5428)、ETH
  −807.4549≡24.7406+(−832.1955)——**过**
- 锚③ 7d 分布钉值：16 日格中 15 格与 SG 实拍完全一致；**10-25 行
  失真**——票面 BTC 821[609,1034]/ETH 263 vs 实拍
  627.3[239.7,1033.6]/225.5[141.7,384.0]。根因本席定位：回填查询
  用右开边界（`ts > '2026-10-25'` 类）吞掉当日 00:00Z 结算行——
  余两行 {608.5,1033.6}→avg 821.05、{141.7,384.0}→262.85 恰合票面
  数；「各 16 拍」实为窗口总行数（2+3+3+3+3+2=16）非每日拍数——
  **过（注记修正正史）**
- 失真实质性：被吞 00:00Z 行=低 funding 行（BTC 239.7/ETH 150.8
  bps），真值只**加强**「7d 窗任何时点未达 800bps 阈」结论；判伪
  方向、峰值 +263bps 顶、T-0145 不升派裁定全部不受影响——不阻判，
  按 T-0283 注记先例入正史。
- 伴生疑点独立实证：BTC/ETH-USDT-SWAP `md_funding` 行止于 10-30
  08:00Z（缺 10-30 16:00~11-01 08:00 共 6 结算拍），同期 MASK/
  EGLD/HBAR/UMA 等他标的照写至 11-01 08:00Z——疑点为真，供决策席
  立案；funding_view BTC/ETH 帧鲜（06:26Z，nft 16:00Z）与票面一致。
- 禁线复核：收口两拍仅触票面（零码/零配置/零秘钥入仓）；施工面
  swap_mode 未单列在表——语义由「swap_long(点)」点值口径+ann_ok
  覆盖，所疑（真零 vs 读径缺席）已解，不阻判。
