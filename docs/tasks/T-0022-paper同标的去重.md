# T-0022 paper 同标的去重（dev 库实证毛边——同 inst 同向 paper 仓叠加）

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 提出 | `[角色:决策]` 2026-10-23（dev 库实证） |
| 类别 | 缺陷修复（小件） |
| 依赖 | T-0012✅ |
| 领取 | devin-t0002 @ 2026-10-23 |

## 背景

dev 实证：BTC 提案每拍 armed→新 paper 策略行，同 inst 同向已挂 5+ 个 paper（fin_proposal 去重闸只管 proposed 态——armed 后下一拍再放一波）。同一标的的 paper 仓叠加=名义额虚增+结算重复，不是「多实例」是「重复实例」。

## 裁决

**同 venue+inst_id+direction 同时最多一个 active paper 策略行**——新提案评估通过时已有同键 active paper=复用（不新建行），已有行非 active（retired/closed）才许新开。策略行 dedupe 唯一索引（部分索引 where state in active 集）。

## 验收标准

- [ ] 部分唯一索引+插入侧拒绝路径测试锚（同键第二 active 行拒建）
- [ ] 已有 active 行时提案不重复建策略（E2E：连拍两次同 inst proposed→strategy count=1）
- [ ] 机检全绿；CI 绿

## 范围外

- probe/live 态去重（真钱态一 strategy-instance 一仓已由 fin_position 面守）；跨 inst 面
