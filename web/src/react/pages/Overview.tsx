// Overview.tsx —— 总览（T-0170 第一性首屏：30 秒答三问——赚亏多少/仓什么
// 状态/系统活着没；hero 大数字+在持+健康汇总+最近动作，数据全走 Connect）。
import { client } from "../api";
import { useSnap } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { fmtUsd } from "../components/fmt";

const money = (v: number) =>
  "$" +
  v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

function pnlCls(v: number): string {
  return v > 0 ? "u-good" : v < 0 ? "u-bad" : "";
}

export function OverviewPage() {
  const ov = useSnap(["ev_position", "ev_cashflow", "ev_account"], () =>
    client.overview({})
  );
  const health = useSnap(["ev_account"], () => client.getSystemHealth({}));

  if (ov.loading && !ov.data) return <Skeleton rows={6} />;
  if (ov.err && !ov.data)
    return <ErrorState msg={ov.err} onRetry={ov.reload} />;
  const o = ov.data!;

  const worst = Math.max(
    0,
    ...(health.data?.components ?? []).map((c) => c.state)
  );
  const doing = o.doing ?? [];

  return (
    <div className="page">
      {/* Hero：三问之一——赚亏多少 */}
      <section className="hero">
        <div className="hero-item">
          <span className="hero-k">净值</span>
          <span className="hero-v">{money(o.navTotal)}</span>
        </div>
        <div className="hero-item">
          <span className="hero-k">今日</span>
          <span className={"hero-v " + pnlCls(o.pnlToday)}>
            {o.pnlToday >= 0 ? "+" : ""}
            {money(o.pnlToday)}
          </span>
        </div>
        <div className="hero-item">
          <span className="hero-k">累计</span>
          <span className={"hero-v " + pnlCls(o.pnlCum)}>
            {o.pnlCum >= 0 ? "+" : ""}
            {money(o.pnlCum)}
          </span>
        </div>
      </section>

      {/* 三问之三——系统一句话健康（点进系统页） */}
      <a className="health-strip" href="/system">
        <StateDot state={worst} />
        <span>
          {worst === 0
            ? "系统全部组件正常"
            : worst === 1
              ? "系统有组件节拍偏慢——点看明细"
              : "系统有组件断流/缺席——点看明细"}
        </span>
        <span className="ink-3">›</span>
      </a>

      {/* 三问之二——仓什么状态 */}
      <Card title="在持">
        {ov.err ? <ErrorState msg={ov.err} onRetry={ov.reload} /> : null}
        {doing.length === 0 ? (
          <EmptyState title="静持——无开仓在案" hint="系统在等下一个入场信号，来了这里会亮行" />
        ) : (
          <div className="feed">
            {doing.map((d, i) => (
              <div key={i} className="feed-row">
                <span>{d.text}</span>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card title="最近动作">
        {(o.recent ?? []).length === 0 ? (
          <EmptyState title="账面平静" hint="现金流落地后在此亮行" />
        ) : (
          <div className="feed">
            {(o.recent ?? []).map((s, i) => (
              <div key={i} className="feed-row mono">
                {s}
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card title="钱在哪" extra={<span className="ink-3">{fmtUsd(o.navTotal)}</span>}>
        {(o.money ?? []).length === 0 ? (
          <EmptyState title="读件缺——venue 行缺席" hint="账户面恢复后这里列出每一块钱" />
        ) : (
          <table className="rows">
            <tbody>
              {(o.money ?? []).map((m, i) => (
                <tr key={i}>
                  <td>{m.label}</td>
                  <td className="num mono">{m.note ? m.note : fmtUsd(m.usd)}</td>
                  <td className="ink-3">{m.section}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </div>
  );
}
