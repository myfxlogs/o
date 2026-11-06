// Shell.tsx —— 应用壳（T-0170：桌面左栏/手机底部 4-tab 双形态单壳；主题
// 明暗 + Watch 连接状态点——断流黄点显形不静默。
// T-0411 交互③：菜单当前项=玻璃胶囊（NavCapsule 实测锚位滑动——沿向
// 拉长 10-15%，到位弹性回弹；纯呈现层，导航语义零改动）。
import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { navigate, usePath, type RouteDef } from "./router";
import { onLive } from "./stream";

// NavCapsule 菜单胶囊：nav 容器内绝对定位胶囊，实测当前项锚位驱动
// top/left/width/height 过渡（弹性贝塞尔见 10-shell.css）；位移发生时挂
// is-moving 沿向拉长（竖 menu=scaleY/横 tab=scaleX），transitionend 落定。
function NavCapsule(props: { navRef: React.RefObject<HTMLElement | null>; activeHref: string; variant: "side" | "tab" }) {
	const [box, setBox] = useState<{ top: number; left: number; width: number; height: number } | null>(null);
	const [moving, setMoving] = useState(false);
	const prevHref = useRef<string | null>(null);
	const measure = () => {
		const nav = props.navRef.current;
		if (!nav) return;
		const el = nav.querySelector<HTMLAnchorElement>(`a[href="${CSS.escape(props.activeHref)}"]`);
		if (!el) return;
		const nb = nav.getBoundingClientRect();
		const b = el.getBoundingClientRect();
		if (props.variant === "tab") {
			// 底栏胶囊：56px 栏内定高 40px 居中（锚位只取水平位）
			setBox({ top: b.top - nb.top + (b.height - 40) / 2, left: b.left - nb.left, width: b.width, height: 40 });
		} else {
			// 侧栏胶囊：锚位四周 4px 呼吸边（胶囊底略大于文字位）
			setBox({ top: b.top - nb.top - 2, left: b.left - nb.left - 4, width: b.width + 8, height: b.height + 4 });
		}
	};
	useLayoutEffect(() => {
		if (prevHref.current !== null && prevHref.current !== props.activeHref) setMoving(true);
		prevHref.current = props.activeHref;
		measure();
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [props.activeHref]);
	useEffect(() => {
		window.addEventListener("resize", measure);
		// 字体就绪后锚位可能漂——补测一拍
		document.fonts?.ready.then(measure).catch(() => {});
		return () => window.removeEventListener("resize", measure);
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [props.activeHref]);
	if (!box) return null;
	return (
		<span
			aria-hidden
			className={`nav-capsule ${props.variant}-capsule` + (moving ? " is-moving" : "")}
			style={{ top: box.top, left: box.left, width: box.width, height: box.height }}
			onTransitionEnd={(e) => {
				if (e.propertyName === "top" || e.propertyName === "left") setMoving(false);
			}}
		/>
	);
}

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
  const sideNavRef = useRef<HTMLElement | null>(null);
  const tabNavRef = useRef<HTMLElement | null>(null);
  return (
    <div className="shell">
      <aside className="side">
        <div className="brand">
          O <LiveDot />
        </div>
        <nav ref={sideNavRef}>
          <NavCapsule navRef={sideNavRef} activeHref={active?.path ?? ""} variant="side" />
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
      <nav className="tabbar" ref={tabNavRef}>
        <NavCapsule navRef={tabNavRef} activeHref={active?.path ?? ""} variant="tab" />
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
