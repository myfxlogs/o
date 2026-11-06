# T-0386 ops_event 防篡改账——hash 链化与禁 UD

- 状态：已批（2026-11-06 业主令——grounder 防篡改账移植：
  裁决账不可追改=自治可信根）
- 领取：常径待派（DM 常径机派）
- 优先级：P1——账完整性硬固（自我迭代加固族①）
- 收口移交归：决策席

## 病灶

ops_event 是全公司裁决/派单/参数调整唯一账——写进程同
角色可 UPDATE/DELETE，账可被追改。自治面越大，账不可篡改
越是可信根（grounder 判例：append-only 脊+SHA 链）。

## 范围

1. 迁移（fin 链水位+1）：ops_event 增 `prev_hash`/`hash`
   两列（text）——从现 max(id)+1 起链（老行不回填，
   创世行 prev_hash=NULL 锚注明）；
2. 链写：INSERT 侧 app 层算 hash=sha256(id|ts|seat|event|
   ticket|note|prev_hash)——写面集中在 report/直插两径
   全改走同一链函数（或 DB trigger 兜,择一注释引证）；
3. 禁 UD：迁移内 REVOKE UPDATE/DELETE ON ops_event FROM
   应用角色（写角色册查——grant 面同迁移修,注释明）；
4. 验链拍：慢拍（挂 hunger/矿工同槽正径）全链重算→
   断链/伪戳=ops_event `账链断`行+升件（自报可自证）；
5. 验链脚本 `scripts/check_ops_chain.sh` 入 check_all
   （或入 bridge-audit 段,择一）。

## 验收锚

① 桩测：插3行链连续+篡改中间行→验链检出+自报行；
② REVOKE 断言：应用角色 UPDATE 被拒；
③ 创世锚+链起点断言；④ check_all 绿戳对口；
⑤ 镜像面件并入发窗。

## 红线

- 不改既有行内容（链从新行起——历史不可变已是事实）；
- 不破坏写面（INSERT 照旧所有席可写）；
- 验链拍纯读零写；
- 不动真单径。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
