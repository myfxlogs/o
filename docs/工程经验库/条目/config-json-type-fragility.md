---
schema_version: 2
id: config-json-type-fragility
kind: pitfall
risk: high
status: verified
keywords: [app_config, jsonb, 型别敏感, getF, 静默失效, fail-closed倒错向, kill_switch, ddline, 夹具形, 裸Get]
scope: [config 热启, 判据参数, 一切 jsonb 标量读径]
summary: app_config 值型别敏感——getF 只吃 JSON 字符串形，裸数字静默废触发器（安全闸被配置型别哑掉零日志）
first_seen: 2026-10-24
last_verified: 2026-11-16
skills: []
---
# config-json-type-fragility —— app_config 值型别敏感：getF 只吃 JSON 字符串，裸数字静默废触发器

## 坑
`app_config` 值是 jsonb——`getF` 实现 `json.Unmarshal(raw, &s)` 进 string 再 ParseFloat，**只认 `"20"` 字符串形，裸 `20` 数字即 unmarshal 败→missing_param→该判据整链静默跳过**。安全触发器被配置型别哑掉且零日志——fail-closed 倒向错方向（该响的触发器不响）。

## 实证
PHASE 仓 1（2026-10-24）：`exec.hedge_band_bps` 入库为裸数字 20（其余键均字符串形——单点异类），delta_drift 判据每 30s 扫一拍每拍 missing_param，30+ 分钟零触发零日志。改值为 `"20"` 字符串后 40 秒内真触发。期间 evaluate 的 5 个 missing_param 静默位全程黑盒。

## 反证条件
- 任何读 app_config 数值的径：读器必须双型兼容（json.RawMessage→先试 float64 再试 string→ParseFloat），禁单型假设
- 新增配置键：写值型别必须与读器约定一致——入库前 `jsonb_typeof` 核拍
- 静默拒类（missing_param/facts_missing/leg_anomaly）计数器必须有**可见出口**——计数增但零日志=黑洞，至少 debug 日志或周期摘要

## 机械化
- T-0046：getF 双型兼容+存量键型别审计脚本（jsonb_typeof 普查）；evaluate 各 skip 分支加结构化日志（限频）。

## 复发实证（2026-11-16，工单#20799）
回撤硬线（guard/ddline.go，-3%/30d→全平+kill_switch）自 10-27 起在产线从未判撤：`killSwitchOn` 用 `cfg.Get`+`json.Unmarshal(raw,&string)` 单型解 `risk.kill_switch`，而产线值是 fin_0008 种的 jsonb 布尔 `false`→每拍解析败→`ddScan` 在判撤行之前返回。T-0046 修了同包 `getF`，漏了同一天写的这处同胞（清扫按函数名、没按「裸 Get+单型 Unmarshal」形）。静默三周的三个放大器：
- **夹具形≠产线形**：guard 测试夹具用 `to_jsonb($2::text)` 把全部键种成字符串，迁移种的却是布尔——测试绿得没有意义；`TestDDNoAutoReset` 还把「解析败→不动作」当成了「不自动复位」的通过证据。
- **计数门无读面**：每拍 `dd_missing_param`+1（约 3 次/秒），guard 的 rejects Door 没有任何读出口——本条「可见出口」反证条件没落到这处。
- **裸修必炸**：只把读器改宽容就上线会立刻出两起事故——①判撤每拍跑 `max(equity)` 全窗扫描（实测 5.5s/次，采点 3 次/秒），守仓环被拖死；②权益三分量非原子读，划转/演习瞬间单采点可低 11%~81%（11-12、11-13 04:15:03 与 11-14 14:47:00 实证），无稳健化即误触全平。**休眠的安全件复活前，先量它醒来第一拍会做什么。**

修法（v2）：采样只进内存、5 分钟柱取中位数落行、收柱才判、峰值只读柱行（偏索引）；开关走 `GetBool` 宽容读；读不出即 `dd_line_blind` 告警。

## 反证条件（补）
- 标量键禁裸 `cfg.Get`——只许 GetBool/GetFloat/GetInt/GetString；清单/映射键裸读入白名单（check_config_write_domain 断言⑤）。
- 安全件测试必须至少一例用**迁移种子原形**（不经夹具改写）跑通触发径。
- 「失明」必须出声：判据/开关读不出=告警，不只计数。
- 安全件的判据参数要有合法域（阈∈(0,1)、窗≥1d、柱长有上下界）——越界=失明告警，不许静默停判。
