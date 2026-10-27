// System.tsx —— 服务器运行管理面板（T-0173 业主④终版：两卡=两台机器，
// 卡题即机名；卡内按交易所分组，venue 在线行并入所属交易所组（私有 WS
// 重复行去重——venue 行覆盖之）。灯语全站同款：绿缓息/黄急促/红静止。
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

// Row 统一行模型（health 组件与 venue 在线行同形渲染——灯/名/龄/注）。
type Row = {
  key: string;
  name: string;
  state: number;
  ageIso: string | null;
  note?: string;
  title?: string;
};

const EXCH_ORDER = ["OKX", "MT5", "蒸馏"];

function ExchSections(props: { rows: Row[] }) {
  const [, setTick] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setTick((x) => x + 1), 30000);
    return () => clearInterval(t);
  }, []);
  const byExch = new Map<string, Row[]>();
  for (const r of props.rows) {
    const ex = (r as { exch?: string }).exch ?? "OKX";
    if (!byExch.has(ex)) byExch.set(ex, []);
    byExch.get(ex)!.push(r);
  }
  const exchs = [...byExch.keys()].sort(
    (a, b) =>
      (EXCH_ORDER.indexOf(a) + 1 || 99) - (EXCH_ORDER.indexOf(b) + 1 || 99)
  );
  return (
    <>
      {exchs.map((ex) => (
        <div key={ex} className="exch-sec">
          <div className="exch-label ink-3">{ex}</div>
          {(byExch.get(ex) ?? []).map((r) => (
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
      ))}
    </>
  );
}

// health 组件→行（exch 挂行上供分组）。
function healthRows(comps: SystemComponent[]): Row[] {
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

export function SystemPage() {
  const h = useSnap(["ev_account", "ev_position"], () =>
    client.getSystemHealth({})
  );
  const ov = useSnap(["ev_position", "ev_account"], () => client.overview({}));

  const devComps = (h.data?.components ?? []).filter((c) => c.host === "dev");
  const sgComps = (h.data?.components ?? []).filter((c) => c.host === "sg");

  // 生产机卡：venue 在线行并入交易所组（在前），health 组件行随后；
  // OKX 私有 WS 由 venue 行覆盖（同一事实源——双行去重）。
  const venueRows: Row[] = [];
  for (const g of ov.data?.venues ?? []) {
    for (const ch of g.channels ?? []) {
      venueRows.push({
        key: "venue:" + g.name + ":" + ch.channel,
        name: ch.name,
        state: ch.state,
        ageIso: ch.lastTs ?? null,
        note: ch.note || undefined,
        title: g.name + " " + ch.channel,
        exch: g.name.toUpperCase(),
      });
    }
  }
  const sgRowsFinal: Row[] = [
    ...venueRows,
    ...healthRows(sgComps.filter((c) => c.component !== "OKX 私有 WS")),
  ];

  return (
    <div className="page">
      {h.loading && !h.data ? (
        <Skeleton rows={8} />
      ) : h.err && !h.data ? (
        <ErrorState msg={h.err} onRetry={h.reload} />
      ) : (
        <div className="host-grid">
          {/* 采集机（dev）——蒸馏/OKX/MT5 采集管道 */}
          <Card
            title={
              <>
                <StateDot
                  state={Math.max(0, ...devComps.map((c) => c.state))}
                  breathe={devComps.some((c) => c.state !== 2)}
                />{" "}
                采集机（dev）
              </>
            }
            extra={
              <span className="ink-3">
                {devComps.filter((c) => c.state !== 0).length === 0
                  ? "全部正常"
                  : `${devComps.filter((c) => c.state !== 0).length} 件异常`}
              </span>
            }
          >
            <ExchSections rows={healthRows(devComps)} />
          </Card>

          {/* 生产机（SG）——交易面+SG 臂+venue 在线行并组 */}
          <Card
            title={
              <>
                <StateDot
                  state={Math.max(
                    0,
                    ...sgRowsFinal.map((r) => r.state),
                    ...sgComps.map((c) => c.state)
                  )}
                  breathe={sgRowsFinal.some((r) => r.state !== 2)}
                />{" "}
                生产机（SG）
              </>
            }
            extra={
              <span className="ink-3">
                {sgRowsFinal.filter((r) => r.state !== 0).length === 0
                  ? "全部正常"
                  : `${sgRowsFinal.filter((r) => r.state !== 0).length} 件异常`}
              </span>
            }
          >
            {sgRowsFinal.length === 0 ? (
              <EmptyState title="无行在读" hint="SG 面落库后此处逐盏亮灯" />
            ) : (
              <ExchSections rows={sgRowsFinal} />
            )}
          </Card>
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
