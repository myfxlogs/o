# T-0311 分析席接线——meta 无头径与 FRED 键路修

## 背景
T-0310 落地采集层（ffcal/COT/OI/FRED 四源+幂等+优先级）。分析席 LLM
缝选型已五模型实测毕（票面实证）：

| 模型 | 径 | 判 |
|---|---|---|
| SWE-2-Max | `devin -p` | ✅ 主径钦定（内部·$0·解释型合成优） |
| glm-5.3 | claude -p+网关 | ❌ 弃——1301 安全闸对加密所内容误拒（业主令） |
| glm-5.3-flash | 同 | ❌ 同上 |
| sensenova-6.8-flash-lite | 裸 API | ⚠️ 合格但 lite 档——留第三备不接线 |
| kimi-k3 | 裸 API | ❌ 共享额度 429 饱和 |

## 范围
### ①meta regime 定时任务（dev 盒 systemd --user timer，6h 拍）
脚本件 `scripts/meta/regime_analyst.sh`（或 go tool——施工择形）：
1. 水印读 `md_regime_fact` max((evidence->>'ctx_max_id')::bigint)（OCI PG-md——
   ssh ubuntu@149.118.58.98 psql 径，dev→OCI 已通）；
2. 拉 `md_ctx_fact` 新行（≤50 条，payload 截 800B 同码径）；
3. `devin -p --model swe --respect-workspace-trust false` 于专用 scratch
   目录（如 /tmp/o-regime——零工具工件）；prompt=票面钦定原文+
   「仅输出 JSON 数组零散文」+confidence 钦定 0~1 数值；
4. stdout 抽 JSON 数组（宽松抽取：末个 `[` 起 json.Unmarshal 试）；
   校验：stance∈{risk_on,risk_off,neutral,uncertain}、confidence 数值、
   evidence 引 id 域内——全过才写；
5. INSERT md_regime_fact（ssh psql 写 OCI PG-md——写域仅此一表+memo 表）；
6. 败=skip 本拍下拍续（fail-soft 同生产臂）；每拍心跳行写
   fin_employee_registry 若「regime 分析席」在册（不在册省心跳件——
   名册编入另议）。
7. timer 件 `deploy/o-regime-analyst.timer`+.service 入仓；journal 可审。

### ②FRED 键路修（装配径错池实锤）
现状：dcd.go `cfg.Secret(ctx,"fred_api_key")` 跑在 md 池 client——
OCI PG-md **无 app_secrets 表**（fin 域表在 SG）→键永缺席。
修：main.go 装配件增 finPool 绑定 config client（FIN_PG_DSN→SG——
SG app_secrets 已入 'fred_api_key' 密文行 2026-11-03）；该 client.Secret
注 AttachFredKey。
**主权钥已落 OCI** `/home/ubuntu/o-research/master.key`（0600，env
O_MASTER_KEY_FILE 已针）——cfg.Secret 解密面全备，只欠正确池。

### ③决策席读面节（skills/决策席/SKILL.md 补段）
`md_regime_fact` 消费口径：活在判词=`valid_until>now()`；stance 枚举域；
advisory 权——判词是裁决一件输入非指令；严禁据此直接改 app_config/
提案/仓位（改动走阈值票径）；判词缺席=neutral 处理不当拒因。

### ④生产 analyst 臂处置
`llm_absent` 守位保留=meta 径断时的第三备（零成本零害）；LLM 缝签名
留——将来若 provider 转 HTTP 型（sensenova 裸件）可原位接线。

## 验收锚
- `systemctl --user list-timers` 见 regime timer；首跑后
  `md_regime_fact` ≥1 行（stance 枚举内+evidence 带 ctx_max_id 水印）；
- o-research 重启后下拍 `md_ctx_fact` kind='macro' source='fred' 行≥1
  （4 series×观测——FRED 键已配应通）；
- 判词样例行 psql 实读贴票面；
- 零涉 fin_*/app_config 写径（grep 静态证）。

## 红线
- devin -p 会话**禁触任何仓内文件**（scratch 目录物理隔离）；
- 判词永不进钱径闸（枚举域+写域收窄双锁不动）；
- meta 任务心跳不冒充生产岗（名册语义分清——见①⑥）。

## 密钥/凭据面
- fred_api_key 密文已入 SG app_secrets（决策席 ops 毕 2026-11-03）；
- O_MASTER_KEY_FILE 已落 OCI o-research env；
- tools/secput 入库备同类键入件（stdin‖两段零明文径）。
