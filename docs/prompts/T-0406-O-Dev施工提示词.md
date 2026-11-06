# T-0406 O-Dev 施工提示词——UI 两修

领单唯一权威范围：
/opt/o/docs/tasks/T-0406-驾驶舱UI两修-弹窗遮挡与UTC时标.md

读 web/src/react/components/QuoteBoard.tsx 弹窗件+Cockpit
待开仓瓦片——修 z 序遮挡+加 UTC 时标（复用既有时标约定）。
domtest 断言两锚。

收口：新 wt→bootstrap→check_all 绿戳→收口文+推支不合并不部署。
