# T-0335 业主打断率量表——owner_interruption 入 gauges 册

- 状态：**完成**（决策席验收通过 devin @ 2026-11-04——合 master，判词在案）
- 领取：devin-o-worker-t0335 @ 2026-11-04（无头施工 run）
- 级别：P3 小件（量表单行+口径落字）
- 族：T-0332 巡检面谱系——治理面度量（「全替」目标的收敛 KPI 本体）

## 任务

`scripts/meta/gauges.tsv` 增一座量表 `owner_interruption`：

- **口径（业主裁定在案，一字不改）**：计**阻断性需求件**——系统卡住等业主出面才走得动的件；**不计**业主在场治理活动（讨论/问进度/授权/巡视——在场≠被需要）。判据：业主离场系统是否停摆。
- **构成**：`fin_condition_gap` 新行（凭据/注册/条件请求）+ `ops_event event='COS上行'` 行（升件打包件待业主裁），trail 30d。
- **型**：`sql_ro`（o_fin_ro 白名单径——condition_gap 经 fin_0108 已授；ops_event 读面施工核实，不可读走 fin_0109 补授先例或换口径并落票注明）
- **阈值**：>5 → 升件（首日基线，事后可调——「打断率上跳」本身就是要浮上来的病）
- **升件模板**：note 带总数+两源分拆数（gap 件数/上行件数）

## 验收锚

1. `gauges.tsv` 行在册，tick 实跑 rc=0 出真值——今日 30d 计数实录落回填
2. 口径落字——「阻断性需求」定义入档（tsv 注释或谱系档），防口径漂移成「交流量」
3. 零误升实证——当前真值 ≤ 阈时滞回闸不升件（烟跑实录）

## 红线

- 只读度量——不动任何业务面/执行径
- 禁计入业主在场活动（交互会话本就不该有数据源去数它）
- 阈值不许设 0——系统初跑期打断率 >0 是常态，>5 才是信号

## 结果（施工回填）

- 分支：`t/0335-owner-interruption`；涉及面=`scripts/meta/gauges.tsv`+`scripts/meta/gauges_tick.sh`+`docs/异常类谱系.md`
- 实现：
  - `gauges.tsv` 增 `owner_interruption` 行（class=淤积，sql_ro，30d trail 两源合成计数，阈列=6⇔票「>5 升件」——tick 判据 v≥阈）+ 口径业主裁定原文注释四行；
  - `gauges_tick.sh` 增可选⑦列 `detail_query`（dq）——升件拍按同型取数径现取填模板 `{d}`；缺席/取数败→`{d}` 落空不阻断升件本体；`{v}/{d}` 改 bash 内建替换（detail 串含 `/`，sed 定界符会断——升件链烟跑实证）；
  - `docs/异常类谱系.md` 注册表格式行补⑦列 + 「治理面度量」口径落字段（锚②双落：tsv 注释+谱系档）。
- 取数径核实（票面「ops_event 读面施工核实」）：fin_0106 白名单**已含** `ops_event`——o_fin_ro 实测 `SELECT` 通过，`fin_condition_gap` 经 fin_0108 已授，**无需** fin_0109 式补授、无需换口径。
- 测试证据：
  - tick 实跑（`GAUGES_TSV`=本支册+prod env）：rc=0，全册（含新增表）零 fetch_error；`owner_interruption` 真值=**0**（detail=`gap=0/上行=0`，30d 实录 2026-11-04）→阈下未触、滞回闸不升件，`ops_event` 升件行实测 0——锚①③；
  - `{d}` 升件链烟跑（隔离 `o_test@localhost:5434`）：强制触阈假表→`ops_event` 升件行 note=`smoke=7（gap=4/上行=3） 阈=6`——总数+两源分拆+阈齐（升件模板锚）；
  - check_all 全链绿戳待本 commit 重跑落（改动含 .sh 非 docs-only 面）。
- 偏离说明：票面言「gauges.tsv 增一量表」，但升件模板「note 带总数+两源分拆数」需 tick 供第二值——原架构仅 `{v}` 单值替换，故同族文件内最小扩可选⑦列（向后兼容：旧六列行 dq=空，零行为差——全册实跑证）。
