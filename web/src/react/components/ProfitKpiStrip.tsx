// ProfitKpiStrip.tsx —— 赚钱页核心指标条（O-41386 仿 TrxKpiStrip 三档首档：
// NAV/今日盈亏/30d净/年化率/部署率——大数字+StateDot 伴随灯）。
// 读面直挂禁新口径：NAV/今日盈亏=Overview；30d净=ListReports daily
// metrics.net 近30行合计（实现盈亏口径同报告）；年化率=monthly
// metrics.annualized（缺回退 weekly）；部署率=GetCockpit money.exposure
// /navTotal（敞口名义/净值——两字段皆服务端既有判态件）。
// ev_cashflow/ev_account/ev_position 信号帧重取零轮询。
import { useCallback, useEffect, useState } from "react";
import { client } from "../api";
import { onData } from "../stream";
import { StateDot } from "./ui";
import { fmtUsd } from "./fmt";

type KpiState = {
  nav: number;
  today: number;
  net30: number;
  net30Days: number; // 日报覆盖行数（合计口径披露——缺行不冒充30d）
  ann: number | null; // 年化率（小数——报告 metrics.annualized 原值）
  deploy: number | null; // exposure/navTotal 比率
};

const fmtSigned = (v: number) =>
  (v > 0 ? "+" : v < 0 ? "−" : "") +
  "$" +
  Math.abs(v).toLocaleString("zh-CN", { maximumFractionDigits: 2 });

const metricOf = (metrics: { k: string; v: number }[] | undefined, k: string) =>
  metrics?.find((m) => m.k === k)?.v;

function Kpi(props: {
  label: string;
  value: string;
  unit?: string;
  cls?: string;
  dot: { state: number; breathe: boolean };
  dotTitle: string;
  note?: string;
}) {
  return (
    <section className="card kpi" role="group" aria-label={props.label}>
      <div className="kpi-head">
        <span className="kpi-k">{props.label}</span>
        <StateDot state={props.dot.state} breathe={props.dot.breathe} title={props.dotTitle} />
      </div>
      <div className={"kpi-v" + (props.cls ? " " + props.cls : "")}>
        {props.value}
        {props.unit && <span className="kpi-unit">{props.unit}</span>}
      </div>
      {props.note && <div className="kpi-note">{props.note}</div>}
    </section>
  );
}

export function ProfitKpiStrip() {
  const [kpi, setKpi] = useState<KpiState | null>(null);
  const [err, setErr] = useState(false);

  const fetchAll = useCallback(async () => {
    try {
      const [ov, daily, monthly, weekly, cockpit] = await Promise.all([
        client.overview({}),
        client.listReports({ period: "daily", n: 30 }),
        client.listReports({ period: "monthly", n: 1 }),
        client.listReports({ period: "weekly", n: 1 }),
        client.getCockpit({}),
      ]);
      // 30d净=近30行日报 net 合计（实现盈亏口径）；覆盖行数如实注。
      let net30 = 0;
      let net30Days = 0;
      for (const rep of daily.reports ?? []) {
        const v = metricOf(rep.metrics, "net");
        if (v != null) {
          net30 += v;
          net30Days++;
        }
      }
      const ann =
        metricOf(monthly.reports?.[0]?.metrics, "annualized") ??
        metricOf(weekly.reports?.[0]?.metrics, "annualized") ??
        null;
      const nav = ov.navTotal;
      const exposure = cockpit.money?.exposure ?? 0;
      setKpi({
        nav,
        today: ov.pnlToday,
        net30,
        net30Days,
        ann,
        deploy: nav > 0 ? exposure / nav : null,
      });
      setErr(false);
    } catch {
      setErr(true);
    }
  }, []);

  useEffect(() => {
    void fetchAll();
    return onData((v) => {
      const t = (v as { topic: string }).topic;
      if (t === "ev_cashflow" || t === "ev_account" || t === "ev_position")
        void fetchAll();
    });
  }, [fetchAll]);

  if (err && !kpi) {
    return (
      <div className="profit-kpi-strip">
        <section className="card kpi kpi-err">核心指标读败——下一流帧自动回填</section>
      </div>
    );
  }
  if (!kpi) return null; // 首拍未落=不占位闪（次级卡骨架已示加载）
  const pos = (v: number) => v > 0;
  return (
    <div className="profit-kpi-strip">
      <Kpi
        label="NAV"
        value={fmtUsd(kpi.nav)}
        dot={{ state: 0, breathe: false }}
        dotTitle="基金净值（Overview moneyRows 口径）"
      />
      <Kpi
        label="今日盈亏"
        value={fmtSigned(kpi.today)}
        cls={pos(kpi.today) ? "u-good" : kpi.today < 0 ? "u-bad" : ""}
        dot={{ state: pos(kpi.today) ? 0 : 1, breathe: pos(kpi.today) }}
        dotTitle="今日实现盈亏（Overview 口径）"
      />
      <Kpi
        label="30d净"
        value={kpi.net30Days > 0 ? fmtSigned(kpi.net30) : "—"}
        cls={pos(kpi.net30) ? "u-good" : kpi.net30 < 0 ? "u-bad" : ""}
        dot={{ state: pos(kpi.net30) ? 0 : 1, breathe: pos(kpi.net30) }}
        dotTitle="近30行日报实现盈亏合计"
        note={kpi.net30Days > 0 ? `日报×${kpi.net30Days}` : "日报未出——如实空"}
      />
      <Kpi
        label="年化率"
        value={kpi.ann != null ? `${(kpi.ann * 100).toFixed(1)}%` : "—"}
        cls={kpi.ann != null && kpi.ann > 0 ? "u-good" : kpi.ann != null && kpi.ann < 0 ? "u-bad" : ""}
        dot={{ state: kpi.ann == null ? 1 : kpi.ann > 0 ? 0 : 1, breathe: kpi.ann != null && kpi.ann > 0 }}
        dotTitle="月报年化（net/本金基数×12——报告口径）"
        note={kpi.ann == null ? "报告未出年化键" : undefined}
      />
      <Kpi
        label="部署率"
        value={kpi.deploy != null ? `${(kpi.deploy * 100).toFixed(1)}%` : "—"}
        dot={{ state: 0, breathe: false }}
        dotTitle="部署敞口名义/NAV（GetCockpit money.exposure ÷ navTotal）"
        note={kpi.deploy == null ? "NAV 缺/零——如实空" : "敞口/NAV"}
      />
    </div>
  );
}
