// Device.tsx —— 设备径页（dashboard-spec v2 三径②：组件面格栅（healthComps
// 同源判态）+探针心跳+会话面+部署面——「死没死/哪病了」一页答；
// 判态全后端出，前端只渲 state 灯）。
import { client } from "../api";
import { useLive } from "../hooks";
import { EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { HostMetricsCard } from "../components/hostmetrics";
import { AGE } from "../components/panel";
import { pbDate } from "../components/fmt";
import { COMPONENT_CN, cn } from "../components/labels";

const compTxt = (v: string) => cn(COMPONENT_CN, v, v);

export function DevicePage() {
  const g = useLive(["ev_account", "ev_position", "ev_pipeline"], () =>
    client.getFaceGrid({})
  );

  if (g.loading && !g.data) return <Skeleton rows={6} />;
  if (g.err && !g.data) return <ErrorState msg={g.err} onRetry={g.reload} />;
  const d = g.data!;
  const worst = Math.max(0, ...(d.faces ?? []).map((f) => f.state));
  const s = d.session;

  return (
    <div className="page v2">
      <section className="v2-hero">
        <div className="v2-eyebrow">
          <span className="v2-livedot" />
          设备径 · LIVE
        </div>
        <h1 className="v2-title">
          {worst === 0 ? "全部组件正常" : worst === 1 ? "有组件节拍偏慢" : "有组件断流"}
        </h1>
        <p className="v2-lede">
          {(d.faces ?? []).filter((f) => f.state === 0).length}/{(d.faces ?? []).length} 绿
          {s?.lastSpawn
            ? ` · 末唤起 ${AGE(pbDate(s.lastSpawn)!)}（24h 唤${s.spawns24h} 收${s.closes24h}${s.fails24h > 0n ? ` 败${s.fails24h}` : ""}）`
            : ""}
        </p>
      </section>

      {/* 组件面——unified panel+hairlines（同级行一面板） */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">组件面</h2>
          <span className="ink-3">healthComps 同源判态</span>
        </div>
        <div className="v2-rows">
          {(d.faces ?? []).map((f) => (
            <div key={f.name} className="v2-row">
              <StateDot state={f.state} breathe={f.state === 0 || f.state === 1} title={f.fact} />
              <div className="v2-row-main">
                <span className="v2-row-title">{compTxt(f.name)}</span>
                <span className="v2-row-sub ink-3">
                  {f.exch || f.cat} ·{" "}
                  {f.lastWrite ? AGE(pbDate(f.lastWrite)!) : "无行"}
                  {f.lagS > 0n ? ` · 滞后 ${Math.round(Number(f.lagS))}s` : ""}
                  {f.note ? ` · ${f.note}` : ""}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* 探针心跳——esc_scope×最近升/复绿；open=未决在案 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">探针心跳</h2>
          <span className="ink-3">{(d.probes ?? []).filter((p) => p.open).length} 未决</span>
        </div>
        {(d.probes ?? []).length === 0 ? (
          <EmptyState title="探针未注册" hint="首个探针升件后在此亮行" />
        ) : (
          <div className="v2-rows">
            {(d.probes ?? []).map((p) => (
              <div key={p.escClass + "/" + p.escScope} className="v2-row">
                <StateDot
                  state={p.open ? 1 : p.lastEsc ? 0 : 3}
                  breathe={p.open}
                  title={p.open ? "未决升件在案" : "已销"}
                />
                <div className="v2-row-main">
                  <span className="v2-row-title">
                    {p.escClass}/{p.escScope}
                  </span>
                  <span className="v2-row-sub ink-3">
                    {p.lastEsc ? `末火 ${AGE(pbDate(p.lastEsc)!)}` : "7d 内未火"}
                    {p.lastRecover ? ` · 末复绿 ${AGE(pbDate(p.lastRecover)!)}` : ""}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 部署面——近 3 部署行 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">部署</h2>
          <span className="ink-3">DEPLOY 读径</span>
        </div>
        <div className="v2-rows">
          {(d.deploys ?? []).map((r) => (
            <div key={r.id} className="v2-row">
              <StateDot state={r.rc === 0 ? 0 : 2} title={`rc=${r.rc}`} />
              <div className="v2-row-main">
                <span className="v2-row-title mono">{r.sha || "—"}</span>
                <span className="v2-row-sub ink-3">
                  #{r.id} · {r.ts ? AGE(pbDate(r.ts)!) : "—"}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* 宿主资源格并入（spec：设备径收宿主面） */}
      <HostMetricsCard />
    </div>
  );
}
