// moneytree.tsx —— 钱在哪 venue 树（O-41386 抽离复用：Overview 钱在哪卡
// 与赚钱页 venue 余额卡同吃 Overview.money 一读面——抽件防第二套渲染）。
// 按平台树状——okx 多户收拢树下，点击展开看明细行；总额=树头常显不藏。
import { useState } from "react";
import type { MoneyRow } from "../../gen/dashboardv1/core_pb";
import { fmtUsd } from "./fmt";
import { VENUE_CN } from "./labels";

export function MoneyTree(props: { rows: MoneyRow[] }) {
  const [open, setOpen] = useState<Record<string, boolean>>({});
  const groups = new Map<string, MoneyRow[]>();
  for (const m of props.rows) {
    const v = m.venue || "其他";
    if (!groups.has(v)) groups.set(v, []);
    groups.get(v)!.push(m);
  }
  return (
    <div className="tree">
      {[...groups.entries()].map(([venue, rs]) => {
        const total = rs.reduce((a, r) => a + r.usd, 0);
        const isOpen = !!open[venue];
        return (
          <div key={venue} className="tree-node">
            <button
              className="tree-head"
              onClick={() => setOpen((o) => ({ ...o, [venue]: !o[venue] }))}
              aria-expanded={isOpen}
            >
              <span className={"caret" + (isOpen ? " open" : "")}>▸</span>
              <span className="tree-name">{VENUE_CN[venue] ?? venue}</span>
              <span className="ink-3 tree-n">{rs.length} 行</span>
              <span className="num mono tree-total">{fmtUsd(total)}</span>
            </button>
            {isOpen && (
              <table className="rows">
                <tbody>
                  {rs.map((m, i) => (
                    <tr key={i}>
                      <td>{m.label}</td>
                      <td className="num mono">{m.note ? m.note : fmtUsd(m.usd)}</td>
                      <td className="ink-3">{m.section}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        );
      })}
    </div>
  );
}
