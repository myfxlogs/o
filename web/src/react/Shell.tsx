// Shell.tsx —— 应用壳（T-0170：桌面左栏/手机底部 4-tab 双形态单壳；主题
// 明暗 + Watch 连接状态点——断流黄点显形不静默）。
import { useEffect, useState, type ReactNode } from "react";
import { navigate, usePath, type RouteDef } from "./router";
import { onLive } from "./stream";

function ThemeButton() {
  const [theme, setTheme] = useState(() =>
    document.documentElement.dataset.theme === "dark"
      ? "dark"
      : document.documentElement.dataset.theme === "light"
        ? "light"
        : window.matchMedia("(prefers-color-scheme: dark)").matches
          ? "dark"
          : "light"
  );
  const flip = () => {
    const next = theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = next;
    try {
      localStorage.setItem("o_theme", next);
    } catch {}
    setTheme(next);
  };
  return (
    <button className="btn theme-btn" onClick={flip} title="明暗切换">
      {theme === "dark" ? "☾" : "☀"}
    </button>
  );
}

function LiveDot() {
  const [live, setLive] = useState<boolean | null>(null);
  useEffect(() => onLive((v) => setLive(Boolean(v))), []);
  return (
    <span
      className={"dot " + (live ? "s0" : live === false ? "s1" : "s1")}
      title={live ? "Watch 流在线——事件驱动实时" : "Watch 断流重连中——数据可能陈"}
    />
  );
}

function Clock() {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(t);
  }, []);
  return (
    <span className="clock">
      {now.toISOString().slice(11, 19)} <span className="ink-3">UTC</span>
    </span>
  );
}

export function Shell(props: { routes: RouteDef[]; children: ReactNode }) {
  const path = usePath();
  const active = props.routes.find((r) => r.path === path) ?? props.routes[0];
  const nav = (to: string) => (e: React.MouseEvent) => {
    e.preventDefault();
    navigate(to);
  };
  return (
    <div className="shell">
      <aside className="side">
        <div className="brand">
          O <LiveDot />
        </div>
        <nav>
          {props.routes.map((r) => (
            <a
              key={r.path}
              href={r.path}
              onClick={nav(r.path)}
              className={r.path === active?.path ? "on" : ""}
            >
              {r.label}
            </a>
          ))}
        </nav>
        <div className="side-f">
          <Clock />
          <ThemeButton />
        </div>
      </aside>
      <header className="topbar">
        <span className="brand-s">
          O <LiveDot />
        </span>
        <span className="top-f">
          <Clock />
          <ThemeButton />
        </span>
      </header>
      <main className="main">{props.children}</main>
      <nav className="tabbar">
        {props.routes.map((r) => (
          <a
            key={r.path}
            href={r.path}
            onClick={nav(r.path)}
            className={r.path === active?.path ? "on" : ""}
          >
            {r.label}
          </a>
        ))}
      </nav>
    </div>
  );
}
