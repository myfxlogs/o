# T-0069 armed 提案淤积退役闸

> 状态：完成（判过并码在案，11-01清账翻面）

审计席：headless @ 2026-10-31

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

## 判决注记（审计席 headless @ 2026-10-31——收口 commit d9922299 全量 diff 逐锚对账）

| 验收锚 | 对账 |
|---|---|
| armed 被吸收/取代后落 fin_retired（或周期清理阀） | PASS——`retireStaleArmed`（run.go）挂 `admitProposals` 尾调、每拍必跑（提案集空也执行），scan 由 ev_eval/ev_strategy 事件驱动（无新增 ticker，D-0008 合规）——清理阀形即票面许型 |
| 同键只留最新一行 armed 活跃 | PASS——`UPDATE ... WHERE state='fin_armed' AND id NOT IN (SELECT max(id) ... GROUP BY inst_id,direction)`；键域 (inst_id,direction) 与去重闸/grammar/review 各处同口径；direction 列 NOT NULL 无空组歧义；单语句原子无 TOCTOU |
| 不动 paper/probe/live 晋升语义 | PASS——diff 仅新增 retireStaleArmed+尾调，admit/promote/openLive 与 fin_strategy_transition 写径零触；只动 fin_proposal.state armed→retired |
| armed 行数收敛常数级 | PASS——每拍后各键 ≤1 armed（≈活跃键数），存量 1580+ 首拍批量收敛；TestPaperDedupe 断言改 2 armed→1 armed+1 retired=防复回归锚在票 |

**产物三证**：宣称件 `retireStaleArmed` 落名实查在册（master run.go:229，现已被 T-0136/T-0161/T-0258 扩挂 tag2 TTL+tag3 策略面——原闸仍在）；路径实跑=admitProposals→retireStaleArmed 接线在案（行 189）；文件 run.go/machine_test.go 实存。

**偏离记**：①retire 行无 retire_reason/retired_at 溯源列——该二列系 T-0161（fin_0065）后增，本 commit 时点不存在，非锚缺；②该 merge SHA 无服务端 CI run 记录（10-27 绿戳律前置窗），测试绿证以票面声明+断言件在册为凭——本席未重跑 lifecycle 套件（并发全谱互踩共享测试库禁令）；③退役败=fail-closed 整拍停（error 上传），淤积治先于新开户放行——倾向保守可收。
