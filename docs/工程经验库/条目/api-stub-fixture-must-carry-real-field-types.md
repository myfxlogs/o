---
schema_version: 2
id: api-stub-fixture-must-carry-real-field-types
kind: pitfall
risk: high
status: verified
keywords: [adapter, unmarshal, 字段类型, stub, fixture, 协议面取证, venue_down, scout, binance, interestRate]
scope: [internal/research/scout, internal/data, 外部API适配层]
summary: 外部 API 适配器测试 stub 用理想形 JSON 未带真实字段类型——interestRate 字符串撞 int 字段整次 unmarshal 恒败，venue_down 静默零行两拍；fixture 须用实测截获载荷
first_seen: 2026-11-03
last_verified: 2026-11-03
skills: [协议面取证]
evidence_refs:
  - type: server_fact
    source: oci-box-premiumIndex
    query: python3 json.load 实测 type(interestRate)=str "0.00000000"
    observed_at: 2026-11-03T05:51:00Z
    server_revision: "fapi.binance.com premiumIndex 945 行实载"
  - type: commit
    ref: e928a8a0
---

## 事实/症状

scout `collectBinance` 两拍零行无错误日志（venue_down 计数门为进程内
atomic、无生产面 Snapshot——静默杀）。盒上 curl/Go 探针取同一端点均
200/209KB 正常。对照真响应逐字段：`interestRate` 返回**字符串**
`"0.00000000"`，而 adapter struct 声明 `IntervalH int`——`json.Unmarshal`
遇 string→int 类型冲突破整次 unmarshal，adapter 恒 err。

## 根因或策略解释

测试 stub 只喂了理想形字段（漏 interestRate 或写成数值），单测过=假绿。
**adapter 字段类型对文档/想象=必踩坑**——真实载荷是唯一权威（协议面取证
纪律同款）。次生：`fundingIntervalHours` 在 premiumIndex 响应**不存在**
（该字段在 `fapi/v1/fundingInfo`）——字段名对了端点错照样零值。

## 自检

新 adapter 收工时：真端点截获载荷存 fixture 跑一次（非手写 JSON）；逐字段
type 核对（Go unmarshal 对 string↔number 不宽容）。生产排查序：DB 零行→
计数门→盒上探针同一 fetch 复现→载荷字段类型对照 struct tag。

## 处置

T-0302 急诊：删 `IntervalH int` 死字段（未被读），真载荷解析 945/896 过，
部署一拍 binance-usdm 896 行落库。`interval_h` 归真属后续票
（`fapi/v1/fundingInfo` 已验有真值）。另立观察面缺口：scout rejects Door
无 Snapshot 消费面，venue_down 运行期不可查。
