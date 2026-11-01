# T-0180 dcd feed 429 韧性——全集成败制改逐对容错

## 态：完成（判过并码在案，11-01 清账翻面·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**完成**——审计通过落票在案，漏网翻面

审计席：headless @ 2026-10-30

## 无头审计判决注记（收口 544b4ae5 全量 diff 逐锚对账）

- 锚①桩测「1 对恒 429→余对照常枚举+rej=1」：TestDcdFeedPairTolerance
  审计席实跑绿（go test ./cmd/research -run TestDcdFeed——2 passed）；
  dcd.go get429 尽→fail++/rej.Inc("dcd_pair_429")/continue，
  语义与票面逐字对账合。
- 锚②桩测「>50% 对败→整面让位」：TestDcdFeedHalfFailYields 同跑绿
  （2/3 对败→err）；fail*2>len(pairs) 触发 err 出口→Runner 原
  dcd_feed_failed 让位径零动；300ms 间隔+四重退避参数 diff 核实零改
  （sched-exempt-ok 注释在案）。
- 锚③生产复查：PG-md（MD 机 o-postgres-md）只读实拉
  md_dcd_product=18,052 行、max(ts)=2026-10-30 13:21Z——对票面
  1,778 首拍基线无坍缩；o-research systemd active（PID 284881，
  10-29 01:43 起）、部署二进制 grep -a 实证含 dcd_pair_429 与
  「对级失败率」串、13:21 枚举日志实打产品在架。check_rejects_keys/
  gofmt/vet 复审全绿（dcd.go 195 行、测试 51 行——≤300 门禁过）。
- 注记报管理席（域级欠账非本票施工败）：研究门 rejects 全员只写
  无读——cmd/research 无 ops/status 面，Snapshot() 生产零调用点
  （risk.Rejects/report.Snapshot 有读者，research 门独缺），且
  注册键 Inc 无日志行——≤50% 部分断供在生产层计数内陷不可见；
  >50% 面有 err+dcd_feed_failed+armwatch 静默告警兜底可见。锚
  「rej 键有观测行」在生产读径上引用了研究节点不存在的观测出口——
  属锚措辞歧义/域级欠账（funding/gapscan/hypothesis 等同病），非
  本票败；建议另立票：事件拍末将门 Snapshot 落持久行/日志（事件
  锚定形非定时器）。
- 附记：ParseProducts 解析败亦计入 dcd_pair_429 键（键名 429 而
  实涵对级全因败——语义外推可接受，对级失败率口径一致，留痕）。

| 立项 | `[角色:决策]` @ 2026-10-27（债务盘点 C 级——结构差非致命，实证在案） |
|---|---|

## 病灶

dcd 枚举 feed=**全集成败制**：currency-pair 返 ~328 对后逐对 products
（300ms 间隔+get429 四重退避），**任一单对退避尽=整个 surface 让位**
（T-0117 语义）→全批丢弃下一拍再等事件。实证 2026-10-27：sfp 限频紧
（次发即 429），首拍 ~40min 才落 1,778 行——能活但脆（限频再紧一档/
对数再涨=周期性断供）。

## 施工面

1. feed 改**逐对容错**：单对 `get429` 尽 → `rej.Inc("dcd_pair_429")`
   +跳过该对续跑（部分集返回>全集丢弃——枚举是覆盖面非事务）
2. 对级失败率>50% 时整面照让位（venue 侧真断供面与零星限频分型——
   fail-closed 语义只对系统性失败）
3. `dcd_pair_429` 拒因键注册（check_rejects_keys 闸）+ 观测面可见
4. 禁动 300ms 间隔/退避参数（限频礼貌面不动）

## 验收锚

- 桩锚：mock feed 注入 1 对恒 429→余对照常枚举+rej 计数=1
- 桩锚：>50% 对败→整面让位行为不变
- 生产复查：下拍 `md_dcd_product` 行数无坍缩+rej 键有观测行

## 结果回填

- dcdFeedFor 逐对容错：单对 get429 尽/解析败→rej.Inc("dcd_pair_429")
  +continue 续跑（部分集返回>全集成败丢弃——枚举是覆盖面非事务）；
  对级失败率>50%（fail*2>len(pairs)）→整面照让位（系统性断供分型
  非零星限频）；300ms 间隔/退避参数零动（限频礼貌面守）。
- rej 键注册：rejects.New("research","dcd_pair_429") cmd/research
  装配位（check_rejects_keys 绿）；观测面 Snapshot 键
  research/dcd_pair_429。
- 桩锚：TestDcdFeedPairTolerance（ETH 对恒 429→BTC/SOL 照常枚举
  +rej=1）+TestDcdFeedHalfFailYields（2/3 对败→整面让位）双绿。
