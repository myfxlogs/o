---
schema_version: 2
id: port-wiring-drift
kind: pitfall
risk: high
status: canon
keywords: [移植, 装配层, goroutine, 同步调用, 基址常量, 依赖注入, cmc]
scope: [engineering, port-migration]
summary: 跨仓移植装配层时逐参对拍源调用位——go 包装/常量基址/注入步三件套最易漂移
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: []
evidence_refs:
  - commit 9dc17e5/f634aaf/dd086c7（T-0108 cmcArm 三修）
  - /opt/nox/cmd/core/wire_cmc.go:35-55（源调用位对拍锚）
---

## 病灶（T-0108 CMC 采集臂实测——一次移植三处漂移，编译全过）

移植 `nox collect/cmc` 到 O 时，`internal/data/cmc` 包文件原样搬入且编译绿，
但**装配层调用语义**漂了三处，部署后凭据到位仍零采集：

1. **`safego.GoLoop` → 裸同步调用**：`auth.StartRefresh` 是 ticker 循环函数，
   nox 用 `safego.GoLoop` 异步跑；移植写成 `auth.StartRefresh(ctx)` 同步
   调用→ goroutine 卡 select，装配流永到不了 Register/上岗日志，
   **零日志零报错**（SIGQUIT goroutine dump 才现形）。
2. **基址常量→裸字符串**：包内 `DefaultPlatformBase`/`DefaultRestBase`
   原样随迁未用，调用位写成 `api-cmc.markets.com`（**NXDOMAIN 不存在域**）。
   正确值=platform.cmcmarkets.com(auth)+connectapi.cmcmarkets.com(REST)。
3. **品种名漂移**：`X-AFQUL`（CMC 内部 TRX CFD 码）写成 `TRXUSD`。
4. **注入步整丢**：`SetCreateTplHex(secRes.Get("NOX_CMC_CREATE_TPL"))` 整行
   未移植→ tplHex 空 → `createTpl header mismatch` 永 fail。

## 判据

- **移植≠拷文件**：包代码零改搬入后，装配层调用位必须与源**逐参对拍**
  （函数调用语义[同步/异步]、字面量 vs 包常量、注册/注入/钩子调用全序）。
- **静默卡死征兆**：装配函数该出「上岗/让位」日志却一行没有——大概率
  卡在首个调用前的同步阻塞件，SIGQUIT dump 是最快取证径。
- ** venue 基址永用包常量**：手敲 URL 必然漂——`Default*Base` 存在即钦定。
- **凭据搬迁清单**：密件不只有账号口令——协议模板/证书/token 模板等
  「非典型凭据」同样列搬迁表（NOX_CMC_CREATE_TPL 是抓包 protobuf 帧）。
