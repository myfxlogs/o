---
schema_version: 2
id: simdrill-entrypoint-concat-and-exempt-probe
kind: pitfall
risk: medium
status: verified
keywords: [simdrill, docker run, entrypoint, excell, 免演, 能力债, capability probe, 假阳, cron]
scope: [演习臂面, simdrill -matrix 手动/docker 调用, capabilityEscCheck 占用感知]
summary: 两个同日坑——①o-core 镜像 ENTRYPOINT=/bin/core exec 形，裸 `docker run img /bin/simdrill -matrix` 被拼成「/bin/core /bin/simdrill」跑成 core 非演习（容器 Up 10min 零凭据行，假死无声）；②能力债探针「enabled 模板须 7d 绿凭据」撞 excell⑤「词表零占免演」钦定——零占用模板结构性产不出凭据，探针会对其永火假阳。修法=①--entrypoint 显式给或走 scripts/simdrill_matrix.sh；②演习债只对 excell.Enumerate 非 Exempt 占用格模板成立。
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
