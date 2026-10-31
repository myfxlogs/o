# T-0284 EXTAUDIT 判决读出双源根修——锚件+原票两落点俱收

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-11-01（管理席令 `20261101-075123-MANAGE-3476`——门2-DM 落箱即点火） |
| 领取 | （session 标识 @ 时间；空=未领） |
| 优先级 | 中（功能无损=噪声级；但每件落原票判词的外部票都产一件假升级，占 MANAGE 人工核查面） |
| 涉及面 | `/opt/meta/devin-headless-audit.sh`（判决读出段+可选第 4 参）、`/opt/meta/mbus-dispatch.sh`（`extaudit_dispatch` spawn 调用点 ~:538）、`/opt/meta/eval/`（自检新增或扩展现件，施工定）；o 仓仅本票+回填，零码面 |

## 背景

EXTAUDIT 派单径（闭环断点3）：`extaudit_dispatch` 把**审计锚**（`施工提示词-外部-<T>审计.md`=审计任务书）作 `$2` 经 `spawn_audit` 传 `devin-headless-audit.sh`。wrapper 判决读出段（`:101-114`）只扫 `$TICKET`（=锚件）与 `/opt/o-wt/*/docs/tasks/` 同名副件——而外部票判词产线实证有**两个合法落点**：

- 锚件自身（12 件实证：BASESWP/DLMDFIX/GATEFIX/DOOR5/…`审计.md` 自带 `状态：审计通过`）
- **原票**（4 件实证：`义务账DB化-影子写`/`门4自动并合`/`门5自动部署`/`驳回自动返修` 原票落判词）

消费侧 REWORK 早已双源（`rework_sweep` 审计件先、原票兜底，`mbus-dispatch.sh:748-762`）；读出侧单源锚件 = **读出域窄于判词落域**，判词落原票时恒读空。

实证链：DBP1 审计真判决=通过（判词 `e291a6d` 落原票 `施工提示词-外部-义务账DB化-影子写.md:4`；`~/devin-audit-DBP1.log` 八锚全过），wrapper 读锚件空 → DM-3454「未读到判决字段——需人工核查」误报升 MANAGE（案底 `/opt/meta/docs/管理席交接态.md:361`）。外部伪票号不进判决路由（`route_report_core` 析不出 `T-NNNN` 即标 done），故功能无损、纯噪声。

## 目标

wrapper 判决读出=双源：锚件 + 原票，任一源行锚命中 `审计通过|驳回:*` 即报真值；两源皆无才报「未读到判决字段」（真异常径保留）。内部票径（`route_done`→`spawn_audit`，`:906`）读出行为零变化。

实现两径施工自裁（判据=验收标准；有第三更简等价解亦可）：

- **A（增参）**：`extaudit_dispatch` 把原票 `$f` 作第 4 参（判决补源，可空）透传 `spawn_audit`→wrapper；读出循环锚件→补源→worktree 回退。`route_done` 调用不传第 4 参=零变化。
- **B（自解析）**：wrapper 仿 `rework_dispatch:675-690` 自解析——锚件 `原票：`行取值，失效则剥『审计』名归一反扫 `EXT_DOCS` 兜底。零签名变化。

**禁径**：spawn_audit `$2` 直接改传原票——`$2` 同时进 wrapper 提示词（`本单票面:`）与 DRYRUN hook，锚件=审计任务书（复验锚所在），换之审计失纲。

## 验收标准

- [ ] DBP1 型复现绿：判词仅落**原票**（锚件无机读判决行）→ wrapper 报 MANAGE 的 `report.verdict` 件含真判词，无「未读到判决字段」。
- [ ] BASESWP 型回归绿：判词仅落锚件 → 读出不退化。
- [ ] 原票 `状态：收口待审`/散文提及 `状态：驳回:` 均不误中（值域收窄 `审计通过|驳回:*`+行首锚双式正则沿用）。
- [ ] 内部票径零变化：`route_done`→`spawn_audit` 调用面与读出源 diff/dryrun 对账证。
- [ ] 账序零动：extaudit state 机、spawn 双证、edr 账、心跳收口序 diff 面自证不变。
- [ ] 机检：`bash -n` 两脚本绿；自检（`/opt/meta/eval/rework-selftest.sh` 扩展或新件）EXTAUDIT 相关段绿——沙箱经 `DISPATCH_DRYRUN=1`+`DISPATCH_HOOKDIR` hook（判决读出前可写判词落任意文件）+`MBUS_BIN`/EXT_* 全量重定向，产线零触声明。
- [ ] meta 仓 commit 落 master；票面回填注明生效径（dispatcher 常驻进程重启与否照 T-0279 例明写边界）。

## 范围外

- 不改任何判词内容/票面 `状态` 字段；不碰在飞 T-0283、DBP2 及其票件、dlmd/mbusd 本体。
- 不动生产部署与服务重启（`mbus-dispatch.service` 重启权归管理席——票面只注明生效径）。
- 外部票判决仍不进返修/门4 路由（伪票号不析 `T-NNNN` 维持现语义）；REWORK 驳回消费面不动。
- 不追溯历史误报件（DM-3454 已人工裁=噪声，洗地零价值）。

## 参考

- 病灶码位：`/opt/meta/devin-headless-audit.sh:101-114`（读出+回报）；`/opt/meta/mbus-dispatch.sh:528-541`（anchor 传参点）、`:675-690`（原票解析同构先例）、`:748-762`（双源消费先例）。
- 实证件：`~/devin-audit-DBP1.log`；`/opt/meta/docs/施工提示词-外部-义务账DB化-影子写.md:4`；`/opt/meta/run/extaudit-DBP1.state`。
- 沙箱先例：`/opt/meta/eval/rework-selftest.sh:195-201`（env 全量重定向+桩替，产线零触）；wrapper DRYRUN hook 位 `:69-73`。
- 同在飞单协调：T-0279（dispatch.sh 面已合、余 dlmd）、T-0280（未领，面在 `spawn_dm` ~:288-320 与本单区段不叠——同文件并行须各自小步提交防冲突）。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
