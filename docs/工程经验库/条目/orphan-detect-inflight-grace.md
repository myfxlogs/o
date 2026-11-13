---
schema_version: 2
id: orphan-detect-inflight-grace
kind: pitfall
risk: medium
status: verified
keywords: [唤起无果, 孤儿唤, 在飞宽限, 探针判据, 狼来了噪声, selfaware_beat, spawnlessOrphans, 死态计数混入在飞态]
scope: [孤儿/死亡/失联类探针判据, 唤起-收口配对查询, 自观卡旗面]
summary: 孤儿判定查询零龄宽限=把在飞主体当死体计数：self_aware 唤起无果旗 2h 窗内唤起行无配对收口即计，唤起后秒级即中（帧 ts 距 spawn 5s 实证），87 帧近乎常开=狼来了噪声，真孤儿反被掩。修法=龄界≥预期寿命（15min，观测会话时长32-202s）：在飞者不计，越界未配对=真孤儿照计，回归测三分支钉死。
first_seen: 2026-11-15
last_verified: 2026-11-15
skills: []
evidence_refs:
  - type: doc
    ref: cmd/core/selfaware_beat.go spawnlessOrphans（ts<now()-15min 龄界）
  - type: doc
    ref: cmd/core/selfaware_beat_test.go TestSpawnlessOrphanGrace（在飞/孤儿/配对三分支）
  - type: server_fact
    source: postgres
    query: SELECT id,ts,note FROM ops_event WHERE event='量表' AND ticket='self_aware' AND note LIKE '%唤起无果%' ORDER BY id DESC
    observed_at: 2026-11-15T18:30:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

self_aware 帧 卡=唤起无果1 常开。帧 ts=18:26:55 逮到的「孤儿」唤起
o-cognition-1794594410 在 spawn 后 5 秒即被计——其 会话收口
18:30:08 正常到、唤起果 18:30:12 正常到，根本不是孤儿。
历史取证：87 个 self_aware 帧带此旗——几乎每次会话运行期间都中，
拍节短致唤起/收口间必落一拍，旗沦为恒置噪声。

## 根因

判据 SQL `event='唤起' AND ts>now()-2h AND NOT EXISTS(配对收口/果/败)`
缺龄界——「唤起后尚无配对」同时涵盖两态：在飞（正常会话进行中）
与孤儿（spawn 后程序暴毙永无收口）。无龄宽限=在飞态混入死态计数。

## 修法/判据

龄界=预期寿命界：观测会话时长 32-202s，取 15min 宽限——
`AND e.ts<now()-interval '15 minutes'`。在飞唤秒级不再误计；
唤起越 15min 仍无配对=真孤儿照升不误。判据抽出 spawnlessOrphans
可测件，TestSpawnlessOrphanGrace 三分支钉死。

## 通用化

凡「X 出现且无后续配对=异常」类配对判据（孤儿进程/失联会话/
悬空单据/未闭环工单）皆须答两问：正常在飞期多长？龄界是否≥其上限？
缺龄界的配对判据必然恒噪——噪声旗的代价不是误报本身，是真信号
被同一旗位掩没。
