---
schema_version: 2
id: seat-identity-selfdeclared
kind: invariant
risk: medium
status: verified
keywords: [席位, sem_gate, 复审过, 分席独立, 自报身份, append-only, 触发器, 约定执行]
scope: [ops_event, ops_event_sem_matrix, land_queue review_ok, 会话身份]
summary: 分席独立复审=约定执行非机械执行——seat 是无身份绑定的自报字段,sem_gate 触发器只验「座×事件×note形合法」不验写者是谁;复审过走记事类无闸;真正防御=追加账+哈希链可溯+诚实申报惯例。防过失不防蓄意,符威胁模型(会话皆业主授权件)。
first_seen: 2026-11-21
last_verified: 2026-11-21
skills: [事实核验]
evidence_refs:
  - type: ledger
    ref: 处置/复审代录实证 2026-11-21(外席以处置席补审计支候审行+以复审席写复审过,闸全过——亲手演示约定执行)
---

# seat-identity-selfdeclared —— 席位身份是自报不是认证

## 实证三层(ops_event_sem_gate_trg 函数体亲读)

闸体判定逻辑:
```
IF 矩阵有 NEW.event 的行 THEN
    IF NOT EXISTS(同event ∧ seat=NEW.seat ∧ note 配 note_like) THEN
        RAISE EXCEPTION
```

逐层成色:

- **相位闸=机械执行**:非法 (seat,event) 对在库层 RAISE EXCEPTION
  (外席写处置位被拒实证)。但覆盖是**选择性的**——只闸
  呈批/唤起/处置/已批/拉起/条件请求/派单/等锁/裁决/起草/领单/驳回
  12 类相位事件;`记事/备忘/工单/升件/量表/落地/会话收口/恢复`
  **全矩阵缺席=全类无闸**。
- **席位身份=约定执行**:NEW.seat 是写者自填文本,闸验证「该座能否
  写」不验证「你是不是该座」。连机器座亦可自报——`seat='唤知机'
  event='唤起'` 该对合法照样过闸。
- **分席独立复审=约定执行**:复审过行是 `记事` 事件(无闸类)——
  land_queue `review_ok` 的 `seat<>aseat` 是自报文本对比;单会话写
  `候审回执`(处置类闸放处置/复审/施工/研究/自我五座任选一)
  +`复审过`(记事无闸)两行即物理绕闸。

## 真实防御链(不是闸是痕)

ops_event 追加制+prev_hash 链(`ops_event_chain_trg`,篡改可证
——opschain_test 专验追改必被揪)+诚实申报惯例(外席代录会自写
「外席/代录」字样)+`唤起`行由真实调度件写=相对硬的身份基片
(理论仍可自报伪造)。

## 裁决定性

防**过失**不防**蓄意**——写错事件类型被拦,蓄意自报双席可绕。
符系统威胁模型:会话全业主授权件,防的是手滑不是内鬼;审计账
+哈希链保证「骗了也永远看得见」。

升级径(若业主要真机械):复审行 `unit=` 对拍 `唤起`行——但 unit
亦自报,真机械须分席 DB 凭据(角色级 GRANT),架构大改不值;现
制「可追溯非可预防」在授权会话域内即合理终态。

## 可防一问

凡以 seat 字段做闸语义的件(review_ok/claim 窗/工单相位),须认清
「seat 合法」≠「写者是该座」——把席位闸当机械防线会算错真防御
厚度;真防御永远是 append-only+哈希链+公开账面。

## 排查口诀

「某闸独立性」先查两物:该 event 在 ops_event_sem_matrix 有无行
(无=无闸类,任意座可写);seat 判定用 NEW.seat(自填)还是有身份
绑定(凭据/单元账对拍)——两问定机械成色。
