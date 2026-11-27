---
schema_version: 3
id: dir-mom-short-flap-regime
kind: case
risk: low
status: observed
keywords: [dir_perp_mom, short flap, 同签压制率, 锯齿 regime, 死签 churn]
ea_link: EA-06
summary: 锯齿反抽市 24h 内 11 币 30 次同签 dir_perp_mom/short_perp 提案被死签压制——提案面 flap 是 regime 判据的独立佐证，同签压制率可作 eval 先验但今日无消费件。
first_seen: 2026-11-27
last_verified: 2026-11-27
evidence_refs:
  - {type: table_row, id: "ops_event#42861"}
  - {type: code, id: "internal/portfolio/lifecycle/lessongate.go deadSigSuppress"}
---

# dir_perp_mom short 同签 flap regime（2026-11-27 · 哨#42861 取证）

**情境**：retire-surge 哨报 24h 退役 32 件。死因聚合：dir_signal_flip×30 /
dir_stop_loss×11 / paper_ttl×6——全带死签证据、零 armed 误退
（#13081 口径内，哨销）。
**实数**：30 次压制集中在 11 个币（AAVE/ARB/BIO/KAITO/LPT/NMR/RAY/ZIL
等 ×3-4 轮）——同 (tmpl,inst,dir) 死签在案，eval 臂每拍仍 re-propose
同一批 short，lessongate 每拍压回。提案面 flap 而非策略面反复死。
**根因**：lessongate 消费在 mint 点（下游拦截）——上游 eval 不读死签面，
同签压制率这个元信号今日只有 v_lesson_family_stats 人读面，
无自动消费件。锯齿反抽 regime 下 short 信号反复死=季节判据本身，
与 eval 侧 z_weak 拒判同根独立佐证。
**判据挂接**：EA-06（持续性窗/统计面非瞬时值——churn 率=窗统计量非
单点信号）。
**回改点**：semantic:dirmom-eval-sigchurn-prior（候选——dirmom/eval
读同签压制率先验：同签名近窗压制≥N 的 inst 信号降权，比提案压制
更上游；立票前不急——下游拦截零漏钱，只把「扇过」记性前移）。
**证据**：ops_event#42861 哨行+24h 教训消费行死因聚合（外席实查）；
lessongate.go deadSigSuppress 源码面。
**处置**：本案=知识落账件——拦截链有效不翻案；语义回改点留裁定席
评审，立不立票归判。
