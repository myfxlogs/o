// floorFocus.tsx —— /floor 聚焦层：点击物件到中央的细节卡（业主令⑮）。
// 白板卡=白板样；各卡挂「进账本页」深链跳问答页。GHOSTS=毛坯工位钦定清单。
import type { ReactNode } from "react";
import { navigate } from "../router";
import { plainText } from "../components/plain";
import { pbDate } from "../components/fmt";
import { AGE } from "../components/panel";

export type Tip = { k: string; x: number; y: number; title: string; body: string[]; to: string };

export const GHOSTS: { name: string; st: "ghost" | "build" | "done"; hint: string }[] = [
  { name: "B 元探", st: "ghost", hint: "缺仪器自动发探——自模型胚" },
  { name: "C 自态摘要", st: "ghost", hint: "会话冷启动账页" },
  { name: "D 目标发起", st: "build", hint: "进水口已通·7源已抓" },
  { name: "旧禁条作废", st: "build", hint: "种子行+引尸探在案" },
  { name: "做梦", st: "ghost", hint: "离线反刍成规律" },
  { name: "举一反三", st: "ghost", hint: "摔一次会一类" },
  { name: "连续自我", st: "ghost", hint: "破日记人格" },
  { name: "懂你", st: "ghost", hint: "业主意图模型" },
];

const fmtMoney = (v: number) =>
  "$" + v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const TO_LABEL: Record<string, string> = { "/": "机赚如何", "/device": "哪病了", "/decision": "何事要裁" };

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
  utc: string; bj: string;
}) {
  const { f } = props;
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
        <div className="iso-focus-h">收银屏 · NAV</div>
        <div className="iso-focus-big num">{props.nav ? fmtMoney(props.nav.total) : "—"}</div>
        <div className="iso-focus-row">
          今日 {props.nav ? (props.nav.today >= 0 ? "+" : "−") + fmtMoney(Math.abs(props.nav.today)) : "—"} ·
          累计 {props.nav ? (props.nav.cum >= 0 ? "+" : "−") + fmtMoney(Math.abs(props.nav.cum)) : "—"}
        </div>
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
    return (
      <div className="iso-focus-in">
        <div className="iso-focus-h">毛坯工位 · {gh.name}</div>
        <div className="iso-focus-row">{gh.hint}</div>
        <div className="iso-focus-big">{gh.st === "build" ? "🟡 在建" : "⬚ 毛坯"}</div>
        {go}
      </div>
    );
  }
  return null;
}
