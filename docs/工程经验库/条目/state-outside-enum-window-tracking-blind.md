---
id: state-outside-enum-window-tracking-blind
title: 状态机终态漏出枚举窗——息腿/健康闸/重试三隐身链同根
kind: pit
date: 2026-11-24
source: 工单O-41350（O-40707 post-merge 三维审遗留，margin.exec_enabled 武装前置）
---

# 终态漏出枚举窗=追踪面三隐身链（margin-short 实证）

**坑形**：fin_margin_loan 状态机含 `flatten_failed` 终态（平仓序败
留账负债仍在息），但 `marginOpenLoans` 枚举窗只圈 `('open','closing')`
——该态负债同时脱离三面：①息腿 accrual 停记（负债仍在 venue 计息，
账面 accrued 冻结低估）；②健康闸 loanUSD 求和漏项（marginRatio 分子
偏小→kill 档可能永不触发=裸奔放行）；③无任何重试径（flatten_failed
写下即沉默——唯一索引也不含该态=同币可双借叠加负债）。

**判别**：凡「枚举窗 IN(...) 白名单」与「状态机 CHECK 词表」并存的
台账面，新增状态时必须同改三处：枚举 SQL、部分唯一索引 WHERE、
消费面重试/升级路径。漏一处=该态变成账外幽灵。

**修法**（O-41350 落地形）：枚举窗扩三态全收；唯一索引 DROP+CREATE
同步；flatten_failed 走防抖重试闸（连击计数+间隔+封顶升件
esc_class=margin_flatten_fail 一次性）——重试序幂等化
（ResolveOrderDetail 同 clOrdID 前序态先定：filled=直进 repay
不双买、live/partial=在途不动、absent=发新单、读败=还原前态
不动不猜）。

**推广**：状态机「失败留账」态设计时三问——谁还在枚举它？谁还在
给它计息？谁负责把它救回来或喊人？三问都无答=该态是事故放大器。
