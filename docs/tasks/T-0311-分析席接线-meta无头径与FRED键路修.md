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

## 决策注记（业主令——席位化简化，取代①原案）

**业主判**：分析席=meta 席位非生产码——一个 tmux 常驻 devin session 领
runbook 即可（同决策席/研究员席位形制）；不牵其他项目、不造 subprocess
包装层。**范围①改判**：
- ~~systemd timer+脚本+devin -p 包装~~ → `skills/分析席/SKILL.md`
  runbook 件：职责（读 ctx→产 regime 判词）、输入径（ssh OCI psql 读
  md_ctx_fact 水印增量）、产出契约（strict JSON 数组；stance 枚举；
  confidence 0~1 数值；evidence 实引 id；memo≤200 字标边界——决策层=AI
  机器可解析优先）、写域收窄（仅 md_regime_fact/md_regime_memo 两表）、
  唤醒口径（席位调度/tmux ping 沿用不新建）、心跳语义（meta 席位口径
  不冒充生产岗 KPI）。
- ②FRED 键路修=唯一生产码件保留；③决策席读面节保留；④生产 analyst
  臂守位=永久后备（meta 席断线时的兜底缝已留）。
- 验收锚改：runbook 入库+席位激活后 24h 内 md_regime_fact ≥1 实产行+
  stance 枚举内+ctx_max_id 水印正确。
- 诚标：agent 席写域=信任+审计制（SKILL 红线+事后审计），非生产码闸；
  活性判据按 meta 席位口径不混生产 KPI。

## 决策注记②（业主令再修——headless 单发，非常驻）

分析席终形钦定：**headless 单发批件非 tmux 常驻**——regime 分析无状态
（水印在库非会话），常驻会话纯吃状态不解决唤醒。终形三件套：
1. `deploy/o-regime-analyst.{service,timer}`——systemd --user timer 6h 拍
   起 `devin -p --model swe --respect-workspace-trust false`，cwd=专用
   scratch（/var/tmp/o-regime 或同——零仓件接触）；
2. `skills/分析席/SKILL.md` runbook=prompt 本体（职责/输入径/产出契约/
   写域收窄/败退口径——agent 按件自主读算写）；
3. 零包装脚本零 JSON 抽取层——校验写库 agent 自行（写域收窄在 runbook
   红线+事后审计钉）。
**决策席无头化另票**：ops_headless 域+relay 基建在册——唤醒模型（收口
事件触发+周期兜底）+红线三件升业主不动=独立票（T-0312 向）不入本票。

## 决策注记③（业主问「数据到能否激活」——闸式轮询钦定）

meta-gateway 考古（/opt/meta 独立仓）：seat_run 持久账+60s ticker＋事件
kick 双源调和器——外源拍=`ticket_event` 行（postmsg.go：「systemd timer
心跳等外源拍，网内零 cron」）。**席位活性押触发源非会话驻留**。

终形修订（timer 6h→闸式 30min）：
1. `deploy/o-regime-analyst.{service,timer}`——OnUnitActiveSec=30min，
   ExecStart=壳件先查水印：`md_ctx_fact` max(id)>regime 水印→
   `devin -p`；无新行秒退零 session 烧（数据驱动唤醒=采集到才激活）。
2. 事件推径否决——生产臂写 meta 事件域=跨域污染；PG NOTIFY 听客径
   否决——多养 daemon。
3. T-0312 决策席无头=挂 meta-gateway ticket_event+seat_run 机制
   （report.sh 事件环收口即拍→调和醒裁决）——基建钦定不另造轮。
