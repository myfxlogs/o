# T-0469 总协调网关 o-gateway——事件驱总线 + 席权调度 + 收敛/升件一本账

- 状态：呈批（设计方案呈 CEO 裁决——业主令 2026-11-08：「直接上 V2，1 步到位」「meta 的模式，网关做总协调，谁起谁退谁做什么都由网关调拨」「网关改造全部走外派外审，不用无头」）
- 归口：呈批→CEO 裁后，外派施工+外审裁决径
- **前置义务**：施工前先查 /opt/meta 弃养死因（目录已空）——死因结论落票面，约束条款随案走，防同死法

## 病灶（2026-11-08 全日实证）

1. **有探测无扳机**：deploy_window_due 三态机完备，「窗清→开火」无承接
   ——OCI 面 0441 merge 后数小时无人发，CTO 三班处置全对仍断链。
2. **调度碎裂 9 壳**：派单（dm_tick 班扫+派单行+systemd-run）、候水
   （dispatch_water）、水位闸（dispatch_worker）、依赖闸（dm_depgate）、
   退班账（dm_shiftend）、交接哨兵（dm_handoff）、升件径（escalate/
   ops_esc）、巡检（gauges_tick）、收敛（mirror_push/deploy_window_due）
   ——同一套「谁起谁退」职能碎成九份，各席位注意力轮流接盘=今日断链根。
3. **席位注意力依赖**：所有收敛/生命周期动作的最终承接都是「某席恰好
   在班恰好看到」——席退/水位满/换班即断链，今日实证三处。

## 定位

**o-gateway = 公司总协调面唯一执行网关**（meta v2）。三本账归一：

- **席权账**：谁起谁退——spawn/retire/补拉/水位/死席清扫全权；
- **路由账**：谁做什么——已批件机派（派单行+拉起同拍）、交接对账
  （交必有接，断链即分派认领）、候水账；
- **收敛账**：推已绿 sha→发 image/oci 面→验活→部署行；升件认领
  路由（升件→归口席→未认领再升/冻结）。

**边界红线（永不逾越）**：gateway 只做**机械调度与钦定执行**——
判断永归席位：票优先级=CTO 裁域、升件定性=归口席裁域、部署该不该
=deploy_window_due 判据件裁。gateway 读判定结果去执行，不自立判据。

## 架构

### 形态

常驻 systemd user service `o-gateway.service`（Type=simple 常驻，
Restart=always，OnFailure 升件）——事件驱非轮询：

| 事件源 | 机制 |
|---|---|
| ops_event 新行 | PG LISTEN/NOTIFY（无则短轮询降级） |
| `.git/check-green-*` 落戳 | inotify 文件事件 |
| 票队/派单态变化 | ops_event 行触再评 |
| systemd 席态 | 周期活性采集（退化为拍,非主驱） |
| 时钟兜底 | 低频拍（30s-60s）防事件漏缝 |

### 一本账状态机

```
IDLE → SENSE(事件+采集) → CLASSIFY(席权|路由|收敛|升件) →
  席权:  已批未领+水位有余 → spawn拉起;rc75 → 候水账;
         席死/孤儿 → 清扫+补拉判;
  路由:  交行无接超阈 → 归口席分派+升件;派单行无拉 → 补拉;
  收敛:  净发窗+绿戳 → push→deploy面→验活;
  升件:  未认领超龄 → 再升一级(归口→CTO→CEO→业主);
→ LEDGER(ops_event seat=gateway 全链行) → IDLE
```

- **幂等键**全动作钉死：spawn=票号单实例、push=sha、deploy=sha:face、
  升件=esc id——重事件零重复动作；
- **败即升件+面冻结+不自愈重试**（处置权归归口席——防循环炸）；
- **崩溃安全**：全状态可从 ops_event 账重放重建，Restart 后接续；
  ledger 先行后动（写账→执行，崩在中途账上有痕）。

### 降级安全（单脑险对冲）

- gateway 死 → **席位自扫描兜底层不灭**：dm_tick/ceo_tick/席位班扫
  保持原样（不动一字）——公司降回「席位注意力制」慢转不瘫；
- gateway 活性自身被监：heartbeat 行每拍落 ops_event，
  gauges_tick 添 `gateway_hb` 行（停鸣>阈=升件——谁看门人=巡检闸）；
- kill switch=stop+disable unit 即回现行制，零迁移成本。

## 判据

- 事件驱实证：戳落文件→≤事件延迟内 PLAN（非等拍）；
- 席权实证：已批票→spawn 拉起同拍派单行；rc75→候水账态正确；
- 交接实证：人造断链→归口席分派行落账；
- 收敛实证：净发窗+绿戳→push→deploy→VERIFY 全链序正确；
- 冻结实证：人造升件未决→收敛面全面零动作；
- 幂等实证：同事件重放零重复动作；
- 降级实证：gateway stop→席位班扫照旧接手（兜底不瘫）；
- 崩溃实证：kill -9→Restart 后状态接续无重复执行。

## 范围

- `scripts/meta/gatewayd.sh`（常驻主循环：事件采/分类/状态机/调钦定件）；
- `deploy/o-gateway.service`（常驻单元，Restart=always）；
- `scripts/meta/gateway/*.sh`（四类动作子件：席权/路由/收敛/升件）；
- gauges.tsv `gateway_hb` 行；
- `scripts/meta/tests/gateway_test.sh`（桩测：事件/席权/路由/收敛/
  冻结/幂等/降级/崩溃八组）；
- mirror_push.timer 退役注记（功能并入）；
- meta 弃养死因查+结论落票；
- 本票回填。

## 禁线

- 禁动判据件/席位职卡/既有闸/dm_*/deploy_* 本体——gateway 只调用；
- 禁自立判断语义（优先级/定性/该不该——永归席位）；
- 禁自愈重试/禁冻结期强发/禁无戳部署/禁推未绿 sha；
- 禁碰钱径；禁用无头席位施工/裁决（业主令：全件走外派外审）；
- 外派收口移交外审席——勿 merge 勿部署勿启用 service。

## CEO 裁决空位

- [ ] 一步到位 V2 全谱放行（事件驱常驻+席权调度+收敛+升件归一）
- [ ] 边界红线认可（机械调度归网关/判断永归席位——派单=已批后机械拉起，
      不裁优先级；升件=路由归口不裁定性）
- [ ] 降级安全认可（gateway 死=席位自扫描兜底不瘫，双轨共存期认可）
- [ ] mirror_push.timer 功能并入 gateway 认可
- [ ] meta 弃养死因前置查为施工首步认可
- [ ] 外派施工+外审裁决径遵令确认
