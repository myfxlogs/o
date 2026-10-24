# T-0042 spot 市价单 tgtCcy 方言根修+APIError 拒码留痕+openfirst 默认策略 sid 漏带

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
