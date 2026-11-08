# 提案：check_all 甩 OCI——戳跑离岸解放本盒（真扩容先决件）

- 状态：**已批**（业主授权COS直裁2026-11-08——CTO串行绕裁令；施工排期另定）（业主令 2026-11-08 问「为什么不甩给 OCI」——
  立项。并行硬顶裁案的**真扩容先决条件**；RSI 方向锚——自改进
  环的吞吐瓶颈搬出本盒）

## 一、病灶

实证 2026-11-08：3 个 check_all 并行→load~15，各跑各慢总时长
无省；merge 串行不变，多施工位=等位挪窝不是提速。**约束是算力
不是坑位**。OCI 免费层机位现成（eventbus 已在跑）——盒上只留
merge+deploy+交互，重载戳跑离岸。

## 二、设计面

1. **输送径**：施工分支 sha 本地未推——`git push oci-ci <branch>`
   到 OCI 侧 scratch remote（SSH 通道现成——隧道已在跑），
   OCI 侧 runner 收 push→跑 check_all scoped→回报结果。
2. **戳语义**：戳=「该 sha 这些测试绿过」——OCI 绿回报携结果
   摘要+包清单+RUN/FAIL/SKIP 数回本盒，本盒核签后落
   `check-green-<sha>`（信任域=自家盒，可接受；结果须可核
   ——附测试输出哈希防伪造面）。
3. **本盒卸载面**：戳跑/桩测跑/施工自测跑全可离岸——本盒保
   merge+部署+交互+生产进程，负载墙消失。

## 三、诚实风险（ARM↔x86 分歧）

OCI 免费层重机位=Ampere ARM64；本盒 x86。**ARM 绿≠100% 证
x86 绿**——时序敏感测试（events 族刚踩过的竞态类）可能架构
差异 flake；cgo/指令集依赖面可能行为分歧。两档裁案：

| 档 | 制 | 评 |
|---|---|---|
| A 筛+终盖 | OCI 跑快筛消 flake 重试环，本盒终盖一趟 | 保守——本盒负载砍大半（失败迭代全离岸），戳信度无损 |
| B ARM 戳直采 | ARM 绿=戳有效+每周 x86 抽检校准 | 激进——本盒零戳跑；分歧漏网风险用抽检兜 |

**建议档 A 起步**——先离岸失败迭代（负载大头本来就是重试），
校准 ARM↔x86 分歧率攒数据后再议 B。

## 四、判据（草案）

- 人造分支 push oci-ci→OCI 侧绿回报携摘要回本盒→戳落；
- 本盒戳跑次数对比周降 ≥50%（档 A 量度）；
- ARM 绿 x86 红的分歧案一旦发生→升件+档 A/B 复审；
- OCI runner 缺席→fail-open 回本盒现径不炸。

## 五、前置（机位已勘 2026-11-08 实测 @149.118.58.98）

- 机位：aarch64 4 OCPU / 23GB 内存（19 可用）/ 25GB 余盘——
  免费层重机位，跑 check_all 够格；
- 已在：docker、git、o-postgres-md 容器（MD 域先例在跑）；
- 缺：go toolchain、o-postgres-test 族容器（fin 测试库）、
  仓库 clone+scratch remote、SSH push 通道（隧道径现成）；
- **业主令：本件加急**——四件引进中唯一「建完立刻省墙钟」
  的，优先级建议提 CTO 评审序首（提案呈批 2026-11-08 在队）。

## 六、施工结果（活体三件·2026-11-09 收口回填）

档 A 四件已并（merge `4ef5fb941`）；残活三件实录如下。全程命令
输出账留存于会话日志+OCI `~/o-ci/reports/`。

### 件①活装——`oci_ci_provision.sh` 真机跑通

- SSH 免密键直通（`ubuntu@149.118.58.98`，BatchMode 无交互）；
  机检 aarch64 / 23G 余盘 / docker+git+python3+curl 齐。
- 装面全落 `~/o-ci`（go1.27.3-arm64 / buf1.74.0 / protoc 插件三件
  版本对 / o-postgres-test:5434 + o-postgres-test-c:5436 健康 /
  scratch.git 裸仓+post-receive+work clone+o-ci-runner@.service /
  web/dist 快照）——**幂等复跑全跳过**（前会话档A开发已装大半，
  本跑证「在=补齐配置不重装」语义），冒烟自检六项全绿。
- 生产面零触：`~/o`、`~/o-research`、o-postgres-md(5435) 未动；
  5435 号位冲突闸在案（o-postgres-test-b 检出即 die）。

### 件②端到端活体——两面取证

