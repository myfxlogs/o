---
schema_version: 2
id: dm-ledger-rc2-unknown-status-batch-starvation
kind: pitfall
risk: high
status: verified
keywords: [dm_ledger, 票翻面饥饿, rc2未识别, 批帽 MAX_BATCH, 状态行异形态, ORDER BY升序, 机械轨A面, 动作集饿死]
scope: [scripts/meta/dm_ledger.sh, 机械轨A面扫描, docs/tasks/done 古票面]
summary: 三态 rc 当两态用——status_done rc2（未识别状态形）判不过「rc0才跳」的收敛守卫，177 件 done/ 古票异形态恒入动作集吃满批帽 MAX_BATCH=40 且升序排新票前，A 面结构性饥饿：新完成票（T-0414 实证/T-0415 在飞）永不进集零翻页。修法=钦定判 A：rc2 视同不属本轨翻面责任直接 continue，rc1 确认识别非终态才入集
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [施工席, 决策席]
evidence_refs:
  - type: doc
    ref: scripts/meta/dm_ledger.sh
  - type: server_fact
    source: postgres
    query: ops_event 终态票扫描对拍 done/-only 票 status_done 三态分布——rc0=125 rc1=7 rc2=177（332 终态票 2026-11-07 实测）
    observed_at: 2026-11-07T00:00:00Z
  - type: doc
    ref: docs/tasks/T-0417-dm_ledger票翻面饥饿-古票rc2恒入集吃满批帽.md
---

## 事实/症状

- T-0414 merge 上 master 后手动 `dm_ledger.sh flip` 两拍，输出同
  `[dm-ledger] 批帽 40 到——余票下拍续`：T-0414 票两拍皆未翻，
  git 零 commit、工作树零变、ops_event 零新行；
- T-0415（裁决通过+已并 master）在 tasks/ 滞留同症——批次被古票
  吃满永不轮新；
- 实机对拍（2026-11-07）：332 终态票中 done/-only 票
  status_done 分布 rc0=125 / rc1=7 / **rc2=177**——rc2 集
  体量 4.4× 批帽，饥饿为常态非边缘。

## 根因或策略解释

- A 面扫描 `ORDER BY l.ticket` 升序——T-0002 起全史票排新票前；
- 收敛守卫 `[ -z "$tf" ] && status_done "$df"` **把三态 rc
  当两态用**：仅 rc=0（已识别完成态）跳票；rc1（识别非终态）
  与 rc2（未识别形）同落「入集」支；
- `status_done` 对古票面三形态全不认（`| 状态 | 完成 |` 表形、
  `### 状态：审计通过` 题头形、无状态字段形）→ rc=2；
- 锁段 `flip_status` 同锚同式→同 rc2→不写不翻
  （`moved=0 && fsrc≠0` 不入 DONE_TS）→ 每拍零产出却占满
  批帽——**恒败件每拍重集=结构性饥饿**；
- 同文件 B 面预扫早已写对（`rc1 才翻 rc2 不动（保守）`）——
  A 面独漏，同函数两调用面判据漂移。

## 处置

- 钦定判 A（T-0417）：A 面守卫改 B 面同 idiom——
  `src=0; status_done "$df" || src=$?; [ "$src" -eq 1 ] || continue`。
  rc2 未识别形视同「不属本轨翻面责任」直接跳；古票异形态翻面
  非机械轨职（认不得的形不碰=机械轨本分）；
- 桩测⑥饿殍景：40 rc2 件前置注入吃帽→新完成票仍入集翻页
  （红→绿钉死回归）；
- 淘汰项 B（状态形谱扩充收 `| 状态 |` 表形/`### 状态：` 题头形）：
  谱张面大、古票史形繁、翻它们无营运义——不荐在案。

## 证据

- check: `ops_event` 终态票扫描 + done/-only 票逐件 status_done 三态
- result: rc0=125 rc1=7 rc2=177；修后桩测六景全绿，实机 flip 单拍
  翻 T-0415 落 commit+push
- ref: T-0417 票面病灶实证节 + dm_ledger_test.sh ⑥饿殍面

## 反证条件

- check: 找出一张 rc2 异形古票确需机械轨翻面的营运由
- falsifies_when: 若异形态古票状态翻完成有营运价值，则钦定判 A
  语义破——改走判 B 形谱扩充（条目照改不退）

## 适用边界

- 守卫只管 df-only 票（`-z "$tf"`）——tasks/ 票 rc2 照入集
  归档 mv 但状态行不写（内容原样保，移位不翻字，合「不碰」义）；
- 判据层通则：**rc/枚举型守卫必穷举全分支**——「识别态」与
  「未识别态」同落一支=把不可判当可判，fail-open 漏进动作集。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a
- 豁免由：meta 机械轨簿记面坑非产线数据面——候选量表=「批帽到
  而 DONE_TS 空」连拍计数（dm_ledger out 批帽行+零 commit 对拍），
  待入表时随调度可靠性票补建。
