---
schema_version: 2
id: guard-close-stuck-corpse-row
kind: pitfall
risk: medium
status: verified
keywords: [fin_guard_close, closing滞留, 守卫环自残探针, 簿面陈态, 尸体行, fin_guard_suspend, 开仓权挂起, guardlift, 解冻判词, loop_alerted_at, 只出不进]
scope: [internal/guard/loop.go 自残探针, fin_guard_close, fin_guard_suspend, tools/guardlift, 仓巡升件面]
summary: closing 滞留升件裁法——先核 venue/账面双清：账实双清=簿面陈态非钱险；处置序=尸体行翻终态→guardlift 解冻开仓权（倒序=探针下拍复挂）；挂起语义=只出不进非全冻
first_seen: 2026-11-07
last_verified: 2026-11-08
skills: [裁定席, 决策席]
evidence_refs:
  - type: doc
    ref: internal/guard/loop.go（loopProbe 阈判定在 SQL 谓词内+熔断幂等每拍保活）
  - type: doc
    ref: tools/guardlift/main.go（解冻唯一径 CLI——-by 闭集=裁定席|业主，-evidence 禁空）
  - type: doc
    ref: docs/tasks/T-0462-陈尸收殓-guard_close终态翻行与287112冻结裁定与残差销.md
  - type: server_fact
    source: postgres
    query: SELECT id,note FROM ops_event WHERE note LIKE '仓#34%' AND event='升件'——2026-11-07 10:22~11-08 04:22 滞留 13.1h→31.1h 每 2h 连鸣 10 报（ops_event#3280/#3350/#3433/#3447/#3476/#3486/#3495/#3584/#3811/#3917）
    observed_at: 2026-11-08T04:30:00Z
  - type: server_fact
    source: postgres
    query: ops_event#3333 决策席裁决——T-0424 处置终了五证全锚：venue 实拍 CORE 残 0.0098/持仓 0/Earn 无 CORE/账面四步落毕/pos34 结案 closed settled-0.3089
    observed_at: 2026-11-07T11:27:30Z
---

## 事实/症状

- 仓#34 `fin_guard_close` 滞留 `closing` attempts=5——自残探针阈
  （attempts≥3 或滞留>2h）每 2h 连鸣升件，13.1h→31.1h 十报
  （ops_event#3280~#3917）；同拍熔断 `fin_guard_suspend` 挂策略
  #287112 开仓权（只出不进）；
- 实际账实双清在案：T-0424 裁决（ops_event#3333）五证——venue 实拍
  CORE 残 0.0098/持仓 0/Earn 无 CORE/账面四步落毕/pos34 已结案
  closed——残留只是 `fin_guard_close` 行未翻终态的簿面陈态。

## 根因或策略解释

- 平仓已终但守卫行不翻终态=「尸体行」——探针 SQL 谓词
  `state='closing' AND (attempts≥阈 OR 滞留>阈)` 不辨死活，陈态行
  恒命中；
- 熔断语义：挂起=禁新开、平仓权保留（只出不进）——非钱险冻结，
  是防「机器发现不对但不知为何照开照烧」的保守侧（T-0423）；
- 连鸣是钦定节流设计（loop_alerted_at 同窗一报，熔断幂等每拍保活）
  ——处置落行携升件#id 引证即停鸣，复鸣=同案续拍非新病灶。

## 处置

- 裁序铁则：先核 venue 实拍+账面结案双清——双清则判「簿面陈态
  非钱险」，未双清则按真卡死走钱径处置（本席不越钱径写域）；
- 收殓序：①尸体行翻终态（守卫域写面手续钦定，T-0462 票径执行）
  →②`tools/guardlift -strategy <ref> -by 裁定席 -evidence <判词锚>`
  解冻开仓权——**倒序无效**：尸体行未翻则探针谓词下拍复挂；
- 判词携案锚：verdict_ref/ops_event 裁行引 fin_eng_lesson#id+
  升件#id 链——guardlift `-evidence` 禁空，无判词不解冻。

## 证据

- check: `SELECT id,note FROM ops_event WHERE note LIKE '仓#34%'`
- result: 升件十连鸣（滞留 13.1h→31.1h）+开仓挂起行#3279+
  决策席/CTO 处置行#3312/#3390/#3435/#3627/#3795/#3816/#3963/#4078
  全判「簿面陈态非钱险，解冻唯径=裁定席判词/业主令」转本席
- check: T-0424 裁决 ops_event#3333 五证
- result: venue 实拍残 0.0098/持仓 0/Earn 无 CORE；pos34 结案
  closed@10:45:10Z settled-0.3089——账实双清
- ref: fin_eng_lesson#12（本簇案）+docs/tasks/T-0462（收殓票已批在派）

## 反证条件

- check: 尸体行翻终态+解冻落地后，同仓升件是否复鸣
- falsifies_when: 翻终态后探针仍升=谓词面漏终态集（病灶升级为探针
  缺陷）；或 venue/账面核出未清残迹=「簿面陈态」定性证伪，按真
  钱径事故重裁

## 适用边界

- 仅 closing 滞留类升件裁法；close_failed 终态陈尸（pos2 案）须先
  核 venue 面再翻——裁法同序但核证门槛更高；
- 解冻判词只管开仓权——平仓权从未被挂（只出不进语义）；
- 策略若另因退役/风控在挂，本条目解冻序不适用——逐挂逐裁。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（候选=「fin_guard_close closing 陈态行计数」：
  state='closing' 且 now()-last_try_at>24h 行数——守卫域 gauges.tsv
  面无适配表，T-0462 收殓落地后随守卫观测面补建）
- class: n/a（候选：淤积）
- query: n/a
- threshold: n/a
