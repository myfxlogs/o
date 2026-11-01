# T-0292 dispatcher 派前票态否决闸 + meta_task 词表部署差补 + 陈行批转

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-11-01（门2-DM directive obl372 落箱即办——MANAGE 急诊件） |
| 领取 | （session 标识 @ 时间；空=未领） |
| 优先级 | **急诊 P1**——作废/完成票继续照派，每票空转 ~9–19 万 token（今已实证 6+ 次） |
| 涉及面 | `/opt/meta/mbus-dispatch.sh`（`task_scan_dispatch_one`）、`/opt/meta/sql/meta_task.sql`（部署重放径）、SG `o-postgres` 生产库 `meta.task_phase_words`/`task_phase_edges`（DDL 幂等重放）、`meta.meta_task` 陈行批转 |

## 背景

obl372（管理席 14:21 急诊件）：14:16:55 dispatcher 重启后 boot 扫 `phase='已派'` 25 行**逐个 spawn**，其中票面已终态的票照派空转——T-0215（票面完成）in=131865 tok rc=0；T-0279（票面作废）自检 stale 让位退场 in=185924 tok；T-0280/0284/0285/DBP3c 同式自检退出。根因：`task_scan_dispatch_one` 只认 `meta_task` 行相位，spawn 前零票面否决；票面 `状态` 字段降为「设计文档载体」后无任何消费面回读终态。

**次生病灶（本席清算时实证）**：SG 生产 `meta.task_phase_words` 仅 13 词，缺 `作废/挂起/候派`——DBP3b 词表归一 DDL（`sql/meta_task.sql` 行 32-34 + 边集 59-66）从未部署上生产。后果：① 钦定停泊/裁死径在生产=空话，`task_transition` 打到缺词全部落 `异常词`；② 本席手工清算 T-0215 等行时被迫经 `异常词→完成` 绕行，五张作废票+孤儿行+候派票至今只能泊 `异常词` 候裁。

**已处置（本席 obl372 即办，task_transition 唯一写径）**：
- `异常词→完成`：T-0215、T-0255（票面完成）、DBP3b-消费换血（票面审计通过 R2=事已毕）。
- 留 `异常词` 候裁待批转：T-0279、T-0280、T-0284、T-0285（票面作废并入 ADM）、DBP3b（孤儿行无票面映射）、DBP3c-前端义务面（票面候派=ADM 重设计停泊）。
- 在飞误派会话不自残：T-0279/0280/0284 已自检退场 rc=0，余者自检径同式——**禁杀进程**，自检干净。

## 目标

`task_scan_dispatch_one` spawn 前读票面 `状态`：终态票→行落终态不派；泊态/越位票→行落挂起不派；可派态/票面不可析→原径不阻断。生产词表补至钦定 16 词+边集全图，六陈行批转到位。

## 验收标准

- [ ] **闸位**：`task_scan_dispatch_one` 内，`task_ticket_file` 定位成功之后、`in_flight`/`claim_spawn` 之前插票态否决（veto 先于 claim——陈锚不白占）。
- [ ] **判词集**（`status_field` 读票面 → `enum_base_of` 取基准词，ENUMFIX 单源不另硬编码词形）：
  - 基准词 ∈ {完成, 作废} → `task_claim 已派→作废`（note 记票面词「票面终态 X——派后清算」）+ `log` + return，**零 spawn**；
  - 基准词 ∈ {挂起, 候派, 收口待审, 审计中, 审计通过, 驳回, 待验, 通过, 施工中, 草案, 返修}（票生命周期越位/停泊——行票分歧）→ `task_claim 已派→挂起`（note 记票面词+「候裁」）+ return，零 spawn；
  - 基准词 ∈ {已批, 待派, 待施工, 已派} 或**空值/解析败/词表外** → 原径照派（行是权威，票面仅否决权；词表外值已有 `enum_warn_once` 钉眼告警面，不重复）；
  - `task_claim` 零行（竞败/他径已转）→ 同式 return 不 spawn + log。
