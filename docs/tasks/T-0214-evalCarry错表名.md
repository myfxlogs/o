# T-0214 evalCarry 查询不存在的 md_savings_rate——carry 族借贷/Earn 永判 unevaluable

> 状态：待施工 | 优先级：P1
> 立项：决策席 @ 2026-10-28

## 病灶

`internal/research/grammar/run.go` `evalCarry` 查询
`md_savings_rate`——全仓 migrations 无此表（真表=`md_earn_rate`，
md_1016/md_1049 定义）。表缺→err→恒落 unevaluable/savings_rate_missing。

第二层：即便改对表名，盒 PG-md `md_earn_rate` 0 行——earn 采样
worker 跑在 SG 写 SG 侧，OCI 盒无该数据（SG 侧 md_earn_rate 有行、
健康板 earn 现率采样=sg 在册）。改表名后仍 savings_rate_missing
（0 行分支）直到数据到盒。

## 钦定修法

1. 表名改 `md_earn_rate`（顺手核列名对上 md_1049 定义）。
2. 数据径二选一（施工调研后决）：
   - 甲·earn 采样臂迁/复制到盒跑（savings-rate 公开读面在能力账
     在册，盒上凭证径已通）；
   - 乙·SG md_earn_rate 反向桥入盒 PG-md（复用 T-0172 反向桥机制
     ——注意 md_* 表域反向桥首例，先查桥表白名单允不允许 md 域）。

## 验收锚

- 桩测：md_earn_rate 有行→evalCarry 不再落 savings_rate_missing。
- 生产：lending_spread/earn_term_spread eval 出现 rejected/proposed
  判词（非 unevaluable）。
