// panel.tsx —— 运行管理面板件库（T-0173 业主④⑥：行模型/交易所组收拢+logo/
// 健康行映射——System 页薄壳化拆分件）。
import { useEffect, useState } from "react";
import { StateDot } from "./ui";
import { pbDate } from "./fmt";
import type { SystemComponent } from "../../gen/dashboardv1/core_pb";

export const AGE = (ts: Date) => {
  const s = Math.max(0, (Date.now() - ts.getTime()) / 1000);
  if (s < 60) return `${s.toFixed(0)} 秒前`;
  if (s < 3600) return `${(s / 60).toFixed(0)} 分前`;
  if (s < 86400) return `${(s / 3600).toFixed(1)} 时前`;
  return `${(s / 86400).toFixed(1)} 天前`;
};

// Row 统一行模型（health 组件与 venue 在线行同形渲染——灯/名/龄/注）。
type Row = {
  key: string;
  name: string;
  state: number;
  ageIso: string | null;
  note?: string;
  title?: string;
  exch?: string;
};

const EXCH_ORDER = ["OKX", "MT5", "蒸馏"];

// ExchLogo 交易所标（OKX 五格/MT5 徽/蒸馏瓶——线稿随墨色）。
export function ExchLogo(props: { ex: string }) {
  if (props.ex === "OKX")
    return (
      <span className="exch-logo" aria-hidden>
        <svg viewBox="0 0 16 16" fill="currentColor">
          <rect x="1" y="1" width="4" height="4" />
          <rect x="11" y="1" width="4" height="4" />
          <rect x="6" y="6" width="4" height="4" />
          <rect x="1" y="11" width="4" height="4" />
          <rect x="11" y="11" width="4" height="4" />
        </svg>
      </span>
    );
  if (props.ex === "MT5")
    return (
      <span className="exch-logo" aria-hidden>
        <svg viewBox="0 0 16 16" fill="none">
          <rect
            x="1"
            y="3"
            width="14"
            height="10"
            rx="2"
            stroke="currentColor"
            strokeWidth="1.4"
          />
          <text
            x="8"
            y="10.6"
            textAnchor="middle"
            fontSize="6.5"
            fontWeight="700"
            fill="currentColor"
            stroke="none"
          >
            MT5
          </text>
        </svg>
      </span>
    );
  return (
    <span className="exch-logo" aria-hidden>
      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
        <path d="M6 2h4M7 2v4L3.5 12a2 2 0 0 0 1.8 3h5.4a2 2 0 0 0 1.8-3L9 6V2" />
        <path d="M5 10.5h6" strokeLinecap="round" />
      </svg>
    </span>
  );
}

export function ExchSections(props: { rows: Row[] }) {
  const [, setTick] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setTick((x) => x + 1), 30000);
    return () => clearInterval(t);
  }, []);
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const byExch = new Map<string, Row[]>();
  for (const r of props.rows) {
    const ex = r.exch ?? "OKX";
    if (!byExch.has(ex)) byExch.set(ex, []);
    byExch.get(ex)!.push(r);
  }
  const exchs = [...byExch.keys()].sort(
    (a, b) =>
      (EXCH_ORDER.indexOf(a) + 1 || 99) - (EXCH_ORDER.indexOf(b) + 1 || 99)
  );
  return (
    <>
      {exchs.map((ex) => {
        const rows = byExch.get(ex)!;
        const bad = rows.some((r) => r.state !== 0);
        // 智能默认：全绿收拢省视野，有异常自动张（手点可覆写）
        const isOpen = collapsed[ex] === undefined ? bad : !collapsed[ex];
        return (
          <div key={ex} className="exch-sec">
            <button
              className="exch-head"
              onClick={() =>
                setCollapsed((c) => ({ ...c, [ex]: isOpen }))
              }
              aria-expanded={isOpen}
            >
              <span className={"caret" + (isOpen ? " open" : "")}>▸</span>
              <ExchLogo ex={ex} />
              <span>{ex}</span>
              <span className="ink-3" style={{ textTransform: "none" }}>
                {rows.length}
              </span>
              <span className="num mono ink-3" style={{ marginLeft: "auto" }}>
                {bad ? `${rows.filter((r) => r.state !== 0).length} 异常` : "✓"}
              </span>
            </button>
            {isOpen && (
              <div className="exch-rows">
                {rows.map((r) => (
                  <div key={r.key} className="comp-row">
                    <StateDot
                      state={r.state}
                      breathe={r.state !== 2}
                      title={r.title ?? ""}
                    />
                    <div className="comp-main">
                      <span>{r.name}</span>
                      {r.note && r.state !== 0 ? (
                        <div className="comp-note">{r.note}</div>
                      ) : r.note ? (
                        <div className="ink-3 comp-note">{r.note}</div>
                      ) : null}
                    </div>
                    <span className="mono comp-age">
                      {r.ageIso ? AGE(new Date(r.ageIso)) : "—"}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </>
  );
}

// health 组件→行（exch 挂行上供分组）。
export function healthRows(comps: SystemComponent[]): Row[] {
  return comps.map((c) => ({
    key: c.component,
    name: c.component,
    state: c.state,
    ageIso: pbDate(c.lastWrite)?.toISOString() ?? null,
    note: c.note || undefined,
    title: c.fact,
    exch: c.exch || "OKX",
  }));
}