- [ ] **生产词表补部**：`sql/meta_task.sql` 经票头钦定径（`ssh sg "docker exec -i o-postgres psql -U o -d o -v ON_ERROR_STOP=1" < sql/meta_task.sql`）幂等重放；执行前先 `SELECT` 存证基线。验收锚：生产 `task_phase_words` = 16 词含 作废/挂起/候派（is_terminal：作废=t、挂起=f、候派=f），`task_phase_edges` 含 `已派→作废`、`已派→挂起`、`候派→已派`、`挂起→已派`、`异常词→作废` 等 DBP3b 归一边全列。
- [ ] **陈行批转**（词表落地后同票执行，task_transition 唯一径）：异常词→作废 = T-0279/T-0280/T-0284/T-0285/DBP3b；异常词→挂起 = DBP3c-前端义务面（note：票面候派=ADM 重设计停泊）。批转后 `SELECT ticket,phase` 贴票。
- [ ] **沙箱自证**（沿用 DBP3b 沙箱注缝径 /tmp/dispatch-e2e-* 或同等）：构造 ①已派行+作废票面→零 spawn+行转作废；②已派行+完成票面→同①；③已派行+候派票面→零 spawn+行转挂起；④已派行+已批票面→spawn 原径不受影响；⑤已派行+票面无状态字段（如 T-0186 式老票）→照派不误伤。逐案断言输出贴票。
- [ ] **生产实证**：dispatcher restart 后 `~/devin-dispatch.log` 见否决行格式（票号+票面词+落相）；`META_TASK_READ=0` 回退态下新闸整段不执行（函数体在 DBP3b 闸内）。
- [ ] commit 格式 `[角色:施工][T-0292] …`；meta 仓提交推 origin/master（meta 仓施工直落 master 为该仓现行惯例，DBP3b 谱在案）。

## 范围外

- `task_scan_audit_one`/`task_scan_rework_one`/`task_scan_pass_one` 同款票态否决——指令仅点名 dispatch 径；如需同构另票。
- 不改 `meta_task` 表结构/函数签名/词表钦定集（补部=repo DDL 既有内容的部署重放，非新增）。
- 不杀任何在飞 headless 进程；T-0249/T-0266（claude-kimi-t8 实工行）与 ~20 张 SETTLE-SWEEP 已批票在飞会话不碰。
- `异常词` 历史存量行（~104，建行写手批装产物）不属本单清面。

## 参考

- 病灶锚：`~/devin-dispatch.log` 14:16:58 boot 扫「已派=25 逐个 spawn」段；`~/devin-headless-T-0215.log`（空转 131865 tok）、`~/devin-headless-T-0279.log`（stale 让位声明全文）。
- 码位：`/opt/meta/mbus-dispatch.sh` `task_scan_dispatch_one` @762、`status_field` @308、`enum_base_of` @138、`task_claim` @215、`ob_gate_once` @179。
- DDL：`/opt/meta/sql/meta_task.sql` 行 18-35（词表）、44-71（边集）、96-211（task_transition 语义——非法词落异常词告警不静默吞）。
- 票面词表：`/opt/meta/ops/ticket-state.enum`（单源；已批/待派/待施工=派标记，完成/作废=终态，挂起/候派=停泊）。
- 管理席指令原件：obl372（meta_obligation 行/mail 箱）。

## 实况注记（决策席 @2026-11-01 14:4x——施工锚位变动同步）

- **锚③④已由他径落实**：SG 生产词表 14:33 补部至 16 词+边集全列（含新增 `施工中→已派` 回队边——管理席 SETTLE-SWEEP2 径）；六陈行批转同拍落位（→作废×5/→挂起×1，owner=dispatcher）。施工对这两项=**核验确认**（SELECT 亲证即可，幂等重放不再必须，跑了也无害）。
- **锚①②⑤⑥仍为本单主体**：派前票态否决闸落码+沙箱五案+生产实证，不变。
- **撞单在案**：管理席同刻另立业主令 T-0292（meta派发池化+前端真源合一——远端撞号将重编 T-0293+），其钦定项④claim 票面闸与本闸语义同族——本单=现役 push 扫径急补丁；池化重设计为超集归彼单。本单范围外已含「不碰扫描架构」。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
