// Collect.tsx —— 采集机页（T-0173 业主③追加：两台主机=两大瓦片，各管各的
// 组件；每组件一行=状态呼吸灯+末拍龄+病因/处理建议（服务端算定，黄红才带
// 建议）。30s 自动重抓+Watch 事件随拍。
import { useEffect, useState } from "react";
import { client } from "../api";
import { useSnap } from "../hooks";
import { EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { pbDate } from "../components/fmt";
import type { SystemComponent } from "../../gen/dashboardv1/core_pb";

const HOSTS: { key: string; label: string; sub: string }[] = [
  { key: "dev", label: "采集机（dev）", sub: "行情 WS/REST 轮询/蒸馏/MT5 桥" },
  { key: "sg", label: "生产机（SG）", sub: "私有流/利润流/枚举/采样臂" },
];

const AGE = (ts: Date) => {
  const s = Math.max(0, (Date.now() - ts.getTime()) / 1000);
  if (s < 60) return `${s.toFixed(0)} 秒前`;
  if (s < 3600) return `${(s / 60).toFixed(0)} 分前`;
  if (s < 86400) return `${(s / 3600).toFixed(1)} 时前`;
  return `${(s / 86400).toFixed(1)} 天前`;
};

function HostTile(props: {
  label: string;
  sub: string;
  comps: SystemComponent[];
}) {
  const [, setTick] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setTick((x) => x + 1), 30000);
    return () => clearInterval(t);
  }, []);
  const worst = Math.max(0, ...props.comps.map((c) => c.state));
  const bad = props.comps.filter((c) => c.state !== 0).length;
  return (
    <section className="card host-tile">
      <div className="card-h">
        <h2>
          <StateDot state={worst} breathe={worst !== 2} /> {props.label}
        </h2>
        <span className="ink-3">{bad === 0 ? "全部正常" : `${bad} 件异常`}</span>
      </div>
      <div className="ink-3 host-sub">{props.sub}</div>
      <div className="comp-rows">
        {props.comps.map((c) => {
          const d = pbDate(c.lastWrite);
          return (
            <div key={c.component} className="comp-row">
              <StateDot state={c.state} breathe={c.state !== 2} title={c.fact} />
              <div className="comp-main">
                <span>{c.component}</span>
                {c.note && c.state !== 0 ? (
                  <div className="comp-note">{c.note}</div>
                ) : c.note ? (
                  <div className="ink-3 comp-note">{c.note}</div>
                ) : null}
              </div>
              <span className="mono comp-age">
                {d ? AGE(d) : "—"}
              </span>
            </div>
          );
        })}
      </div>
    </section>
  );
}

export function CollectPage() {
  const h = useSnap(["ev_account", "ev_position"], () =>
    client.getSystemHealth({})
  );
  if (h.loading && !h.data) return <Skeleton rows={8} />;
  if (h.err && !h.data) return <ErrorState msg={h.err} onRetry={h.reload} />;
  const comps = h.data?.components ?? [];
  return (
    <div className="page">
      {comps.length === 0 ? (
        <EmptyState title="健康板暂无组件行" hint="采集足迹落库后此处逐盏亮灯" />
      ) : (
        <div className="host-grid">
          {HOSTS.map((host) => (
            <HostTile
              key={host.key}
              label={host.label}
              sub={host.sub}
              comps={comps.filter((c) => c.host === host.key)}
            />
          ))}
        </div>
      )}
      {h.data ? (
        <section className="card">
          <div className="card-h">
            <h2>进程指标（SG core 采集臂）</h2>
          </div>
          <div className="stat-strip">
            <span>
              帧 <b className="mono">{h.data.frames.toLocaleString("zh-CN")}</b>
            </span>
            <span>
              末帧{" "}
              <b className="mono">
                {(() => {
                  const d = pbDate(h.data?.lastFrame);
                  return d ? AGE(d) : "—";
                })()}
              </b>
            </span>
            <span>
              带宽 <b className="mono">{h.data.netKbps.toFixed(0)} kbps</b>
            </span>
            <span>
              拒因{" "}
              {Object.entries(h.data.rejects ?? {}).filter(([, v]) => v > 0)
                .length === 0 ? (
                <b>零</b>
              ) : (
                Object.entries(h.data.rejects ?? {})
                  .filter(([, v]) => v > 0)
                  .map(([k, v]) => (
                    <b key={k} className="u-bad mono">
                      {" "}
                      {k}:{v}
                    </b>
                  ))
              )}
            </span>
          </div>
        </section>
      ) : null}
    </div>
  );
}
