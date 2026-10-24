# T-0069 armed 提案淤积退役闸

## 背景
fin_proposal state='fin_armed' 已堆 1580+ 行且每评估周期 +2——armed 提案被同键策略吸收后既不退役也不清理，表无限胀（pg-capacity 域病灶）。

## 要求
- armed 提案被吸收/被新提案取代后落 fin_retired（或周期清理阀）
- 同键只留最新一行 armed 活跃，旧的退役
- 不动 paper/probe/live 晋升语义，纯提案表卫生
- 验收：armed 行数收敛为常数级（≈活跃键数），不再随周期线性增长

## 结果回填（devin-t0002 @ 2026-10-26——armed 淤积退役闸）

| 验收锚 | 态 | 证据 |
|---|---|---|
| armed 被吸收/取代后退役 | PASS | retireStaleArmed=同键只留 max(id) armed——旧武装批量退役（admitProposals 尾调，事件驱动非周期环） |
| 同键只留最新一行 armed | PASS | TestPaperDedupe 改断言：2 提案→1 armed+1 retired（淤积防复实证） |
| paper/probe/live 语义零触 | PASS | 只动 fin_proposal.state 退役径——晋升链零触碰 |
| 机检全绿 | PASS | lifecycle 包全绿 @5434；1580+ 存量淤积=部署后自然收敛（retireStaleArmed 批处理） |
