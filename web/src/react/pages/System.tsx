// System.tsx —— 服务器运行管理面板（T-0173 业主④：采集+系统合一页——
// 两卡【采集】【交易（生产）】，卡内按机→按交易所分组组件；席位在册保留
// （非采集席的唯一活性板），灯语与交易所同款：绿缓息/黄急促/红静止。
// 病因+处理建议服务端算定，黄红才带。
import { useEffect, useState } from "react";
import { client } from "../api";
import { useSnap } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { pbDate } from "../components/fmt";
import type { SystemComponent } from "../../gen/dashboardv1/core_pb";

const AGE = (ts: Date) => {
  const s = Math.max(0, (Date.now() - ts.getTime()) / 1000);
  if (s < 60) return `${s.toFixed(0)} 秒前`;
  if (s < 3600) return `${(s / 60).toFixed(0)} 分前`;
  if (s < 86400) return `${(s / 3600).toFixed(1)} 时前`;
  return `${(s / 86400).toFixed(1)} 天前`;
};

const HOSTS: { key: string; label: string }[] = [
  { key: "dev", label: "采集机（dev）" },
  { key: "sg", label: "生产机（SG）" },
];

// CompGroup 卡内组件组：机→交易所两级分组，组头+逐行组件。
function CompGroup(props: { comps: SystemComponent[] }) {
  const [, setTick] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setTick((x) => x + 1), 30000);
    return () => clearInterval(t);
  }, []);
  return (
    <>
      {HOSTS.map(({ key, label }) => {
        const hostComps = props.comps.filter((c) => c.host === key);
        if (hostComps.length === 0) return null;
        const exchs = [...new Set(hostComps.map((c) => c.exch || "其他"))];
        return (
          <div key={key} className="host-sec">
            <div className="host-label">
              <StateDot
                state={Math.max(0, ...hostComps.map((c) => c.state))}
                breathe={hostComps.some((c) => c.state !== 2)}
              />
              {label}
            </div>
            {exchs.map((ex) => (
              <div key={ex} className="exch-sec">
                <div className="exch-label ink-3">{ex}</div>
                {hostComps
                  .filter((c) => (c.exch || "其他") === ex)
                  .map((c) => {
                    const d = pbDate(c.lastWrite);
                    return (
                      <div key={c.component} className="comp-row">
                        <StateDot
                          state={c.state}
                          breathe={c.state !== 2}
                          title={c.fact}
                        />
                        <div className="comp-main">
                          <span>{c.component}</span>
                          {c.note && c.state !== 0 ? (
                            <div className="comp-note">{c.note}</div>
                          ) : c.note ? (
                            <div className="ink-3 comp-note">{c.note}</div>
                          ) : null}
                        </div>
                        <span className="mono comp-age">{d ? AGE(d) : "—"}</span>
                      </div>
                    );
                  })}
              </div>
            ))}
          </div>
        );
      })}
    </>
  );
}

function HealthCard(props: {
  title: string;
  cat: string;
  comps: SystemComponent[];
}) {
  const mine = props.comps.filter((c) => c.cat === props.cat);
  const worst = Math.max(0, ...mine.map((c) => c.state));
  const bad = mine.filter((c) => c.state !== 0).length;
  return (
    <section className="card">
      <div className="card-h">
        <h2>
          <StateDot state={worst} breathe={worst !== 2} /> {props.title}
        </h2>
        <span className="ink-3">{bad === 0 ? "全部正常" : `${bad} 件异常`}</span>
      </div>
      {mine.length === 0 ? (
        <EmptyState title="无组件在册" hint="组件足迹落库后此处逐盏亮灯" />
      ) : (
        <CompGroup comps={mine} />
      )}
    </section>
  );
}

export function SystemPage() {
  const h = useSnap(["ev_account", "ev_position"], () =>
    client.getSystemHealth({})
  );
  const ov = useSnap(["ev_position", "ev_account"], () => client.overview({}));
  const comps = h.data?.components ?? [];
  return (
    <div className="page">
      {h.loading && !h.data ? (
        <Skeleton rows={8} />
      ) : h.err && !h.data ? (
        <ErrorState msg={h.err} onRetry={h.reload} />
      ) : (
        <div className="host-grid">
          <HealthCard title="采集" cat="collect" comps={comps} />
          <HealthCard title="交易（生产）" cat="trade" comps={comps} />
        </div>
      )}

      {/* SG core 采集臂进程观测 */}
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

      <Card title="交易所在线（SG 视角）">
        {(ov.data?.venues ?? []).length === 0 ? (
          <Skeleton rows={3} />
        ) : (
          (ov.data?.venues ?? []).map((g) => (
            <div key={g.name} className="venue-group">
              <div className="venue-name">{g.name}</div>
              {(g.channels ?? []).map((ch) => (
                <div key={ch.channel} className="feed-row">
                  <StateDot state={ch.state} breathe={ch.state !== 2} />
                  <span>{ch.name}</span>
                  <span className="ink-3 mono">
                    {ch.lastTs ? AGE(new Date(ch.lastTs)) : "读件缺"}
                  </span>
                </div>
              ))}
            </div>
          ))
        )}
      </Card>

      <Card title="席位在册">
        {(ov.data?.staff ?? []).length === 0 ? (
          <EmptyState title="花名册暂缺" hint="席位注册后此处逐盏亮灯" />
        ) : (
          <div className="chips">
            {(ov.data?.staff ?? []).map((s) => (
              <span key={s.name} className="chip">
                <StateDot state={s.state} breathe={s.state !== 2} /> {s.name}
              </span>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
