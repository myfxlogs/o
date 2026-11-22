---
schema_version: 2
id: wordform-gate-anchor-position
kind: pitfall
risk: high
status: verified
keywords: [词面闸, ops_event note, 正则谓词, LIKE, 锚定位, 方言, 全角冒号, 前缀互吞, aseat, dedup, unit=, 完工回执, 复审过, colon, ghost ticket, 产消守恒]
scope: [scripts/cognition, scripts/land_queue.sh, cmd/core, ops_event 消费面, 唤起/收口/回执契约]
summary: 词面闸一律锚定位判——散文 note 非机器接口；写端产契约形（键=值/邻接/行首），读端按位取+尾界防前缀互吞；同族五发实证（colon/aseat双修/dedup幽灵/unit界缺/selfgap锚缺）
first_seen: 2026-11-22
last_verified: 2026-11-22
skills: [施工席, 研究席]
evidence_refs:
  - type: doc
    ref: ops_event#32790（O-32638 普查表先行件——四面全谱+逐条判定）
  - type: doc
    ref: scripts/cognition/dispatch_research.sh:57（^求知[:：]|研究席[:：] 锚+双冒号=修后范本）
  - type: doc
    ref: scripts/cognition/sessionpulse.sh:47（unit=$unit([^0-9]|$) 尾界=对照范式）
  - type: doc
    ref: scripts/land_queue.sh:141（^复审过[[:space:]]*sha=<tip> 锚行首+sha绑=范本）
  - type: commit
    ref: a0303adb（dedup-taken-anchor——trigger= 取键位修幽灵票）
---

## 事实/症状

同日五发同族方言毒坑，全部=「散文 note 当机器接口+词面闸不锚定位」：

1. **colon 案**：dispatch_research 读端只认半角 `:`，产票写全角 `：`——
   票产了消费者看不见，求知票滞 44h+。
2. **aseat 泛词案**：land_queue `review_ok` 排除式泛词「复审过」误排
   回执行——作者席取行被复审行顶掉，分席判据失真。
3. **回执正文污染案**：收窄后仍被回执/处置正文里「复审过 sha=」字面
   误排——排除式 `note !~ '复审过'` 拦不住正文引用，须锚行首。
4. **dedup 幽灵票案**：研究机触发去重泛词面匹配把「留拍」键误判已唤
   ——幽灵票 tk#32021/tk#32274 实证，修法=trigger= 取键位+尾界。
5. **普查新获（O-32638）**：unit= 消费族 12 处 `LIKE %unit=X%` 无尾界
   （sessionpulse/doing_face 已带 `([^0-9]|$)`=对照范式在案）；
   selfgap_esc `求知[:：]` 无锚 vs dispatch_research `^求知` 锚——
   产消谓词不对称（句中「工单/x:…求知：…」会误升 learn_gap 而研究机
   不可见）；aseat 支名 `note ~ '$bre'` 无右界——处置席销件散文
   「fix/x滞因」直引支名可劫持作者席取行（30d 实测 281 行句中回执词
   vs 75 锚首）；closeoutprobe `[[:space:]]`/Go `\s` 双侧漏 U+3000
   全角空格方言缝。

## 根因或策略解释

ops_event.note 是**散文账**——会话写判词/销件/回执都是自然语言。
凡拿 note 子串/正则当「这行是某类契约行」的判据，词面就同时是
**产出面**（契约声明）和**引用面**（散文提及）——不锚定位时引用面
必然反噬：回执正文引用「复审过 sha=」、销件散文引用「完工回执+
支名」、判词引用「unit=X」都会被当契约命中。

结构共性：**写端产契约形（键=值/邻接/行首锚），读端必须按位取**——
`^` 锚行首、`key=` 键位、`#id` 引用位、邻接 `A[空白]*B`、尾界
`([^0-9A-Za-z_-]|$)` 防 `unit=o-x` 吞 `unit=o-x4`。缺一即同族新坑。

## 处置

- **锚修（机械径）**：谓词补 `^`/键位/邻接/尾界——同今天五件修法。
  判据=对照在案范式件（sessionpulse unit 界、dispatch_research
  `^求知[:：]`、land_queue `^复审过 sha=`、closeoutcert `^完工回执`、
  closeoutprobe `工单#N\s*完工回执` 邻接）。
- **方言宽容**：全半角冒号 `[:：]`、U+3000 全角空格入空白类
  `[[:space:]　]`/`[\s　]`。
- **seat/结构位锚**：作者席判据加位锚（支名右界 `[@[:space:]=]`）。
- **结构性项另立案**：散词交替谓词（evalfloor `裁定|审批|假想|…`）
  无锚位可依——方向保守（假新鲜不催单）故留判不硬改。
- **回归锚**：`scripts/cognition/unit_boundary_test.sh` 16 案钉全谱
  界锚在场+旧无界形缺席。

## 证据

- check: `bash scripts/cognition/unit_boundary_test.sh` → PASS=16 FAIL=0
- check: `psql` 实证 `note ~ 'fix/relay-gate-order([@[:space:]=]|$)'` 对
  「fix/relay-gate-order滞因」=f、「fix/x@sha」=t（支名右界语义）
- check: 30d ops_event 实证句中回执词 281 行 vs 锚首 75 行（污染面量化）
- ref: ops_event#32790（普查表）、#32720/#32721（断点实证）、
  #32346/#32495/#32530/#32533/#32556/#32620（本族四案全链引证）

## 反证条件

- falsifies_when: 出现「契约词合法出现在散文正文且读端不得命中」的
  反例设计——即某消费面**有意**靠散文引用判活（如升件#id 引证族
  「引而未决」语义）——此时该面应显式声明引用语义而非误判污染。

## 适用边界

- 适用：ops_event.note/event 一切机器消费谓词（SQL ~ / LIKE /
  Go regexp / shell grep）。
- 不适用：纯展示层过滤（ops_relay/feed_gloss 词面=人读增强非闸）；
  seat/event 枚举位匹配（词表受控非散文）。
- 引用语义例外：升件#id 引证族、工单#N 销账引用**就是设计语义**
  （散文引用=处置动作本身），不按污染修。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a
- 豁免由：词面闸污染无单一 SQL 可量——断点实证面=ops_esc_pending
  未决簇+landq_starved 升件流（O-32638 面④已实证在案）；回归闸=
  unit_boundary_test.sh 静态锚（非时序量表域）。
