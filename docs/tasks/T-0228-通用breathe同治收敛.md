# T-0228 通用 `@keyframes breathe` 同治——黄灯默认自息同走珠内生辉

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
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