**面甲 scripts-only（判据④ fail-open 活证）**：
`oci-ci-probe` 枝 commit `2aa4fd702`（仅 `scripts/oci_ci_probe_marker.sh`）
→ push scratch → runner 实跑分类 `surface-full` → 报告 `exit=75
note=surface-full` → accept rc=75 → push.sh fail-open 本盒径。
（`OCI_CI_LOCAL_FALLBACK` 桩证调度面；真径=check_all.sh 现行不变。）
**取证点**：`oci_ci_classify_surface` 与 check_all 测谱分级同判
——scripts-only=full 是**钉死语义**（桩测 A7b 断言+check_all
CODE_SEEN=0→full 同径），非缺陷；用户预期「scripts-only 跑
scoped」与钉死判据不符，按令「不符处先取证」——证据=桩测 A7b
+check_all 分类代码同判，故面乙补 .go 码种子取真绿链。

**面乙 scoped 绿链（判据①活证）**：
commit `56f01d10c`（面甲+`internal/platform/netmeter/zz_oci_ci_probe.go`
一码种子→面=scoped）→ `oci_ci_push.sh` 推 scratch（master 基线
同推）→ post-receive 起 `o-ci-runner@56f01d10…` → runner 串行锁→
fetch→checkout→buf gen→go mod download→`CHECK_ALL_DEFER_BRIDGE=1
check_all.sh` scoped 定向谱（testdb-lease 租 o-postgres-test:5434
独占）→ 报告+整跑日志回本盒 → `oci_ci_accept.sh` 核签（sha/exit/
mode∈{scoped,merge-scoped}/arch/pkgs/哈希复算/log 字节数六断言）→
**netgates=local:bridge-audit 延后闸本盒同 sha 净树补跑
check_bridge_registry.sh 绿** → 落 `check-green-56f01d10…`+
`.mode`+`oci-report-` 侧档（arch=aarch64 记档，分歧探针输入）。

**活体实测参数**：runner 单跑 **5分30秒**（18:20:06→18:25:36 UTC，
ARM 4 OCPU scoped 谱 30 包含恒测册 risk/ledger/trading/guard/
lifecycle——本盒同谱量级 20min+，离岸墙钟省 ~75%）；报告
`run=458 fail=0 skip=327`；本盒 accept 核签+延后闸补跑合计
**~105s**（两次 ssh 拉取+sha256 复算+check_bridge_registry 四列
断言即过）。

### 件③半成品窗复核（OCI 缺席期本盒径零差）

- **接入面**：`oci_ci_push.sh` 仅施工/席位显式调用——本盒
  `check_all.sh`/`git_push.sh`/pre-push 链路零改动；OCI 全缺席
  （remote 未配/push 败/SSH 缺席/报告超时/accept 不可核）→
  `local_check` 回 `check_all.sh` 现径，判据④桩测 C1–C3+D2–D4
  全绿在案。
- **runner 红/超时/畸形回本盒**：report `exit≠0,75`→accept 1
  （不落戳交施工修）或 2（不可核→fail-open）；超时→push 轮询
  上界→fail-open；畸形/哈希不符→rc=2→fail-open——桩测 A3/A4/A5/
  A10/A11+C3 活证+面甲 live 互证。
- **档 A 边界**：surface-full/full/docs-only 不烧 OCI（面甲 live
  实证）；bridge-audit 延后径本身不落戳（check_all 写 .deferred
  不产 check-green——不绕闸产戳面）；ARM 绿≠x86 证——分歧探针
  `oci_ci_divergence.sh` 接 check_all fail()，侧档 arch 比对，
  桩测 B1/B2+E1 在案。
- **本盒终盖不变**：master 不走 OCI（档 A 终盖归本盒——面甲 push
  明示+桩测 C1）；同 sha 本盒重跑 check_all 盖戳=旧径零差。

### 判据四条对账

| # | 判据 | 结论 |
|---|---|---|
| ① | 人造分支 push→OCI 绿回报携摘要回本盒→戳落 | **过**——面乙 scoped 全链活证（戳/侧档/延后闸补跑留痕） |
| ② | 本盒戳跑次数周降≥50%（档A量度） | **挂起量度**——机制就绪，周对比需攒运行周数据（本件不裁） |
| ③ | ARM 绿 x86 红分歧案发生→升件+档复审 | **探针在案零触发**——面乙 OCI 绿、本盒未复跑（档A 设计）；分歧探针接 fail() 桩测 B1 实证 |
| ④ | OCI runner 缺席→fail-open 本盒现径不炸 | **过**——面甲 surface-full 活证+桩测 C/D 全谱 |

### 遗留/移交提示

- 面乙戳 `check-green-56f01d10…` 落公共 gitdir——系 throwaway
  探针 sha（oci-ci-probe 枝不入 origin），戳仅作活体证据不消费；
  侧档 `oci-report-56f01d10…` 同域留档供分歧探针口径核查。
- `oci-ci-probe` 枝（面甲+面乙两 commit）仅推 oci-ci scratch，
  **未推 origin**——探针面不入正式枝图。
- 提案判据②周降量度与档 B 升级评估归后续运行数据，非本件范围。
