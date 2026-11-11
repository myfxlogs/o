// floorFocus.tsx —— /floor 聚焦层：点击物件到中央的细节卡（业主令⑮）。
// 白板卡=白板样；各卡挂「进账本页」深链跳问答页。GHOSTS=毛坯工位钦定清单。
import type { ReactNode } from "react";
import { navigate } from "../router";
import { client } from "../api";
import { useLive } from "../hooks";
import { plainText } from "../components/plain";
import { pbDate } from "../components/fmt";
import { AGE } from "../components/panel";
import { P, pts, IsoBox } from "../components/iso";

export type Tip = { k: string; x: number; y: number; title: string; body: string[]; to: string };

export const GHOSTS: { name: string; st: "ghost" | "build" | "done"; hint: string; keys: string[] }[] = [
  { name: "B 元探", st: "ghost", hint: "缺仪器自动发探——自模型胚", keys: ["元探", "自动发探", "缺仪器"] },
  { name: "C 自态摘要", st: "ghost", hint: "会话冷启动账页", keys: ["自态", "#10489"] },
  { name: "D 目标发起", st: "ghost", hint: "进水口已通·7源已抓", keys: ["目标发起", "abcd", "A-D", "进水口", "intake"] },
  { name: "旧禁条作废", st: "ghost", hint: "种子行+引尸探在案", keys: ["禁条", "红线重定义", "redline", "法典"] },
  { name: "做梦", st: "ghost", hint: "离线反刍成规律", keys: ["做梦"] },
  { name: "举一反三", st: "ghost", hint: "摔一次会一类", keys: ["举一反三"] },
  { name: "连续自我", st: "ghost", hint: "破日记人格", keys: ["连续自我"] },
  { name: "懂你", st: "ghost", hint: "业主意图模型", keys: ["懂你"] },
];

// 毛坯态推导——钉墙最新匹配钉定态（开钉→在建；销/部署→完工；无钉→毛坯）。
// trail 新→旧序；DONE 词表=销/部署/落账毕，BUILD 词表=领命/工单/备记/开工。
const DONE_RE = /销案|部署销|落账毕|法典已部署|已部署|已完成|收口/;
export function ghostSt(gh: (typeof GHOSTS)[number], trail: { note: string; open: boolean }[]): "ghost" | "build" | "done" {
  const hit = trail.find((r) => gh.keys.some((k) => r.note.includes(k)));
  if (!hit) return "ghost";
  if (hit.open || !DONE_RE.test(hit.note)) return "build";
  return "done";
}

const fmtMoney = (v: number) =>
  "$" + v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const TO_LABEL: Record<string, string> = { "/": "机赚如何", "/device": "哪病了", "/decision": "何事要裁" };

type Hot = (tt: Omit<Tip, "x" | "y">, x: number, y: number, z?: number, cls?: string) => Record<string, unknown>;

// 毛坯工位排——待长器官钦定清单（暗=毛坯，金=在建，绿=完工）。
export function GhostRow(props: { hot: Hot; trail: { note: string; open: boolean }[] }) {
  return (
    <>
      {GHOSTS.map((gh, i) => {
        const x = 0.8 + i * 1.65, y = 8.3;
        const st = ghostSt(gh, props.trail);
        const gold = st === "build", done = st === "done";
        const c = gold ? { top: "#5c5132", front: "#4d4329", right: "#403723" }
          : done ? { top: "#2e4d3a", front: "#274233", right: "#1f3529" }
          : { top: "#2c3444", front: "#252c3a", right: "#1f2531" };
        const [nx, ny] = P(x + 0.6, y + 1.35);
        return (
          <g key={gh.name}
            {...props.hot({ k: "gh" + i, title: `毛坯位 · ${gh.name}`, body: [gh.hint, done ? "已完工" : gold ? "在建" : "毛坯"], to: "/decision" }, x + 0.6, y + 0.4, 0.8, gold ? "fx-breathe" : "")}>
            <IsoBox x={x} y={y} w={1.2} d={0.8} h={0.7} top={c.top} front={c.front} right={c.right} dashed={!done} op={0.85} />
            <text x={nx} y={ny + 8} className={`iso-cap${gold ? " gold" : done ? "" : " dim"}`} textAnchor="middle">{gh.name}{done ? "✓" : ""}</text>
          </g>
        );
      })}
    </>
  );
}

