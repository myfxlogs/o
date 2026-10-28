# T-0228 通用 `@keyframes breathe` 同治——黄灯默认自息同走珠内生辉

| 字段 | 值 |
|---|---|
| 状态 | ✅ 完成 |
| 领取 | `zcode-glm-0228` @ 2026-10-28 |
| 立项 | `[角色:决策]` @ 2026-10-28（T-0224 施工偏离①自报转单——通用自息键同款外廓病未治） |
| 优先级 | 低 |
| 涉及面 | `web/src/react/styles/10-shell.css`（`@keyframes breathe` 一段） |

## 背景

T-0224 已把 breatheCalm/breatheQuick 两档呼相外廓病治净（filter 珠内生辉+外辉光恒定+浓度对冲）。施工自报漏网一段：**通用 `@keyframes breathe`（`10-shell.css` L182-192，`.dot.s1` 默认 1.6s 自息）仍是旧法**——50% 帧 `opacity 0.35 + 0 0 4px 1px` 外辉光胀缩，呼时大灯同款病。实际使用面：StateDot `breathe=false` 的 s1 灯（如 TrxKpiStrip 年化≤0 黄灯）走此键。

## 钦定方向（验收锚）

1. **同法迁移**：照 T-0224 breatheQuick 的处方改造——filter brightness/saturate 主驱（幅度可参照黄档或略收）、两帧 filter 显式成对、外辉光弥散半径两帧逐字同 2px/0.5px、呼相浓度÷brightness 增益对冲、opacity 通道撤除（恒 1）。
2. **节奏/灯语不动**：1.6s 节奏逐字；`.dot.s1` 挂此键的语义（黄=较快自息）不动；`.dot.still` 钳制语义不动；红死静止终裁三行不动。
3. **grep 锚**：`@keyframes breathe` 段内旧 `0 0 4px`/opacity 声明清零；两帧 `0 0 2px 0.5px` 恒定弥散+filter 成对在册。
4. **目检锚**：s1 走通用键的灯（TRX KPI 年化≤0 或任意黄灯）呼/息两相截图——呼=息同包络仅更亮。

## 范围外

- breatheCalm/breatheQuick/其他灯面（已治/已裁定）。
- `breatheAlarm` 死帧（既存档，同 T-0222 偏离④例留卫生债）。

## 参考

- `web/src/react/styles/10-shell.css` L182-192 通用 breathe 现状；L218-263 T-0224 同治后的 breatheCalm/breatheQuick 成方法。
- T-0224 票面+验收注：对冲预算算法（浓度=息相÷brightness 增益）。

## 结果回填

（`[角色:施工]` `zcode-glm-0228` @ 2026-10-28）

### 实现摘要

通用 `@keyframes breathe`（10-shell.css）照 T-0224 breatheQuick 处方同治：单 50% 帧改两帧显式成对（`0%,100%` 基帧 + 50% 呼帧）；明暗主驱=filter brightness/saturate（幅度=黄档 1.3/1.12——同 warn 色同包络，灯语只差节奏）；opacity 通道撤除（恒 1）；外辉光弥散半径两帧逐字恒定 `0 0 2px 0.5px`，呼相浓度=息相÷brightness 增益对冲（47/1.3≈36→35，与黄档同款取整）；T-0222 3D 恒驻件（描环+底部内影+接触微影）逐帧同携。节奏 1.6s 逐字、`.dot.s1` 挂键语义、`.dot.still` 钳制、红死静止三行、breatheCalm/breatheQuick/breatheAlarm 全部未动。通用键使用面全查：仅 `.dot.s1`（1.6s 自息）与 `.dot.breathe` 裸类（2.2s，实际被状态变体覆盖）——两用面同获同治。

### 验收锚对账

1. **同法迁移**: PASS——两帧 filter 显式成对（基帧 `brightness(1) saturate(1)`）、两帧 `0 0 2px 0.5px` 逐字恒定、呼相浓度 47/1.3→35 对冲、opacity 零声明（证据:`sed -n '/^@keyframes breathe {/,/^}/p'` 段内 `0 0 2px 0.5px`×2、`filter:`×2）。
2. **节奏/灯语不动**: PASS——`.dot.s1` 仍 `breathe 1.6s ease-in-out infinite`（L167 原行未触）；`.dot.still`/红死三行 diff 零触碰（证据:`git diff` 仅 keyframes 一段+注释）。
3. **grep 锚**: PASS——段内旧 `0 0 4px`/`opacity` 零命中（grep exit=1）；`0 0 2px 0.5px`×2+filter 成对在册。
4. **目检锚**: PASS——真实 SPA 渲染（worktree vite dev 5288+Connect JSON stub 50051 供数，T-0222/0224 同法；年化 7d/30d 负值→TRX KPI 三黄灯走通用键），呼相以 -半周期 delay+paused 冻结 50% 峰帧。六件入 `docs/design/T-0228-*.png`（真页呼/息×亮/暗 4+两相并列面板 1x/5x×亮/暗 2）。判读三验：①呼=息同珠径同辉光包络（面板 5x 灰晕两相同径）②呼相仅珠体更亮（highlight 更白）③真页无「大灯/大团光」读感，呼相灯读「小而亮的珠+极窄贴边辉光」。

### 机检

check_all 全链绿（build✓ vet✓ import✓ gofmt✓ env✓ rejects✓ migrations✓ test 全谱✓）——绿戳随 commit。

### 偏离说明

1. **工作树基点=本地 master `890f6f6`，非票面钦定 `origin/master`**：领单时 master 与 origin/master 分叉（merge-base=784db1e；本地领先 4 commit 含 T-0224 处方与本票面；origin 领先 5 commit 纯决策面文档）——票面参考处方（L218-263）与票面本身只在本地 master，基 origin 建树则参考不存在。**决策席随后重铸 master 调和分叉（PR #17+T-0224 链重放，内容同 SHA 变）**；推支被 T-0219 闸红（同径票面内容异），按闸指引**合流 origin/master 归位**（本 merge commit）——基点偏离消解，票面基点复归钦定 `origin/master`。
2. **息相帧补恒驻贴珠微光**（原息相=20-cards 硬化段三件栈、无外辉光）：票面锚 3 钦定「两帧 `0 0 2px 0.5px` 恒定弥散+filter 成对在册」——息相从无辉光到 47% 贴珠微光属钦定内语义，非超单。


---

## 验收注（`[角色:决策]` @ 2026-10-28）

**裁：验收通过，票翻完成。** 四锚全过（同法迁移/铁律零触/grep/目检六件——负年化实态命中 s1 通用键灯，呼=息同包络仅更亮）。偏离①基点偏离已按闸合流归位=认可；②息相恒驻微光属钦定锚内语义。至此灯语三段（calm/quick/通用自息）全走珠内生辉径，「呼时大灯」病诉三键全愈。
