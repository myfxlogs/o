# T-0042 spot 市价单 tgtCcy 方言根修+APIError 拒码留痕+openfirst 默认策略 sid 漏带

> 状态：作废（superseded——11-01 清账定性·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**作废**——superseded：tgtCcy方言/APIError分码族/sid漏带三修并master+返修锚绿（8c11e027）——真弹面由真钱链实弹覆盖

## 病灶（exec_id=3 实弹实证 2026-10-24）
- leg1 SWAP sell 0.03 **filled**（posSide 修生效）→leg2 BTC-USDT spot buy sz=0.00030914 **failed**→leg1 rolled_back 成真平（平仓语义修生效）。零残仓零损。
- 真凶甲：OKX **spot 市价买单 sz 默认=计价币（quote ccy）口径**——0.00030914 本意 BTC 数量，被按 USDT 金额解释→低于单笔最小额拒单。修=OrderParams 加 `TgtCcy`，spot 市价买单腿传 `"base_ccy"`（市价卖默认即 base 口径可不带；perp/换腿不受影响）。**禁猜**——改前读 skills/okx-exec+协议面取证卡核证（官方文档/实测响应），修后该方言事实入能力账（D-641 在用中修订钩）。
- 真凶乙：leg2 venue_err 空——拒码留痕只兜 body 内 sCode≠0，`do()` 的 envelope `*APIError` 走 err 径丢码。修=flow Place 败时 `errors.As` 提 APIError.Code/Msg 进 leg 行 venue_err_*（与 body 拒同轨）。
- 顺带：openfirst 默认策略径 `-strategy` 未传时 StrategyRef 恒 0→intent_incomplete（decision=8 实证）。修=SELECT 回 id 赋 sid。

## 施工面
- `internal/trading/okxexec/order.go` OrderParams+TgtCcy 序列化
- `internal/trading/execution/okx.go` buildOrderParams：tdMode==cash && side==buy && ordType==market → TgtCcy="base_ccy"
- `internal/trading/execution/flow.go` Place err 径 APIError→venue_err 留痕
- `tools/openfirst/main.go` 默认策略径回赋 sid
- 测试锚：tgtCcy 断言（spot buy=cash→base_ccy；perp 不带）+APIError 径留痕断言+sid 回赋断言
- 能力账 skills/okx-exec 补 tgtCcy 方言条目（同票）

## 验收锚
- [ ] 测试锚三枚绿
- [ ] 真弹第四发：leg1+leg2 双 filled，或拒因真实入 venue_err（再拒=新方言坑如实回填）
- [ ] skills/okx-exec tgtCcy 条目在


## 结果（施工回填 2026-10-24）

### 码面三修（commit 8f0…两笔于支 t/0042-tgtccy-apireerr）
1. **tgtCcy 方言**：`okxexec.OrderParams` 加 `TgtCcy`（`tgtCcy,omitempty` 序列化）；`buildOrderParams`：tdMode==cash && side==buy && ordType==market → `TgtCcy="base_ccy"`（spot 卖默认即 base 口径免带；SWAP 不带）。核证链：票面 exec_id=3 实弹实证（0.00030914 BTC 意图按 USDT 金额拒）+ 官方 docs-v5 place-order tgtCcy 参数义（base_ccy/quote_ccy，spot 市价买缺省 quote_ccy）。
2. **APIError 拒码留痕**：`flow.go placeAndResolve` Place err 径 `errors.As(*okxexec.APIError)` → leg 行 `state=failed`+`venue_err_code/msg`（envelope 拒=确定单未建，不走 Resolve 歧义径；与 body sCode≠0 同轨）。
3. **sid 回赋**：openfirst 策略行解析抽 `resolveStrategy(ctx,pool,sid)`——默认径 `SELECT id,…` 回赋 sid（防 StrategyRef=0→intent_incomplete）。

### 测试锚三枚（绿）
- `TestBuildOrderParamsTgtCcy`：spot buy→base_ccy；spot sell/perp buy/sell 不带。
- `TestPlaceAPIErrorLeavesTrace`：fakeVenue 返 `*APIError{51016}` → fin_order_leg failed+venue_err_code=51016+真拒因。
- `TestResolveStrategyDefaultScansID`：默认径回赋真实 strategy id。

### 机检
build/vet/gofmt/test 全量 / -race（trading+openfirst）/ import-direction（59/60 边）/ migration-contract / lines / env-reads——全绿。无新 migration。

### 真弹第四发——**前置态未备，未发**
核验（dev 库 phase-postgres）：fin_strategy 最新 669=BTC-USDT-SWAP **paper**（无 probe/live 行）；`invest.lifecycle.live_switch=false`。按 T-0025 段二/三纪律（禁自行翻 live_switch、未晋升禁动），施工席不可自造前置态。**待决策席：策略晋升 probe + live_switch=true 后，跑 `openfirst`（默认径即验 sid 回赋）一发即验**——预期 leg1 SWAP sell+leg2 spot buy（携 tgtCcy=base_ccy）双 filled；若再拒，venue_err 留痕已在（两径兜齐）。

### 能力账
skills/okx-exec 「下单方言」补 tgtCcy 方言坑条目（同票同修，D-641 钩）。

## 返修（决策令 2026-10-24）：APIError 分码族定态
- `placeAndResolve` envelope 拒分流：`51` 前缀（参数/业务拒族）=确定单未建→absent+拒码留痕、**禁查单**；非 `51`（50004 超时/50011 限速/50013 系统忙等）=歧态势→**留痕透传（vc/vm）后仍走 Resolve 定真态**，定态结果行（filled/ack/ambiguous）均带拒码。
- 返修锚两枚绿：`TestPlaceAPIError51FamilyNoResolve`（51016→failed+留痕+resolveCalls=0）、`TestPlaceAPIErrorNon51FamilyResolves`（50004→Resolve 被调→filled+venue_err_code=50004 透传）。
- 机检全量复跑绿（build/vet/gofmt/test/race/import/migration/lines/env）。
