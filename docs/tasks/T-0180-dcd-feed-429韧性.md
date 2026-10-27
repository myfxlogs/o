# T-0180 dcd feed 429 韧性——全集成败制改逐对容错

## 态：✅ 验收毕投产（merge 544b4ae——逐对容错+>50%断供分型落地、桩2测绿、o-research换件在线）

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
