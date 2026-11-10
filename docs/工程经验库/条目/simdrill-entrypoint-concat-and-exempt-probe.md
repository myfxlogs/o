---
schema_version: 2
id: simdrill-entrypoint-concat-and-exempt-probe
kind: pitfall
risk: medium
status: verified
keywords: [simdrill, docker run, entrypoint, excell, 免演, 能力债, capability probe, 假阳, cron, escEdgeU, 销案重火, deadSince, ephemeral, fin_strategy_active_dedupe, retireEphemeral, basisSpotLeg]
scope: [演习臂面, simdrill -matrix 手动/docker 调用, capabilityEscCheck 占用感知, escEdgeU 未销才抑]
summary: 四个同日坑——①o-core 镜像 ENTRYPOINT=/bin/core exec 形，裸 `docker run img /bin/simdrill -matrix` 被拼成「/bin/core /bin/simdrill」跑成 core 非演习（容器 Up 10min 零凭据行，假死无声）；②能力债探针「enabled 模板须 7d 绿凭据」撞 excell⑤「词表零占免演」钦定——零占用模板结构性产不出凭据，探针会对其永火假阳；③escEdgeU 沿触发 deadSince 键占死不再发——「定性要码修」销案（分类≠修好）后信号成黑洞，唯重启回流；④演习 ephemeral 策略行 degraded 不退役——占 active_dedupe 键致同格复跑全撞（24h TTL 不赶日拍间隔）。修法=①--entrypoint 显式给或走 scripts/simdrill_matrix.sh；②演习债只对 excell.Enumerate 非 Exempt 占用格模板成立；③escEdgeU 死态期间每拍过 escalateUnresolved（DB 判未决——开着静默/销了重火/火完再静默），dedup 粒=未决事件非「沿」；④演习毕即退 retireEphemeral+Transition 留痕。
first_seen: 2026-11-10
last_verified: 2026-11-10
skills: []
evidence_refs:
  - type: server_fact
    source: docker ps on SG
    query: docker ps -a --format '{{.Names}} {{.Command}} {{.Status}}'
    observed_at: 2026-11-10T04:40Z
---

## 坑①：exec 形 ENTRYPOINT 拼接

o-core 镜像 `ENTRYPOINT ["/bin/core"]`（exec 形）。`docker run img cmd args`
替换的是 CMD 不是 ENTRYPOINT——argv 拼成 `/bin/core /bin/simdrill -matrix`，
跑的是 core 本体（core 忽略未知 argv 静默起服务）——容器 Up、零报错、
零 fin_exec_drill 行，假死无声。

修法：`docker run --entrypoint /bin/simdrill ...`，或统一走
`scripts/simdrill_matrix.sh`（已带 --entrypoint，SG cron 调用同径）。

## 坑②：探针不变量撞枚举豁免

`capabilityEscCheck` 初版对**所有 enabled 模板**要 7d 绿凭据；但
`excell.Enumerate` ⑤ 钦定「execfam 册内模板全格集空=免演登记，不产凭据」
——零占用 enabled 模板（收养未出水的 vol_sell 类）**结构性产不出凭据**，
探针会永火且缺项载「无绿演习凭据」误导处置（处置者会试图钻一个
不存在的格）。

修法：占用感知——演习债只对 `excell.Enumerate` 返回的非 Exempt 格
模板成立；零占 enabled 族健康度只由「14d 零评估产出」判（evaluator
枚举尝试含拒行——枚举着没候选=市场平窗非债）。占用判定唯走
Enumerate 真源，禁探针自查 fin_strategy 复述（枚举径演进探针不漂）。

## 坑③：escEdgeU 沿锁——销案≠修好的类销后黑洞

`escEdgeU` 原实现是沿触发：条件首拍为真→deadSince 落键+升件；此后
键占着，`escalateUnresolved` 不再被唤。「未销才抑」语义钦定 dedup
粒=未决事件（销了=新事证重升），实现却把它绑成「沿」——**销案而
条件仍真**（定性要码修类：分类≠修好）时信号成黑洞，唯进程重启
deadSince 清空才回流。歧态批类无碍（销案=修好=条件翻假=自然解锁），
能力债类销案=定性，债还在。

修法：dead 期间每拍过 `escalateUnresolved`——其内 DB 判未决升件，
开着静默/销了重火/火完再静默；deadSince 只记首死供恢复行陈窗。
测试锚=「销而未修→一拍内重火」（TestCapabilityEscCheck 处置行销
E1→E2 重升）。

同案附注：basis 演习格 `BTC-USD-270326` fail「现货乐器不在册」
非真债是派生缺——期货基键直查现货（BTC-USD 实有无对），产线
`lifecycle.basisSpotLeg` 从来归一 -USD/-USDG→-USDT（CEX 现货报价
币=USDT 族，业主口径复核）。演习 venue 册+腿构双消费 `hedgeSpotKey`
单源——旧锚⑥b 断言 fail=误锚（前提错非机理错），已转 pass 锚。

## 坑④：ephemeral 策略行占活跃去重键——复跑全撞

演习径建 `fin_strategy` ephemeral 行（state='degraded'）从不退役；
`fin_strategy_active_dedupe` 唯一键 `(venue,inst,direction,template_ref)
WHERE state<>'retired'` 下占键——同格复跑必撞（矩阵二跑 6 格全
`duplicate key`）。`retireStaleDegraded` 24h TTL 兜底不赶日拍间隔
（23.5h<24h——cron 臂每天必撞）。

修法：演习毕即退——`retireEphemeral` 三径 defer 共用（UPDATE
degraded→retired + `fin_strategy_transition` 留痕
reason='drill_ephemeral_done'；retire.go 律裸 UPDATE 零留痕禁）。
测试锚=TestMatrixRerunIdempotent 同格连跑双绿。

## 证据

- check: SG `docker ps` 见 affectionate_roentgen `/bin/core /bin/simd…` Up 10min；
  同拍 `fin_exec_drill` 30min 窗零行
- result: 杀跑偏容器+`--entrypoint` 重跑——6 格 4 pass 2 fail（真债如实）；
  修后探针测试日志 `capability_vol_sell_strangle 缺项=[14d 零评估产出]` 单缺
- ref: `scripts/simdrill_matrix.sh`；`cmd/core/meta_esc.go` 占用感知段；
  `internal/platform/excell/excell.go` Enumerate⑤

## 反证条件

- check: `docker run img /bin/simdrill` 不拼接直接执行（镜像改 shell 形
  ENTRYPOINT 或无 ENTRYPOINT）
- falsifies_when: excell⑤ 免演语义撤除（零占也强制产凭据）
