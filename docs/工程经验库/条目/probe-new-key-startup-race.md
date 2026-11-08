---
schema_version: 2
id: probe-new-key-startup-race
kind: pitfall
risk: medium
status: verified
keywords: [探针, 新挂键, 启动竞态, 首见宽限, 覆盖失明, 观测盲, md_ticks, ActiveSetUnion, legset, 腿覆盖探针, T-0451]
scope: [hunger 腿覆盖探针, 所需集类探针, md_ticks 新鲜判, 策略立仓即报]
summary: 所需集×新鲜行类探针对「刚入所需集的键」即时判盲=结构性假阳——入集→订阅→采集→桥投影→消费面落行全链延迟分钟级，键龄 0.9s 即报产候选票噪音。所需集类探针必须带首见宽限（键龄<链路延迟上界不判盲），宽限过仍盲照报不误。
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: []
evidence_refs:
  - type: task
    ref: T-0451（观测盲:ZEC-USDT——funding-hedge 派生腿新挂即报根修）
  - type: doc
    ref: internal/portfolio/hunger/coverage.go ProbeLegCoverage（coverNewGraceMin 宽限件）
  - type: server_fact
    source: postgres
    query: SELECT id,ts,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='ZEC-USDT'
    observed_at: 2026-11-08T02:30:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

T-0451 升件 `观测盲:ZEC-USDT`（ops_event #3449）：腿行情覆盖失明——
live/paper 策略腿所需行情键近 10min md_ticks 零新鲜行。生产取证：
`fin_strategy` `ZEC-USDT-SWAP` funding-hedge 新实例 created_at=
2026-11-07 16:24:12.08，hungerTick 探针拍 16:24:12.98——腿键入所需集
（ActiveSetUnion FundingHedgeSpotArm 派生）与判盲间隔 0.9s；订阅→
采集→反向桥→桥投影→SG 落行全链延迟分钟级，首见键判盲=结构性假阳。
链路三段对拍（订阅 tickGate/镜像 refreshSets funding 臂/写入闸）零断点，
SG/PG-md md_ticks 双端 fresh 实证——病灶在探针判时缺宽限不在归一链。

## 根因或策略解释

- **所需集判据与供价时延不同步**：「键∈所需集」是瞬时谓词，「键有新鲜
  行」依赖异步管道（collector actTick 30s×2 端+反向桥+OKX 首推+写批+
  投影≈分钟级，部署重启 backoff≤60s+对账另计）。探针每拍对拍两集
  差即报，未给新入集键留管道送达窗——键龄<链路延迟上界时判盲必然
  为假阳，与链路健康与否无关。
- **陈行放大器**：md_ticks 为状态表（upsert 末帧覆盖）——键退集后
  断供，行龄随退集时长增长；再入集瞬间必呈「陈行盲」，假阳概率被
  放大到每次腿换手/策略重开。
- **同类面**：凡「所需集/活跃集 × 新鲜行/存活行」对拍探针（腿覆盖/
  依赖对枚举/feed 死闸同族）同构风险——新元素入集首拍即被判死。

## 处置

`internal/portfolio/hunger/coverage.go`：`coverNewGraceMin=15min`
新挂腿宽限——`Sensor.coverSeen` 首见戳册（hungerTick 单协程串行免锁；
键离集除册复入重计），宽限内不判盲、宽限过仍盲照报不误；判据本体
（coverFreshMin/签名/类册/24h 去重）零动。宽限取链路延迟上界留裕量
（≪6h 拍周期故检测迟滞可忽略）；内存册重启清零=全键重计宽限
（有界迟报一窗，在案取舍）。

## 证据

- check: `go test ./internal/portfolio/hunger -run TestProbeLegCoverageNewLegGrace`
- result: 修前 FAIL（首见 0s 键即报 覆盖失明=1 升件=1）→修后 PASS
  （宽限内零报+宽限过仍盲照报双点）
- ref: T-0451 收口 commit（t/0451-zec-usdt 枝）

## 反证条件

- check: 生产部署后同签名病灶复发——新挂腿键龄>15min 仍判盲上报
- falsifies_when: 宽限内（键龄<15min）仍产生 `观测盲:<inst>` 升件行——
  说明首见戳册未生效或病灶另有根因（如订阅面真断）

## 适用边界

- 仅覆盖「键新入所需集」场景；所需集持续成员的断流盲不受宽限影响
  （首见戳已旧照报）。
- 内存册=进程重启清零——重启后存量盲键让位至多一窗（宽限+一拍），
  部署批稀有+链路重启本需预热，取舍成立；若重启频发致盲报饿死，
  应升级持久化首见戳（另票评估）。
- 宽限值 15min 绑当前链路延迟上界——actTick/反向桥/投影节拍变更
  时须复核（订阅同步拍调大则宽限同步复核）。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：探针判时缺陷非量表可测运行指标——验收锚=
  桩测红绿双点+生产升件行随停，非阈值型量表）
