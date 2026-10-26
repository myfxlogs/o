# D-0025 黑名单回扫覆面补 probe/degraded 态（T-0147 验收衍生）

## 态：待施工
## 原编号：T-0151（2026-10-26 前后台分系改编——后台 D 系/前端 T 系）
## 域：执行径 / venue 终局分类
## 源出：T-0147 验收生产窗实案——0061 trigger 谓词只覆
  live/paper/proposed 三态，策略 286181（SHEIN-USDT-SWAP）probe 态
  漏扫滞留（决策席手动退役清尾，transition 行已落）。

## 病灶
`fn_untradeable_retro()` + `hedgeOpener.untradeable()` 的 live 守卫
均不覆 probe/degraded 两态——黑名单 inst 上的 probe 策略滞留：
probe→live 晋升后首开仓才被 opener 黑名单闸退役，窗口期策略
在册活着=状态机谎言（venue 已证开不了）。degraded 同形。

## 施工
- 0062 迁移：trigger 谓词集 IN ('live','paper','proposed','probe','degraded')
  ——retired 外全覆（闭集拒自挡已退役行）。
- `hedgeOpener.untradeable()` live 守卫同扩（显式迁移先于入册序位
  不动——仅谓词集对齐）。
- probe→live 晋升径顺带核：promotion 闸若读黑名单更优（未读=靠
  opener 兜底，可接受但记档）。
- 桩锚：probe/degraded 态策略+黑名单入册→trigger 回扫退役+
  transition 行落。

## 验收锚
- 五态全覆盖桩绿；存量黑名单 inst 上任意非 retired 策略=零。
- 生产实证：probe 态幽灵（如再出现）入册即退。
