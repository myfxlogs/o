# IBKR carry 族枚举（T-0339 产出③）

核证日 2026-11-04；数值均为公开页实取，估算处注明。
执行闸不因此松一毫米——本件=能力面地图，非提案。

## 族表

| 族 | 形态 | 毛收来源 | 保证金/占用 | 成本项（实测数） | 门槛 | 出处 |
|---|------|---------|------------|-----------------|------|------|
| **CME 加密期货基差** | 空 CME BTC/ETH 期货 vs 多现货（或 ETF/永续对冲腿）——机构级 cash-and-carry | 期货升水收敛（历史中枢随 regime 波动，CME 机构盘主导） | **MBT 保证金 ≈ $1,435/$6,334 名义 ≈ 22.6%**（CME 官网 est）；全档 BTC（5BTC）保证金 >$100k | 佣金 MBT $0.85/张/边 + CME exchange recovery $1.15/张 ⇒ **≈$2.00/边/张 ≈ 3.2bps**；全档 BTC 佣金 $5 + recovery $8.25 ⇒ ≈0.3bps（量级差 10×）；滚动换月成本另计 | Futures-US 权限+CME L1 $1.55/月；现货腿需在册 venue（对冲跨所=执行面新族形态） | cmegroup.com MBT 页（notional/margin est）；IBKR commissions-futures 页；IBKR CME fee recovery 页 |
| **闲置现金利息（法币 Earn 对标）** | 证券段 settled cash 躺平生息——**基准族**：任何 IBKR carry 的机会成本锚 | IBKR 按 BM（benchmark）-0.5% 付息 | 零占用（现金本身） | **USD：首 $10k=0%，>$10k 按 BM−0.5%，当前 3.38%**（NAV≥$100k 满档；NAV<$100k 按比例递减）；IBKR Lite 再 −1%；commodities 段现金**不付息** | 无（自动生效）；满档 NAV≥$100k | interactivebrokers.com/en/accounts/fees/pricing-interest-rates.php |
| **FX carry（现金汇差）** | 持高息币现金/空低息币（IDEALPRO spot 或 CME FX 期货） | 两币 BM 利差 | spot FX margin：主要币对 **≈3.33%（30:1）**零售上限档（管辖区上限取 max(监管,IBKR)）；期货走 CME 保证金 | FX 点差（IDEALPRO 主流对 fraction of pip 档）+ 换汇佣金极小；负腿按 BM+spread 计息 | Futures/FX 权限；美国户 cash forex 需 ECP 门槛（$10M 资产/hedging $5M）——**业主辖区决定 spot 可及性，期货径无此限** | IBKR margin-forex 页；commissions-forex 页 |
| **box/期权融资（可选）** | SPX/XSP 指数期权 box spread 锁定隐含利率借贷 | 借出/借入利率≈隐含 SOFR 段——相对 margin loan（USD BM+1.5%≈5.38%）常便宜 | 欧式指数期权 box=净债组合，占用极低 | 期权佣金+行权/指派费；腿多执行复杂度高 | Options 权限；属进阶面，本轮只记存在性 | IBKR commissions-options 页；margin-rates 页（对照锚） |
| **证券借出 SYEP（可选）** | Stock Yield Enhancement Program——全付股票出借分润 | 借券费 **50% 分润**（IBKR 收 50%） | 持仓本身 | 收入随借券需求浮（hard-to-borrow 才有显著率） | 开通 SYEP（Settings 内一键）；非美户可及性按实体 | IBKR Stock Yield Enhancement 页 |

## 对 O 评价面的映射

- **可算口径**：CME 基差族可直套现有 `futures basis` 文法
  （短腿=CME 期货，长腿=现货/ETF——但长腿 venue 不在册=跨所腿新形态，
  落地另立族提案票）。
- **基准锚**：闲置现金 3.38%（USD、NAV≥$100k、>$10k 段）= IBKR 侧一切
  carry 的 floor，与 OKX Earn 停用档同位（对照锚已入候选卡 evidence）。
- **数据面缺口**：IBKR 无公开免鉴权行情端点——scout 公开面度量臂
  采不到 IBKR（候选卡 metrics 注定空，须凭据落地后走 credentialed
  adapter，另票）。
