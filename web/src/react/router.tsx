// router.tsx —— 极简 history 路由（T-0170：四页单 SPA，依赖极简律——
// 不引 react-router，pushState+popstate 四十行自足；服务端 mountUI 已
// SPA fallback，深链直开可达）。
import { useEffect, useState, type ReactNode } from "react";

export type RouteDef = { path: string; label: string; el: ReactNode };

export function usePath(): string {
  const [path, setPath] = useState(() => window.location.pathname);
  useEffect(() => {
    const on = () => setPath(window.location.pathname);
    window.addEventListener("popstate", on);
    return () => window.removeEventListener("popstate", on);
  }, []);
  return path;
}

export function navigate(to: string) {
  if (to === window.location.pathname) return;
  window.history.pushState(null, "", to);
  window.dispatchEvent(new PopStateEvent("popstate"));
}
