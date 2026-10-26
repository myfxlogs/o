# T-0147 终局分类补面——腿构 meta 径漏分类 + 黑名单存量回扫（T-0142 覆盖洞实证）

## 态：施工中
## 领取：zcode-t0147（ZCode 施工席）@ 2026-10-26
## 域：执行径 / venue 终局分类
## 源出：T-0144 部署验证窗实案——SHEIN-USDT-SWAP 在 fin_venue_untradeable
  黑名单在册，策略 286078 仍 live 状态每 ~2s 空转重试开仓（05:51 实拍）

## 病灶实证（2026-10-26）
- 策略 286078 SHEIN-USDT-SWAP live，开仓败 `腿构: spot 元数据: okx code=51001`
  每 ~2s 重试（28+/min），TerminalInstErr 未触发——51001 返回在
  **腿构 spot 元数据拉取段**，不走 okxexec order-submit 分类径。
- 提案 290952 fin_armed（可再武装），策略 286078 live——皆生于黑名单
  种子（04:01）之前。**黑名单只挡新流入，不回扫存量**——种子时同 inst
  live 策略/armed 提案各一漏网，决策席手动清尾（transition 行已落）。
- T-0142 钦定「armed 提案一并退役+consumer 面滤」覆盖的是**分类触发时刻**
  之后的对象；种子/手工入册时刻之前的存量无人清。

## 施工
- 51001/51003 终局分类面补齐：开仓全链一切 venue 拒语出口统一经
  TerminalInstErr 分类（腿构元数据/腿构校验/下单提交——错误冒泡点收口，
  非按函数名散贴）。
- 黑名单入册钩子：fin_venue_untradeable 插入触发存量回扫——同 inst_id
  live/proposed/paper 策略→retired（transition 行，reason=venue_untradeable
  retroactive），armed 提案→fin_retired。DB trigger 或服务侧入册函数内
  同事务/同拍均可，幂等。
- openLive 终局拦已查（T-0142 件）——本轮补**拦截前的周期重试闸**：
  live 策略开仓败若命中黑名单 inst_id→直接退役不重试（防御纵深第二层，
  防存量漏网形复发）。
- 桩锚：腿构 meta 51001→终局三件（入册+retire+提案清）；
  手工 INSERT fin_venue_untradeable→同 inst live 策略/armed 提案自动退役；
  51000 不误伤（参数族除外条款不变）。

## 验收锚
- 生产实证：黑名单词条在册 inst 的 live 策略零残留、armed 提案零残留
  （回扫钩子实跑——本轮决策席手动清尾的两行为其人工替身）。
- 开仓败 51001 日志风暴形绝迹（命中黑名单即退役非重试）。
- 51000 参数族错仍走瞬态重试不误退（回归桩）。

## 备注
- 同窗口观测 `rejects[okxws] 未注册门键 depth5_no_ctval` 每分 ~77 行洪泛——
  拒因门键注册遗漏同族病灶，另案 T-0148。

## 结果回填（zcode-t0147 @ 2026-10-26）

- **①开仓全链终局收口**：cmd/core/open.go hedgeOpener.Open——BuildLegs
  腿构径（spot 元数据拉取段实案形 286078）与 Execute 提交径**同闸统一**
  TerminalInstErr 分类（错误冒泡点收口非按函数名散贴）；任一径命中→
  untradeable 三件（入册/策略退/armed 提案清）。
- **②入册钩回扫 trigger**（migrations/0061）：AFTER INSERT ON
  fin_venue_untradeable FOR EACH ROW fn_untradeable_retro——同 inst
  live/paper/proposed 策略→retired（transition 行
  reason=venue_untradeable_retroactive:<code> 溯源）+armed 提案→
  fin_retired。幂等二重：inst_id PK（重复入册走 UPDATE 不触发 AFTER
  INSERT）+状态谓词。**ROW 级钉死**（STATEMENT 级 NEW=NULL 零匹配
  实案——开发期桩测揪出，迁移注释在案防再议）；覆盖手工入册径，服务侧
  自动径同 trigger 双保险（谓词幂等零冲突）。
- **③拦截前周期重试闸**：openLive Open 失败径查黑名单命中→
  Transition retired（reason=venue_untradeable:blacklisted）+拒计数
  blacklist_retire（预注册）——防御纵深第二层（NOT EXISTS 候选滤为第一层，
  本闸防存量漏网形复发）。51000 参数族不触发（TerminalInstErr 除外条款
  不变——TestParamErrNoTriage 回归桩）。
- **桩锚三件**（PG 实跑全绿）：TestLegBuildTerminalTriage（腿构 meta
  51001 冒泡 wrap 链→errors.As 穿透判终局+三件全落）；TestRetroactiveScanTrigger
  （手工 INSERT→存量 live 策略自动退役+retroactive transition 溯源+
  armed 提案清+重复入册 transition 恒 1 行幂等）；TestParamErrNoTriage
  （51000 零处置+零入册零回扫）。
- **机检**：build✓ vet✓ gofmt✓ staticcheck✓ check-lines✓ migration 闸✓
  （0061 登记）回归（cmd/core 全包+okxexec+lifecycle）绿。
- **移交面（部署后决策席实证）**：黑名单词条在册 inst 的 live 策略零残留/
  armed 提案零残留（回扫实跑——本轮决策席手动清尾两行的人工替身）；
  开仓败 51001 日志风暴绝迹（命中即退役非重试）。
