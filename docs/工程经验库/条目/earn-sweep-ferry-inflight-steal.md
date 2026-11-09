---
schema_version: 2
id: earn-sweep-ferry-inflight-steal
kind: pitfall
risk: high
status: verified
keywords: [earn, sweepIdle, funding 户, 摆渡, AssetTransfer, dcd, 竞态, inFlight, 51736]
scope: [internal/earn, internal/trading/dcdexec]
summary: earn funding 段扫资缺 inFlight 让位闸——开仓摆渡款落 funding 即被抢去 Earn=ping-pong 死环；在途闸须罩全域非只罩交易户段。
first_seen: 2026-11-09
---

# earn funding 段扫资缺在途闸=抢摆渡款死环

## 症状
dcd_put #287249 live 后开仓连拒 `okx 51736 Insufficient balance`：
交易户持 76.92 USDT 名义 50 却报不足——sfp/dcd=finance 产品
**扣款源=funding 户**（Earn 同律），funding 余=0。
加摆渡修后浮出第二病灶：渡款落 funding 到 DcdTrade 之间，
`sweepIdle` funding 段直读余量>阈（25）即购 Earn——把刚渡的钱
抢走→trade 仍拒→下拍再渡→再抢=**ping-pong 死环**。

## 根因
`sweepIdle` 的「执行在途（fin_execution state='open'）让位」闸
只罩交易户段，funding 段（T-0149② idleFund）裸奔在闸前执行——
原序认为 funding 余=闲置天然，没料到「在途单预备金」语义。

## 修法（两件同 commit）
1. `dcdexec.Open`：Gate 后先落 exec 行（state='open'=在途证据锚）
   →按 notionalCcy==USDT 查 funding 缺额→AssetTransfer 幂等摆渡
   （终拒禁补/传输败→transfer-state 三态仲裁——earn T-0059 同律）
   →才 DcdTrade。非 USDT 锁仓币直通交 venue 仲裁（异币尺度另票）。
2. `earn.sweepIdle`：inFlight 让位闸上提到 funding 段之前罩全域。

## 判据
凡「钱先过 funding 再消费」的钱径（finance 产品族：sfp/dcd、Earn），
都要问一句 sweepIdle 会不会抢；凡新增「在途」判据，检查是否罩住
所有动钱段而非只罩新增的那段。
