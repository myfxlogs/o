# T-0320 双头 carry 发现层校准——质押腿入 eval 与 TRX 标定案

| 字段 | 值 |
|---|---|
| 状态 | 草案 |
| 立项 | 业主直令 @ 2026-11-04（`[角色:讨论]` session 代笔——业主明令「直接立票进待办」；草案→已批仍归决策审批） |
| 领取 | |
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

（施工写：实现摘要 / 测试证据 / 偏离说明）
