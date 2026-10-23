---
schema_version: 2
id: ws-e2e-stub-handshake-race
kind: pitfall
risk: high
status: verified
keywords: [E2E 测试, ws fake server, login ack, 竞态, CI 红, 固定睡眠断言, 共享测试库残留]
scope: [internal/data/okxpriv, go test E2E, websocket fake stub]
summary: WS 私有流 E2E 测试桩不回 login ack=与生产握手语义不对称挂死；固定睡眠断言+共享测试库残留行双病灶——本地假绿 CI 红
first_seen: 2026-10-23
last_verified: 2026-10-23
skills: []
evidence_refs:
  - type: doc
    ref: internal/data/okxpriv/collect_test.go
  - type: server_fact
    ref: CI TestPrivE2E「事件未落事实行 n=0」（T-0011 票面）
---

# WS E2E 测试桩握手竞态（T-0011）

## 病灶

`TestPrivE2E` 本地过 CI 红「事件未落事实行 n=0」。三层根因叠加：

1. **桩握手不对称**：T-0008 修 login-ack 先行（生产 OKX 先回 login ack 才受订阅）后，
   fake WS 服务端仍不回 ack→collector 同步等 ack 超时挂死，事件永不落库。
2. **固定睡眠断言**：`time.Sleep(1500ms)` 按时序假设断言——异步链（dial/login/ack/订阅/推帧/落库）
   在 CI 慢机竞态必炸；断言须按就绪条件轮询。
3. **共享测试库残留假绿**：测试复用共享实例通用键（ccy=USDT/channel）断言，
   历史跑残留行使本地秒过——掩盖病灶 1/2，本地过≠修好。

## 修法（三件同修，单修任一件都留复发面）

- fake 服务端 login 读后回 `{"event":"login","code":"0"}` ack——桩与生产握手语义对称。
- 断言改事件驱动轮询（deadline+150ms tick），禁固定睡眠。
- 断言锚专属标记（inst_id='TESTPRIV-*'/ccy='TSTCCY'）+用前清场——防残留假绿。

## 判据

写 WS/外部服务 fake 桩时：握手序列逐帧对生产协议表（ack/订阅回应/ping-pong），
缺一帧=桩假；E2E 断言=就绪条件轮询+专属数据锚，禁 sleep 定时、禁通用键断言。
