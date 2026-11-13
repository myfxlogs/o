---
schema_version: 2
id: ticket-close-finality-predicate
kind: pitfall
risk: high
status: verified
keywords: [销票, 终局判据, 接续, 挂起条件, 谓词漂移, codework_stall, tickEnded, runbook, audit_close_ticket_pred]
scope: [cmd/core/supply_esc.go, cmd/core/feed_esc.go, scripts/cognition/runbook.md, scripts/audit/asserts/ticket-close-predicate.sh]
summary: 销票判据写成「任意处置引证存在」=接续/挂起行也当销票,票被误销至死区(#11481活例:真缺口仍在而 stall/OPEN 双面失踪)。终局销=引证∧非挂起条件=∧非「工单#id ?接续」锚定行;挂票限内暂抑、过期回哨。同一语义三处独立实现必漂——修复面=四处谓词同式归一。
first_seen: 2026-11-13
last_verified: 2026-11-13
skills: []
---

# 销票终局性谓词——「引证存在」≠「已销」（审计#12643）

**坑**：`codeworkStallEscCheck`(supply_esc)+runbook OPEN 查询的销票
判据=「存在任意引工单#id的处置行」→`工单#11481 接续…`进度行
与`挂起条件=`挂票行都判成销票→真未决票（日K数据面缺口）从
stall 哨与 OPEN 队双面消失=死区。

**终局语义**（销案三态的机械判据）：

- `挂起条件=` 引证行：**非终局**——期限内暂抑（票开着但缓办），
  票期限一过回 OPEN/stall 面复验（与 `feed_esc.tickEnded` 同语义）。
- `工单#id ?接续` 引证行（**ref 紧邻锚定**）：**非终局**——进度/
  接续报告非销票。
- 其余 `工单#id` 处置引证=终局销。

**锚定纪律**：剔接续**必须**锚到票引证（`note ~ ('工单#'||id||' ?接续')`），
裸 `NOT LIKE '%接续%'` 会误剔真销行——#11287 实案：销案行附言
含「#11331 接续件」，全词剔即把真销票判活。

**漂移根因**：同一「销票」语义在 supply_esc/feed_esc/runbook
三处各自手写 SQL——feed_esc 先长出挂起语义而另两处漏剔。
修法=四处谓词同式归一（c65519d3e→merge ae9407592→
o-core:ae94075923fe），回归锚=`asserts/ticket-close-predicate.sh`。
新谓词面再改=三处同改，漏一即再病。
