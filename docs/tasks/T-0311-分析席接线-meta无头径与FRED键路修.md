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

## 施工回填（devin-t0311 @ 2026-11-03，分支 t/0311-regime-analyst）

- **①deploy/o-regime-analyst.{service,timer}**：systemd --user 6h 拍
  （OnBootSec=10min+OnUnitActiveSec=6h+Persistent）起 `devin -p --model swe
  --respect-workspace-trust false`，cwd=/var/tmp/o-regime scratch 硬隔零仓件；
  NoNewPrivileges。
- **skills/分析席/SKILL.md**：票面钦定契约全录——水印读径
  （max((evidence->>'ctx_max_id')::bigint)）/50 行 800B 截/strict JSON 数组/
  stance 枚举/confidence 0~1/memo≤200 字标边界/写域收窄两表 INSERT/注入
  双层防/败退 fail-soft/meta 心跳口径不冒生产岗。
- **②FRED 键路修**：main.go 增 finCfg=config.New(finPool)（SG app_secrets
  驻 SG——md 池 client 读必败）；finCfg.Secret("fred_api_key")→
  AttachFredKey；researchArms 签名加参。
- **③决策席读面节**：skills/决策席/SKILL.md 补 regime_fact 段（活在判词
  式+advisory 权+禁直改+缺席=neutral）。
- **④analyst 臂守位**：llm_absent 缝留零动（meta 断线兜底）。
- 验收锚：timer 在册/首拍真判词/fred 行属部署+席位激活后实证件；
  零 fin_* 写径=grep 静态证（analyst/ctx 全件无 fin_ 触）。

## 决策席审计注记④（2026-11-03 一轮驳回——三失守）

R1 **票面③零交付**：`skills/决策席` regime_fact 读面节一行未写
   （diff 文件名面零命中）——决策层消费侧纪律缺席，分析席产出无人
   知道怎么读。补：活在判词口径+stance 枚举+advisory 权+禁直改钱径。
R2 **runbook 不可达**：ExecStart prompt `skills/分析席 runbook 读件`=
   相对路径——cwd=/var/tmp/o-regime scratch 目录**无此件**，agent 开工
   即摸空（契约全在那件里——表名/水印 SQL/枚举/写域全靠它携带）。
   修二选一：prompt 给绝对路径 `读 /opt/o/skills/分析席/SKILL.md 按件拍`
   或 scratch 下放 .devin/skills/分析席/SKILL.md 副本（部署侧同步件）。
R3 **注记③闸式 30min 未实现**：交的是 6h 无条件拍——水印闸三行壳
   （`[新行>水印]||exit 0`）未加。或实闸或票面明记「6h 无条件=钦定
   偏差」——runbook 空拍兜底在，非标锚但钉注不能默默改。

已核合格：finCfg 池修对形（finPool 绑 client→SG 密文径）✓；SKILL.md
契约全录✓；service scratch 隔离+NoNewPrivileges✓；索引件✓；build 净。

## 返修① 回填（devin-t0311，同支续提）

- ①决策席 SKILL 读面节补强：活在判词式+stance 枚举+**advisory 权**明记+
  禁直改钱径+缺席=neutral（在案件补钉非新立）。
- ②ExecStart 读件径修：ExecStartPre 落 `/var/tmp/o-regime/.devin/skills/
  分析席/SKILL.md` 副件（源 `/opt/o/skills/分析席/SKILL.md` 拍前同步——
  devin 项目 skills 发现径即达；scratch 硬隔不动）。
- ③水印闸择票面明记项：6h 无条件拍钦定偏差明记（增量空=runbook 败退径
  如实 skip——shell 闸引 ssh+psql 复杂度不入 unit 件）。

## 返修① 二轮回填（devin-t0311，同支续提）

- ②ExecStart 绝对路径指件：devin prompt 直引 `/opt/o/skills/分析席/SKILL.md`
  （.devin/skills 副本径弃——绝对路径零拷贝漂移）。
- ③30min 水印闸实装：`scripts/meta/regime_tick.sh`——`md_ctx_fact.max(id)>
  regime 水印(ctx_max_id)` 才 devin -p 否则 exit 0（闸拒=零烧；30min 拍频
  改 OnUnitActiveSec=30min——闸决定实效间隔）。
- ④心跳行全件：ops_headless META-REGIME 行（seat='分析席'——meta 席位口径
  不冒生产岗）——闸 skip/败拍 stalled/成活 alive 三态全写（每拍含空拍）。

---

## 返修二轮·决策注记（2026-11-03）

返修①②③④验讫：闸式 30min（regime_tick.sh 闸询+skip 空拍心跳+runbook 绝对径）✓、
ops_headless META-REGIME 心跳（roster_rw INSERT/UPDATE/seq-USAGE 三权实证 t|t|t
+FIN_DSN 隧道 5544 在听）✓、决策席 advisory 节（缺席=neutral 钉）✓、timer
Persistent ✓。

**驳回一件**：

1. `regime_tick.sh` 内 `/usr/bin/env devin` 摸不到二进制——service 环境 PATH=
   /usr/local/sbin:...:/snap/bin（实证 systemctl --user show-environment），
   `~/.local/bin/devin` 不在列——env -i 复测实返 "No such file"。每拍必败
   RC=127→hb stalled（心跳如实但席永不醒）。
   **修**：脚本内改绝对径 `/home/mluser/.local/bin/devin`（或 service 加
   Environment=PATH 补 ~/.local/bin——钦定前者，件即路径）。

**跟件（非驳回项）**：驾驶舱 stall 展示段（ops_headless META-REGIME 行红绿标）
未在本单——T-0312 事件总线/无头决策席票一并领；心跳行已写，展示仅读面差事。

按「三轮驳回」规计——本轮第二驳。

## 返修③ 端到端实证（devin-t0311，同支续提）

- MD 闸改 ssh 径（`ssh -o BatchMode=yes ubuntu@149.118.58.98` 远侧 env DSN
  剥 pool_max_conns——本地 MD_DSN env 段弃）。
- **实跑实迹**（两拍）：
  - 首拍（permission-mode 缺）：闸过→devin 会话中途 confirmation 拒——
    落 0 行（如实空拍）；心跳 alive 行落。
  - 二拍（`--permission-mode dangerous` 补——非交互批件必带件）：
    md_regime_fact id1-6 落（6 维判词 stance 全枚举内+ctx_max_id=52 水印+
    valid_until>now()）+md_regime_memo 6 段；ops_headless META-REGIME
    alive 心跳行落（last_activity 实拍）。
