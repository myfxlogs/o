// System.tsx —— 服务器运行管理面板（T-0173 业主④⑥终版+第一性重排：
// 页面使命=「生产与管道健康——哪里病了、怎么修」。层级：
//   ① 异常摘要条置顶（一切正常=一行轻带过；有病=红字点名）
//   ② 卡序=交易（生产）优先于采集——钱径面永远先于管道面
//   ③ 进程指标并入采集卡 SG 段（帧/带宽=SG 采集臂的组级事实，非独立卡）
//   ④ 主机资源垫底（硬件层）；席位在册收尾。
// 灯语全站同款：绿缓息/黄急促/红静止；硬化为描边+底色（玻璃底不洗白）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { useSnap } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { AGE, ExchSections, healthRows } from "../components/panel";
import { pbDate } from "../components/fmt";
import { HostMetricsCard } from "../components/hostmetrics";
import type { SystemComponent } from "../../gen/dashboardv1/core_pb";

// ProcStrip SG core 采集臂进程观测（并入采集卡 SG 段——组级事实行同构渲染）。
function ProcStrip(props: {
  data: NonNullable<Awaited<ReturnType<typeof client.getSystemHealth>>>;
}) {
  const d = props.data;
  const stale = (() => {
    const t = pbDate(d.lastFrame);
    return t ? AGE(t) : "—";
  })();
  const rejects = Object.entries(d.rejects ?? {}).filter(([, v]) => v > 0);
  const state = rejects.length > 0 ? 1 : 0;
  return (
    <div className="exch-sec">
      <div className="exch-label ink-3">进程观测</div>
      <div className="comp-row">
        <StateDot state={state} breathe title="okxws 采集臂进程计数" />
        <div className="comp-main">
          <span>okxws 采集臂</span>
          <div className="ink-3 comp-note">
            帧 {d.frames.toLocaleString("zh-CN")} · 末帧 {stale} · 带宽{" "}
            {d.netKbps.toFixed(0)} kbps
            {rejects.length > 0
              ? " · 拒因 " + rejects.map(([k, v]) => `${k}:${v}`).join(" ")
              : ""}
          </div>
        </div>
      </div>
    </div>
  );
}

export function SystemPage() {
  const h = useSnap(["ev_account", "ev_position"], () =>
    client.getSystemHealth({})
  );
  const ov = useSnap(["ev_position", "ev_account"], () => client.overview({}));

  const devComps = (h.data?.components ?? []).filter((c) => c.host === "dev");
  const sgComps = (h.data?.components ?? []).filter((c) => c.host === "sg");

  // 生产机行：venue 在线行在前（私有 WS 由 venue 行覆盖去重），健康行随后。
  const venueRows = [];
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
  const sgRowsFinal = [
    ...venueRows,
    ...healthRows(sgComps.filter((c) => c.component !== "OKX 私有 WS")),
  ];

  const devRows = healthRows(devComps);
  const devBad = devRows.filter((r) => r.state !== 0);
  const sgBad = sgRowsFinal.filter((r) => r.state !== 0);
  const allBad = [
    ...devBad.map((r) => ({ state: r.state, name: r.name })),
    ...sgBad.map((r) => ({ state: r.state, name: r.name })),
  ];

  return (
    <div className="page">
      {/* ① 异常摘要条置顶：一切正常=一行轻带过；有病=红框点名（第一性：
          异常是这页存在的理由，正常是顺便的） */}
      {h.loading && !h.data ? (
        <Skeleton rows={8} />
      ) : h.err && !h.data ? (
        <ErrorState msg={h.err} onRetry={h.reload} />
      ) : allBad.length === 0 ? (
        <div className="health-strip">
          <StateDot state={0} />
          <span>两机全线正常——{devComps.length + sgComps.length} 件组件在岗</span>
        </div>
      ) : (
        <div className="health-strip" style={{ borderColor: "var(--critical)" }}>
          <StateDot
            state={Math.max(...allBad.map((r) => r.state))}
            breathe={allBad.some((r) => r.state !== 2)}
          />
          <span>
            {allBad.length} 件异常：
            {allBad
              .slice(0, 3)
              .map((r) => r.name)
              .join("、")}
            {allBad.length > 3 ? " 等" : ""}
          </span>
        </div>
      )}

      {/* ② 交易（生产）第一——钱径面永远先于管道面 */}
      {h.data ? (
        <Card
          title={
            <>
              <StateDot
                state={Math.max(0, ...sgRowsFinal.map((r) => r.state))}
                breathe={sgRowsFinal.some((r) => r.state !== 2)}
              />{" "}
              生产机（SG）——交易与本地臂
            </>
          }
          extra={
            <span className="ink-3">
              {sgBad.length === 0 ? "全部正常" : `${sgBad.length} 件异常`}
            </span>
          }
        >
          {sgRowsFinal.length === 0 ? (
            <EmptyState title="无行在读" hint="SG 面落库后此处逐盏亮灯" />
          ) : (
            <ExchSections rows={sgRowsFinal} />
          )}
        </Card>
      ) : null}

      {/* ③ 采集——管道面（含 SG 采集臂进程观测组） */}
      {h.data ? (
        <Card
          title={
            <>
              <StateDot
                state={Math.max(0, ...devRows.map((r) => r.state))}
                breathe={devRows.some((r) => r.state !== 2)}
              />{" "}
              采集（dev）——行情与蒸馏管道
            </>
          }
          extra={
            <span className="ink-3">
              {devBad.length === 0 ? "全部正常" : `${devBad.length} 件异常`}
            </span>
          }
        >
          {devRows.length === 0 ? (
            <EmptyState title="无组件在册" hint="采集足迹落库后此处逐盏亮灯" />
          ) : (
            <>
              <ExchSections rows={healthRows(devComps)} />
              <ProcStrip data={h.data} />
            </>
          )}
        </Card>
      ) : null}

      {/* ④ 主机资源垫底（硬件层） */}
      <HostMetricsCard />

      <Card title="席位在册">
        {(ov.data?.staff ?? []).length === 0 ? (
          <EmptyState title="花名册暂缺" hint="席位注册后此处逐盏亮灯" />
        ) : (
          <div className="chips">
            {(ov.data?.staff ?? []).map((s) => (
              <span key={s.name} className="chip">
                <StateDot state={s.state} /> {s.name}
              </span>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