// venue 屏排——右墙顶带小屏，几所几屏（overview money 行按 venue 分组）。
export function VenueScreens(props: { hot: Hot; vens: [string, number][] }) {
  return (
    <>
      {props.vens.map(([vn, usd], i) => {
        const x = 8.6 + i * 1.15;
        const [lx, ly] = P(x + 0.45, 0, 2.45);
        return (
          <g key={vn} {...props.hot({ k: "ven:" + vn, title: `${vn} 资金屏`, body: [`合计 ${fmtMoney(usd)}`], to: "/" }, x + 0.45, 0, 2.7)}>
            <polygon points={pts([P(x, 0, 2.6), P(x + 0.9, 0, 2.6), P(x + 0.9, 0, 2.95), P(x, 0, 2.95)])} fill="#0a1020" stroke="#5a7ec2" strokeWidth="0.9" />
            <text x={lx} y={ly + 6} className="iso-cap" textAnchor="middle" fill="#9db8ff" fontSize="8.5">{vn}</text>
          </g>
        );
      })}
    </>
  );
}

// 聚焦卡数据源——场景页把已抓数据整包传入，本件只管排版。
export function FocusBody(props: {
  f: Tip;
  openEsc: { escId: bigint; note: string }[];
  openReq: { id: bigint }[];
  trail: { escId: bigint; note: string; open: boolean }[];
  probes: { escClass: string; escScope: string; open: boolean; lastEsc?: unknown }[];
  openProbes: number;
  engCanon: number; econCanon: number;
  nav?: { total: number; today: number; cum: number };
  session?: { running?: boolean; lastSpawn?: unknown; spawns24h?: unknown; closes24h?: unknown; fails24h?: unknown };
  money?: { label: string; section: string; ccy: string; usd: number; venue: string; note: string }[];
  live?: boolean;
  utc: string; bj: string;
}) {
  const { f } = props;
  // 仓单读面——聚焦打开才拉（ev_position 帧同刷）。
  const pos = useLive(["ev_position"], () => client.listOpenPositions({}));
  const hist = useLive(["ev_position"], () => client.listClosedPositions({ limit: 6, offset: 0 }));
  const go = f.to ? (
    <button className="iso-focus-go" onClick={() => navigate(f.to)}>进账本页 →{TO_LABEL[f.to]}</button>
  ) : null;
  const s = props.session;
  const body: Record<string, () => ReactNode> = {
    board: () => (
      <div className="iso-focus-in wb">
        <div className="wb-h">白板 · 待办——工单钉上就算活</div>
        {props.openEsc.length + props.openReq.length === 0 && <div className="wb-row dim">板上干净——零积压</div>}
        {props.openEsc.slice(0, 8).map((r) => (
          <div key={String(r.escId)} className="wb-row">🔴 {plainText(r.note).slice(0, 46)}</div>
        ))}
        {props.openReq.slice(0, 4).map((r) => (
          <div key={String(r.id)} className="wb-row">📌 条件请求 #{String(r.id)} · 待处理</div>
        ))}
        {go}
      </div>
    ),
    desk: () => (
      <div className="iso-focus-in">
        <div className="iso-focus-h">值班台 · 唤知会话</div>
        <div className="iso-focus-big">{s?.running ? "🟢 在班——正在干活" : "⚫ 下班了"}</div>
        <div className="iso-focus-row">{s?.lastSpawn ? `末唤起 ${AGE(pbDate(s.lastSpawn as Parameters<typeof pbDate>[0])!)}` : "唤起机未写行"}</div>
        <div className="iso-focus-row">24h 唤 {String(s?.spawns24h ?? 0)} · 收 {String(s?.closes24h ?? 0)} · 败 {String(s?.fails24h ?? 0)}</div>
        {go}
      </div>
    ),
    nav: () => (
      <div className="iso-focus-in">
        <div className="iso-focus-h">收银大屏 · NAV + 持仓 + 历史单</div>
        <div className="iso-focus-big num">{props.nav ? fmtMoney(props.nav.total) : "—"}</div>
        <div className="iso-focus-row">
          今日 {props.nav ? (props.nav.today >= 0 ? "+" : "−") + fmtMoney(Math.abs(props.nav.today)) : "—"} ·
          累计 {props.nav ? (props.nav.cum >= 0 ? "+" : "−") + fmtMoney(Math.abs(props.nav.cum)) : "—"}
        </div>
        <div className="iso-focus-h" style={{ marginTop: 10 }}>在仓 {pos.data?.positions?.length ?? "…"}</div>
        {(pos.data?.positions ?? []).slice(0, 5).map((p) => (
          <div key={String(p.id)} className="iso-focus-row">
            {p.venue} · {p.instId} · {p.direction} · ${p.notionalUsd.toFixed(0)}{p.template ? ` · ${p.template}` : ""}
          </div>
        ))}
        {(pos.data?.positions ?? []).length === 0 && !pos.loading && <div className="iso-focus-row dim">零在仓</div>}
        <div className="iso-focus-h" style={{ marginTop: 10 }}>最近平仓</div>
        {(hist.data?.positions ?? []).slice(0, 5).map((p) => (
          <div key={String(p.id)} className="iso-focus-row">
            {p.instId} · {p.settledAmount >= 0 ? "+" : "−"}{Math.abs(p.settledAmount).toFixed(2)} {p.settledCcy}
          </div>
        ))}
        {go}
      </div>
    ),
    wall: () => (
      <div className="iso-focus-in">
        <div className="iso-focus-h">日记钉墙 · ops_event</div>
        {props.trail.slice(0, 8).map((r) => (
          <div key={String(r.escId)} className="iso-focus-row">
            <span className={r.open ? "c-red" : "c-green"}>●</span> {plainText(r.note).slice(0, 52)}
          </div>
        ))}
        {go}
      </div>
    ),
    shelf: () => (
      <div className="iso-focus-in">
        <div className="iso-focus-h">经验书架 · canon</div>
        <div className="iso-focus-big num">工程 {props.engCanon} 盒</div>
        <div className="iso-focus-row">{props.econCanon ? `经济 ${props.econCanon} 盒` : "经济架·空——裁定 26 件无一入库（如实空态）"}</div>
        {go}
      </div>
    ),
    door: () => (
      <div className="iso-focus-in">
        <div className="iso-focus-h">门</div>
        <div className="iso-focus-row">它对外只有一扇门——飞书日行从这里递给你；你只收报告，不进屋插手。</div>
      </div>
    ),
    ruler: () => (
      <div className="iso-focus-in">
        <div className="iso-focus-h">身高尺 · 婴→少年→成年</div>
        <div className="iso-focus-big">现位：婴→少年（A-D 在建）</div>
        <div className="iso-focus-row">成年判据：① 点破率连续 30d=0 ② 真账年化 ≥8% 足样本期</div>
        {go}
      </div>
    ),
    utc: () => (
      <div className="iso-focus-in">
        <div className="iso-focus-h">UTC 钟——机器时间</div>
        <div className="iso-focus-big num">{props.utc}</div>
      </div>
    ),
    bj: () => (
      <div className="iso-focus-in">
        <div className="iso-focus-h">北京钟——你的时间</div>
        <div className="iso-focus-big num">{props.bj}</div>
      </div>
    ),
  };
  const k = f.k;
  if (body[k]) return <>{body[k]()}</>;
  if (k.startsWith("srv:")) {
    const nm = k.slice(4);
    return (
      <div className="iso-focus-in">
        <div className="iso-focus-h">机柜 · {nm}</div>
        <div className="iso-focus-big">{props.live ? "运行中" : "断流"}</div>
        <div className="iso-focus-row">常驻器官——o-core 家电谱一件；健康细节进「哪病了」</div>
        {go}
      </div>
    );
  }
  if (k.startsWith("ven:")) {
    const vn = k.slice(4);
    const rows = (props.money ?? []).filter((r) => r.venue === vn);
    const tot = rows.reduce((a, r) => a + (r.usd || 0), 0);
    return (
      <div className="iso-focus-in">
        <div className="iso-focus-h">{vn} · 资金屏</div>
        <div className="iso-focus-big num">{fmtMoney(tot)}</div>
        {rows.map((r, i) => (
          <div key={i} className="iso-focus-row">{r.label} · {r.section} · {fmtMoney(r.usd)}{r.note ? ` · ${r.note}` : ""}</div>
        ))}
        {go}
      </div>
    );
  }
  if (k === "sentry" || k.startsWith("probe"))
    return (
      <div className="iso-focus-in">
        <div className="iso-focus-h">哨兵窗 · {props.probes.length} 探在册</div>
        <div className="iso-focus-row">{props.openProbes ? `${props.openProbes} 探在报警` : "全谱静默"}</div>
        {props.probes.slice(0, 10).map((p, i) => (
          <div key={i} className="iso-focus-row">
            <span className={p.open ? "c-red" : p.lastEsc ? "c-gold" : "c-green"}>●</span> {plainText(p.escClass + "/" + p.escScope)}
          </div>
        ))}
        {props.probes.length > 10 && <div className="iso-focus-row dim">…另 {props.probes.length - 10} 探，进页看全谱</div>}
        {go}
      </div>
    );
  if (k.startsWith("gh")) {
    const gh = GHOSTS[Number(k.slice(2))];
    const st = ghostSt(gh, props.trail ?? []);
    return (
      <div className="iso-focus-in">
        <div className="iso-focus-h">毛坯工位 · {gh.name}</div>
        <div className="iso-focus-row">{gh.hint}</div>
        <div className="iso-focus-big">{st === "done" ? "🟢 已完工" : st === "build" ? "🟡 在建" : "⬚ 毛坯"}</div>
        {go}
      </div>
    );
  }
  return null;
}
