# T-0069 armed 提案淤积退役闸

## 背景
fin_proposal state='fin_armed' 已堆 1580+ 行且每评估周期 +2——armed 提案被同键策略吸收后既不退役也不清理，表无限胀（pg-capacity 域病灶）。

## 要求
- armed 提案被吸收/被新提案取代后落 fin_retired（或周期清理阀）
- 同键只留最新一行 armed 活跃，旧的退役
- 不动 paper/probe/live 晋升语义，纯提案表卫生
- 验收：armed 行数收敛为常数级（≈活跃键数），不再随周期线性增长
