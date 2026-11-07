---
schema_version: 2
id: config-key-helper-escapes-domain-scan
kind: pitfall
risk: medium
status: verified
keywords: [config-write-domain, app_config, getF, 机检漏捕, 分册]
scope: [config, check_all, tooling]
summary: 新增 app_config 消费键须入写域分册（tsv+dmwrite 同集镜像）——getF()/助手包装径逃过 `.GetFloat(` 字面扫描=机检宇宙漏捕，check_all 撞红才显形
first_seen: 2026-11-07
last_seen: 2026-11-07
evidence_refs:
  - type: ticket
    ref: T-0439-返修1（决策席驳回——earn.sweep_whitelist 未登记红迹）
---

## 新 app_config 消费键=写域分册必修两件——助手包装径逃过字面扫描

- **实证**：T-0439 新增 `earn.sweep_whitelist`/`earn.residual_clear_min_usd`
  两消费键，`w.getF("earn.residual_clear_min_usd")` 走助手包装——
  check_config_write_domain 的宇宙扫描只捕 `.GetFloat(`/`.GetBool(` 等
  字面调用点，助手径漏捕；`earn.sweep_whitelist` 直读面被捕撞红收口驳回。
- **教训**：新消费任何 app_config 键→入册两件同改
  （`scripts/config-write-domain.tsv` 行 + `internal/platform/config/dmwrite.go`
  dmWritableGlobs/ownerKeyGlobs 同集镜像）——无论扫描捕不捕，分册
  语义同盖（漏捕键不入册=DM 写径 fail-closed 不可达+下票再撞）。
- **判类**：闸值/放行线族=dm；凭证/上限/对账/通知通道=owner。
- **收口**：入册后 `bash scripts/check_all.sh`——写域闸三断言
  （集同/硬界/宇宙覆盖）转绿为准。
