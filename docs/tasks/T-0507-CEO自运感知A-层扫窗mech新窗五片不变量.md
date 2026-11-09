# T-0507 CEO 自运感知·票A——层扫窗（mech_tick 新窗+五片不变量集）

- 状态：已批（CEO 管理令放行 2026-11-09——提案-CEO自运感知三缺
  分票 ABC 序之首；业主令「做上去」锚在案）
- 级别：P2 机械轨感知面——提案档：`docs/infra/提案-CEO自运感知三缺-层扫窗诊断径动作包线.md`

## 范围

1. `scripts/meta/mech_tick.sh`——新「层扫」窗（节拍同现窗，
   零新守护件零新 unit）；
2. `scripts/meta/layer_scan.sh`（或 mech_tick 内函数族——施工
   裁形）——五片不变量机械扫：

   a. **队列一致**：dispatched 龄>30min 无拉起∧无在飞 unit；
      pulled 无存活 o-worker unit；review/approval 老化
      （龄>2h 初值）；draft 滞留>1h；
   b. **相位↔实体**：相位非终而票件无（幽灵行）；票件
      done/ 而相位非 done；相位=done 而无收口事件链；
   c. **容量水位**：连续 3 拍 spawn rc≠0；候选集非空而
      拉起=0 超 2 拍；worker 退出后无收口事件链>30min；
   d. **账本真值**（抽查非全量）：最近完成票的收口 commit
      在 master 可解、票面回填存在；
   e. **外事聚合**：origin/master 落后龄>2h；deploy 批龄
      >12h——已有探针者不重复，本窗只做聚合判断。

   每破例→`ops_event` 升件行（seat=MECH，note=「层扫
   定性<类目>:<对象>」——走 T-0506 语义闸钦定 seat；
   dedup=同对象同类目 24h 一行不刷屏）。
3. **digest 行**：每窗落 `event='层扫'` 行（seat=MECH，
   note=各面计数+异常计数）→进 ceo_queue 收件面；
   `scripts/meta/ceo_queue.sh` seat 白名单补 `'MECH'` 或
   照现有 seat 口径归一（施工按现行词表钦定）。
4. `scripts/meta/tests/`——层扫桩测新件。

## 判据

1. 回放三病灶各报出：人造 dispatched-无拉老化/pulled-无unit/
   票件幽灵行→对应类目升件行落库且 24h dedup 生效；
2. digest 行落库且 `ceo_queue.sh` 出列含层扫行；终态行
   （处置/裁决）后正确出队；
3. 全绿态层扫=零升件+digest 照落（静默≠缺席，锚在案）；
4. 全谱零新增 FAIL/SKIP。

## 禁线

- 层扫**只报不判**——机械枚举不变量，定性词限类目级
  （「dispatched-无拉」非「为什么无拉」），诊断归票 B；
- 枚举阈值写常量带出处注，禁魔术数；
- 不碰钱径/下单径；扫面只读 ops_event/票件/systemd 状态，
  零写任何生产面。
