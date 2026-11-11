// Device.tsx —— 设备径页（v2 三径②，吞并原系统页成唯一设备页——单源律：
// 组件面/venue 通道/hostmetrics/席位在册只在本页渲，他页指路不复读；
// 序=异常摘要→生产面→采集面→探针心跳→会话·部署→宿主→管道→席位（
// 异常优先，正常退背景）。判态全后端出（healthComps 同源/venue 行/
// esc_scope 心跳/DEPLOY 行），前端零判据。
import { client } from "../api";
import { useLive } from "../hooks";
import { EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { HostMetricsCard } from "../components/hostmetrics";
import { OrgPipelineTile } from "../components/pipeline";
import { AGE, ExchSections } from "../components/panel";
import { pbDate } from "../components/fmt";
import { PagerBar, usePager } from "../components/pager";
import { COMPONENT_CN, REJECT_CN, cn } from "../components/labels";
import type { FaceRow, GetSystemHealthResponse } from "../../gen/dashboardv1/core_pb";

const compTxt = (v: string) => cn(COMPONENT_CN, v, v);

// ProcStrip SG core 采集臂进程观测（原 System 页挪入——组级事实行同构渲染）。
function ProcStrip(props: { data: GetSystemHealthResponse }) {
  const d = props.data;
  const stale = (() => {
    const t = pbDate(d.lastFrame);
    return t ? AGE(t) : "—";
  })();
  // T-0441：拒因 snake_case 键过 REJECT_CN 词表——已知键人话、未知键归并
  // 「其他拒因」不逐键裸出；同键多值只计一次。
  const rejects = Object.entries(d.rejects ?? {}).filter(([, v]) => v > 0);
  const rejectTxt = rejects.map(([k, v]) =>
    `${REJECT_CN[k] ?? "其他拒因"}:${v}`).join(" ");
  const state = rejects.length > 0 ? 1 : 0;
  return (
    <div className="exch-sec">
      <div className="exch-label ink-3">进程观测</div>
      <div className="comp-row">
        <StateDot state={state} breathe title="OKX 采集臂进程计数" />
        <div className="comp-main">
          <span>OKX 采集臂</span>
          <div className="ink-3 comp-note">
            帧 {d.frames.toLocaleString("zh-CN")} · 末帧 {stale} · 带宽{" "}
            {d.netKbps.toFixed(0)} kbps
            {rejects.length > 0 ? " · 拒因 " + rejectTxt : ""}
          </div>
        </div>
      </div>
    </div>
  );
}

// FaceRow→ExchSections 行模型（panel.Row 同构——名过词表、note 后端出）。
function faceRow(f: FaceRow) {
  return {
    key: f.name,
    name: compTxt(f.name),
    state: f.state,
    ageIso: pbDate(f.lastWrite)?.toISOString() ?? null,
    note: f.note || undefined,
    title: f.fact,
    exch: f.exch || "OKX",
  };
}

export function DevicePage() {
  const g = useLive(["ev_account", "ev_position", "ev_pipeline"], () =>
    client.getFaceGrid({})
  );
  const h = useLive(["ev_account", "ev_position"], () =>
    client.getSystemHealth({})
  );
  const ov = useLive(["ev_position", "ev_account"], () => client.overview({}));
  const probePager = usePager(g.data?.probes ?? [], 8);

  if (g.loading && !g.data) return <Skeleton rows={6} />;
  if (g.err && !g.data) return <ErrorState msg={g.err} onRetry={g.reload} />;
  const d = g.data!;

  // 生产面行：venue 在线行在前（私有 WS 由 venue 行覆盖去重），健康行随后——
  // 与 System 去重律同款。
  const venueRows = [];
  for (const grp of ov.data?.venues ?? []) {
    for (const ch of grp.channels ?? []) {
      venueRows.push({
        key: "venue:" + grp.name + ":" + ch.channel,
        name: ch.name,
        state: ch.state,
        ageIso: ch.lastTs ?? null,
        note: ch.note || undefined,
        title: grp.name + " " + ch.channel,
        exch: grp.name.toUpperCase(),
      });
    }
  }
  const faces = d.faces ?? [];
  const sgFaces = faces.filter((f) => f.host === "sg").map(faceRow);
  const devFaces = faces.filter((f) => f.host === "oci").map(faceRow);
  const sgRowsFinal = [
    ...venueRows,
    ...sgFaces.filter((r) => r.key !== "OKX 私有 WS"),
  ];
  const allBad = [...devFaces, ...sgRowsFinal]
    .filter((r) => r.state !== 0)
    .map((r) => ({ state: r.state, name: r.name }));
  const worst = Math.max(0, ...allBad.map((r) => r.state));
  const s = d.session;
  const staff = ov.data?.staff ?? [];

  return (
    <div className="page v2">
      <section className="v2-hero">
        <div className="v2-eyebrow">
          <span className="v2-livedot" />
          哪病了 · LIVE
        </div>
        <h1 className="v2-title">
          {worst === 0 ? "全部组件正常" : worst === 1 ? "有组件节拍偏慢" : "有组件断流"}
        </h1>
        <p className="v2-lede">
          {faces.filter((f) => f.state === 0).length}/{faces.length} 绿
          {(d.probes ?? []).filter((p) => p.open).length > 0
            ? ` · 探针未决 ${(d.probes ?? []).filter((p) => p.open).length}`
            : ""}
          {s?.lastSpawn
            ? ` · 末唤起 ${AGE(pbDate(s.lastSpawn)!)}${s.running ? "（在跑）" : ""}`
            : ""}
        </p>
      </section>

      {/* 异常摘要条——有病点名，全绿一行带过（第一性：异常是这页存在的理由） */}
      {allBad.length > 0 && (
        <div className="v2-alert">
          <StateDot state={worst} breathe={allBad.some((r) => r.state !== 2)} />
          <span>
            {allBad.length} 件异常：
            {allBad.map((r) => r.name).join("、")}
          </span>
        </div>
      )}

      {/* 生产面 SG+采集面 OCI——双小卡入栅格并置（业主令 11-11：小卡
          一行多列） */}
      <div className="v2-cols">
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">生产面（SG）——交易与本地臂</h2>
          <span className="ink-3">{sgRowsFinal.length} 行</span>
        </div>
        {sgRowsFinal.length === 0 ? (
          <EmptyState title="无行在读" hint="SG 面落库后此处逐盏亮灯" />
        ) : (
          <ExchSections rows={sgRowsFinal} />
        )}
      </div>

      {/* 采集面 OCI——健康行+进程观测 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">采集面（OCI）——行情与蒸馏管道</h2>
          <span className="ink-3">{devFaces.length} 行</span>
        </div>
        {devFaces.length === 0 && !h.data ? (
          <EmptyState title="无组件在册" hint="采集足迹落库后此处逐盏亮灯" />
        ) : (
          <>
            <ExchSections rows={devFaces} />
            {h.data ? (
              <ProcStrip data={h.data} />
            ) : h.err ? (
              <ErrorState msg={h.err} onRetry={h.reload} />
            ) : null}
          </>
        )}
      </div>
      </div>

      {/* 小卡多列栅格（业主令 11-11：按卡片大小排版——探针心跳+会话部署
          +宿主+管道四小件两两并置，页面不再单列撑高） */}
      <div className="v2-cols">
      {/* 探针心跳——open 先行排（后端已序），分页防撑高 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">探针心跳</h2>
          <PagerBar {...probePager} onPage={probePager.setPage} />
        </div>
        {(d.probes ?? []).length === 0 ? (
          <EmptyState title="探针未注册" hint="首个探针升件后在此亮行" />
        ) : (
          <div className="v2-rows">
            {probePager.slice.map((p) => (
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
                    {p.lastEsc ? `末火 ${AGE(pbDate(p.lastEsc)!)}` : "未有火记录"}
                    {p.lastRecover ? ` · 末复绿 ${AGE(pbDate(p.lastRecover)!)}` : ""}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 会话·部署——唤起/收口尾流+近部署行（同一面板两段，发丝分隔） */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">会话 · 部署</h2>
          <span className="ink-3">
            {s?.lastSpawn
              ? `24h 唤${s.spawns24h} 收${s.closes24h}${s.fails24h > 0n ? ` 败${s.fails24h}` : ""}`
              : "唤起机未写行"}
          </span>
        </div>
        <div className="v2-rows">
          {(d.deploys ?? []).map((r) => (
            <div key={String(r.id)} className="v2-row">
              <StateDot state={r.rc === 0 ? 0 : 2} title={`rc=${r.rc}`} />
              <div className="v2-row-main">
                <span className="v2-row-title mono">{r.sha || "—"}</span>
                <span className="v2-row-sub ink-3">
                  部署 #{String(r.id)} · {r.ts ? AGE(pbDate(r.ts)!) : "—"}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
      </div>

      {/* 宿主资源+工程票管道（复用件——小卡并列入栅格） */}
      <div className="v2-cols">
        <HostMetricsCard />
        <OrgPipelineTile />
      </div>

      {/* 席位在册——心跳延迟红点名 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">席位在册</h2>
          {staff.length > 0 ? (
            <StateDot
              state={Math.max(0, ...staff.map((x) => x.state))}
              breathe={staff.some((x) => x.state !== 2)}
              title={`${staff.length} 席在册——红=心跳延迟超过 5 分钟${
                staff.filter((x) => x.state === 2).length
                  ? ` · 陈席: ${staff.filter((x) => x.state === 2).map((x) => x.name).join("、")}`
                  : ""
              }`}
            />
          ) : undefined}
        </div>
        {staff.length === 0 ? (
          <EmptyState title="花名册暂缺" hint="席位注册后此处逐盏亮灯" />
        ) : (
          <div className="chips v2-chips">
            {staff.map((x) => (
              <span
                key={x.name}
                className="chip"
                title={`${x.duty}——红=心跳延迟超过 5 分钟（检查进程与花名册）`}
              >
                <StateDot state={x.state} breathe={x.state !== 2} /> {x.name}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
