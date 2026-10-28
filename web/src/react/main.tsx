// main.tsx —— SPA 单入口（T-0170：四页一栈，原 vanilla 总览+trx 双入口
// 双栈归一；数据面唯一入口=CoreService Connect——D-0009）。
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Shell } from "./Shell";
import type { RouteDef } from "./router";
import { usePath } from "./router";
import { OverviewPage } from "./pages/Overview";
import { SystemPage } from "./pages/System";
import { TrxPage } from "./pages/Trx";
import { ReviewPage } from "./pages/Review";
import "./styles/00-base.css";
import "./styles/10-shell.css";
import "./styles/20-cards.css";
import "./styles/03-table-chip.css";
import "./styles/06-buttons.css";
import "./styles/20-semantic.css";
import "./styles/22-effects.css";

const ROUTES: RouteDef[] = [
  { path: "/", label: "总览", el: <OverviewPage /> },
  { path: "/system", label: "系统", el: <SystemPage /> },
  { path: "/trx", label: "TRX", el: <TrxPage /> },
  { path: "/review", label: "复盘", el: <ReviewPage /> },
];

function App() {
  const path = usePath();
  const active = ROUTES.find((r) => r.path === path) ?? ROUTES[0];
  return (
    <Shell routes={ROUTES}>
      {/* 路由切换整页重挂（页内卡片各自以 topics 订阅局部刷新） */}
      <div key={active.path}>{active.el}</div>
    </Shell>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
