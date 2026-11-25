---
id: sem-gate-note-like-narrow-second-strike
kind: pitfall
date: 2026-11-25
ticket: O-39400
---

# sem_gate note_like 收窄=机制写径静默拒第二处实证

`ops_event_sem_matrix` PK=(event,seat) 一对一行——fin_0237 给
(处置,落地件) 补的放行行 note_like 收窄到「工单#% 销——落成自动销%」。
凡机制写径（探针/脚本）复用同 seat×event 但 note 形不同=BEFORE
INSERT trigger RAISE 拒写。

**O-39400 实证**：desklife 探针销票行（交付实据销/premise-dead）用
seat='落地件' event='处置'——矩阵放行面不覆盖→测试库即炸
`ops_event_sem_gate: 拒写非法`。与 fin_0237 landq_autoclose 同型坑
第二发——「机制销行」语义族共享 seat×event 对但 note 形各异。

**处置**：fin_0254 把该行 note_like 放宽到「工单#% 销——%」——
「销——」前缀本身就是 ops_act_parse 的 close 词位（销[——：:，,+$]），
闸面与 act 语义对齐；落地件其余处置面（挂起条件=/说明类）仍不放行，
收窄性不变。

**判据**：新增机制写径落 ops_event 前，先核 `ops_event_sem_matrix`
该 (seat,event) 行 note_like 是否覆盖新 note 形——矩阵外 event 直通、
矩阵内 event 无命中=拒写。ON CONFLICT(event,seat) DO NOTHING 语义下
第二种 note 形不能靠加行解决，只能放宽 note_like 或换 seat。
