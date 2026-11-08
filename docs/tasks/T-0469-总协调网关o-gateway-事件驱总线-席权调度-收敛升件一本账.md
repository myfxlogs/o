# T-0469 部署收敛网关 o-conductor——v0 轮询壳：推送→部署→验活全链归一

- 状态：呈批（设计方案呈 CEO 裁决——业主令 2026-11-08：「做网关，出整体设计方案给 CEO 审，审完交回；网关改造全部走外派外审，不用无头」）
- 归口：呈批→CEO 裁后，外派施工+外审裁决径
- 业主令原文：「做网关吧，出整体设计方案，给CEO审，审完交回给你，网关改造全部走外派外审，不用无头」

## 病灶（2026-11-08 全日实证）

1. **有探测无扳机**：deploy_window_due（T-0398/T-0437）三态机完备，但
   「窗清→开火」无任何席位职卡/机制承接——OCI 面 0441 merge 后 3h+
   无人发，CTO 三班处置全对仍断链（席位注意力依赖病，与跑步机同族）。
2. **判据碎裂于散件**：推程=mirror_push timer，发窗=deploy_window_due
   探测无执行，迁移=deploy 内嵌随窗——三个收敛动作各自为政，无状态机
   一本账，窗与窗之间靠运气协调。
3. **v1 事件驱的入口缺**：15min 轮询是上限不是目标——戳落/merge/
   升件皆事件源，需一个常驻收敛口承接，不能靠再叠 timer。

## 设计方案（呈批本体）

### 定位

**o-conductor = 部署收敛面唯一执行网关**。上游一切产出（merge/绿戳/
升件态）经它收口，下游一切部署动作（push/image/oci/迁移/验活）由它
编排。席位继续裁「该不该」——conductor 只执行「已判定该发」。
**不发明判据**——全部复用钦定判据件，conductor 是编排层不是新法官。

### 形态（v0=轮询壳，v1=事件驱另票）

systemd user service `o-conductor.service`（Type=oneshot 随 timer 拍起，
`o-conductor.timer` 15min 拍沿 mirror_push 旧拍）。单实例收敛口——
本服务即取代 mirror_push.timer（功能并入，非并存）。

### 输入面（只读采集）

| 源 | 取数 |
|---|---|
| `.git/check-green-*` | 绿戳集+mtime（同 mirror_push 口径） |
| git | origin/master 位、HEAD、ff 关系 |
| `deploy_window_due.sh` | faces=image|oci 净发窗集+挡件清单 |
| ops_event | 升件未决/部署行/冻结语义（事故期冻结断言） |
| systemd/锁面 | deploy 锁态、worker 在飞面（判挡件用） |
| 三臂部署基线 | SG 镜像 sha+OCI 臂 sha（同 deploy_window_due 双源） |

### 状态机（一本账）

```
IDLE → EVAL(采输入) → PLAN(净发窗+可推戳) →
ACT_push(推已绿sha:master) →
ACT_image(faces∋image→deploy_image.sh) →
ACT_oci(faces∋oci→deploy_oci.sh) →
VERIFY(健康探针+部署行落账) → IDLE
任一 ACT 败 → ESCALATE(升件+该面冻结) → IDLE
```

- 幂等键=`<sha>:<face>`——同键已部署行在案即跳过，重拍零重复动作；
- 顺序钉死：push 先行（未并码禁部署——deploy_oci 既有闸同语义），
  镜像/OCI 按 faces 净集各发，不交叉；
- 冻结断言：**升件未决/事故期→全面不发**（显断言，票面新增，
  继承 deploy_window_due fail-closed 语义再收紧）；
- 败即升件+面冻结+不自愈重试（防循环炸部署——处置权归处置席）。

### 输出动作

`git push <已绿sha>:master`（复用 mirror_push 三断言选戳）→
`deploy_image.sh`（faces∋image）→ `deploy_oci.sh`（faces∋oci）→
健康探针（容器自报镜像/臂探针/迁移水位）→ ops_event
`seat=conductor` 全链落账行（EVAL/PLAN/ACT/VERIFY/ESCALATE 五态）。

### 安全闸

- repolock/部署锁先敬后动（锁持=本轮跳过非强抢）；
- 零凭据新增（沿用既有 ssh/env 径）；
- kill switch=disable unit 即停，单点可关可回退到现行散件；
- 钱径面零涉；判据件（mirror_push/deploy_window_due/deploy_*）一字不改，
  conductor 只调用——判据改了走判据自己的票。

### v1 预留（另票，本票不建）

mbus/事件订阅面（戳落即发+merge 事件即评+升件即冻）——/opt/meta 已空，
事件基座需另票立；v0 轮询壳留订阅接口位（输入面采集函数可换事件源）。

## 判据

- conductor 单跑一轮：正确跳过无戳/有挡态（空窗零动作同 mirror_push）；
- 有绿戳+净发窗：push→对应面 deploy→VERIFY 全链落账序正确；
- 幂等实证：同 sha+face 重复拍零重复部署；
- 冻结实证：人造升件未决→本轮全面零动作+落冻结行；
- 败注入证：mock deploy rc≠0→升件行+该面冻结+不重试；
- mirror_push.timer 退役切换零窗口（conductor 首拍即接管）。

## 范围

- `scripts/meta/conductor.sh`（编排主件：采输入/状态机/调判据/落账）；
- `deploy/o-conductor.service`+`o-conductor.timer`（沿 15min 拍）；
- `scripts/meta/tests/conductor_test.sh`（桩测：空窗/发窗/幂等/冻结/败态五组）；
- mirror_push.timer 退役注记（功能并入 conductor——弃用不删件）；
- 本票回填。

## 禁线

- 禁动判据件本体（mirror_push.sh/deploy_window_due.sh/deploy_image.sh/
  deploy_oci.sh/pre-push 全闸——一字不动）；
- 禁自愈重试/禁冻结期强发/禁无戳部署/禁推未绿 sha；
- 禁碰钱径/席位裁决语义；
- 外派施工收口移交外审席——勿 merge 勿部署勿启用 timer。

## CEO 裁决空位

- [ ] 方案整体放行（v0 轮询壳先行，v1 事件驱另票）
- [ ] 15min 拍频认可/改（check_all 20-40min 节奏下 15min 已够）
- [ ] 冻结断言行「升件未决全面不发」收紧度认可/放宽
- [ ] conductor 取代 mirror_push.timer（并入非并存）认可
- [ ] 外派施工+外审裁决径遵令确认
