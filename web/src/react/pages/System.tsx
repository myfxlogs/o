// System.tsx —— 服务器运行管理面板（T-0173 业主④终版：两卡=两台机器，
// 卡题即机名；卡内按交易所分组可收拢+logo；venue 行并入生产机卡；席位
// 灯纯三色。病因+处理建议服务端算定，黄红才带）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { useSnap } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { AGE, ExchSections, healthRows } from "../components/panel";
import { HostMetricsCard } from "../components/hostmetrics";
import { pbDate } from "../components/fmt";
import type { SystemComponent, } from "../../gen/dashboardv1/core_pb";

// SystemPage：面板页（数据=GetSystemHealth+Overview 双快照）。
export function SystemPage() {
  const h = useSnap(["ev_account", "ev_position"], () =>
    client.getSystemHealth({})
  );
  const ov = useSnap(["ev_position", "ev_account"], () => client.overview({}));

  const devComps = (h.data?.components ?? []).filter((c) => c.host === "dev");
  const sgComps = (h.data?.components ?? []).filter((c) => c.host === "sg");

  // 生产机卡：venue 在线行并入交易所组（在前），health 组件行随后；
  // OKX 私有 WS 由 venue 行覆盖（同一事实源——双行去重）。
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

      {/* 主机资源（CPU/内存/磁盘/网络——各机自报 30s 拍） */}
      <HostMetricsCard />

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
                <StateDot state={s.state} /> {s.name}
              </span>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
