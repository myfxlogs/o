---
id: md-ticks-colname-px-vs-last-union-kills-gate
title: md_ticks 价列名是 last 不是 px——UNION 两级读面写错列名=闸恒拒死锁
kind: pit
date: 2026-11-26
source: 工单O-40707 margin-short 接线（borrowquote.spotPx 潜伏 bug 实证修）
---

# 两级价读面列名不对称（O-39460→O-40707 病灶链）

**坑形**：`md_ticks` 价列=`last`（numeric），`md_mark_index` 价列=`px`。
UNION 两级读面若两臂都写 `px`——md_ticks 臂结构性报错（column does
not exist），整查询恒 err。39460 `borrowquote.spotPx` 即此形：eval 侧
额度判据 `loanUSD=maxLoan*px` 里 px 恒 0 → `borrow_quota_insufficient`
恒拒——**证据面修好后闸仍全拒，病灶从「无证据」换成「证据读不出」**。

**判别**：同型读面已立正本——`internal/ledger/diff_px.go` 两级读法
（md_tick_scan.last→md_ticks.last∪md_mark_index.px+10min 鲜度窗）。
新价读面先抄 diff_px 列名，不要凭「两表都有 px」直觉写 UNION。

**修法**：UNION 两臂各写真列名（`SELECT last v,... UNION ALL SELECT
px,...`）+ 鲜度窗 `ts>now()-interval '10 minutes'`；读败/无行=0 由
调用面按不可证 fail-closed。O-40707 `cmd/core/margin.go
marginSpotPx` 按此落地，borrowquote 同步修（判据语义零改——只修
列名让判据能真读）。

**推广**：time-series 双表 UNION 读面——列名先 `\d 表` 核，不同源
表（ticks=last / mark_index=px / tick_scan=last）同域不同名是常态。
「闸恒拒」排查序：先证据行在不在，再证读面查询真不真能跑通——
恒拒+恒 0 读数=读面病灶非判据病灶。
