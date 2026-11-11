// floorFocus.tsx —— /floor 聚焦层：点击物件到中央的细节卡（业主令⑮）。
// 白板卡=白板样；各卡挂「进账本页」深链跳问答页。GHOSTS=毛坯工位钦定清单。
import type { ReactNode } from "react";
import { navigate } from "../router";
import { client } from "../api";
import { useLive } from "../hooks";
import { plainText } from "../components/plain";
import { pbDate } from "../components/fmt";
import { AGE } from "../components/panel";
import { P, pts, Person } from "../components/iso";
import { Sprite } from "./officeParts";

export type Tip = { k: string; x: number; y: number; title: string; body: string[]; to: string };

export const GHOSTS: { name: string; st: "ghost" | "pin" | "build" | "done"; hint: string; keys: string[] }[] = [
  { name: "B 元探", st: "ghost", hint: "缺仪器自动发探——自模型胚", keys: ["元探", "自动发探", "缺仪器"] },
  { name: "C 自态摘要", st: "ghost", hint: "会话冷启动账页", keys: ["自态", "#10489"] },
  { name: "D 目标发起", st: "ghost", hint: "进水口已通·7源已抓", keys: ["目标发起", "abcd", "A-D", "进水口", "intake"] },
  { name: "旧禁条作废", st: "ghost", hint: "种子行+引尸探在案", keys: ["禁条", "红线重定义", "redline", "法典"] },
  { name: "做梦", st: "ghost", hint: "离线反刍成规律", keys: ["做梦"] },
  { name: "举一反三", st: "ghost", hint: "摔一次会一类", keys: ["举一反三"] },
  { name: "连续自我", st: "ghost", hint: "破日记人格", keys: ["连续自我"] },
  { name: "懂你", st: "ghost", hint: "业主意图模型", keys: ["懂你"] },
];

// 毛坯态推导——事实面（ops_event 全事件类，新→旧序）最新匹配行定态。
// 判词取关键词 ±25 字邻域：一行可带多器官不同定性（如 #10507 同注
// 「法典部署销」∧「自态账领命」），整行判会串味。
export type Fact = { id: bigint; note: string };
// 四态词谱（±25 字邻域判，done>build>pin 优先级，无词兜底 build=有动过）：
//   pin=认领未动工（领命/工单/备记类）；build=真在施（施工/落码/处置类）。
const DONE_RE = /销案|部署销|落账毕|法典已部署|已部署|已完成|收口|上线|落地|启用|投产|生效/;
const BUILD_RE = /施工|在施|开工|落码|码修|修码|部署中|处置中|实现|开发中|改造|推进/;
const PIN_RE = /领命|工单|业主令|备记|清单|纳入|认领|立项|排期|消化径/;
export function ghostFact(gh: (typeof GHOSTS)[number], facts: Fact[]): Fact | undefined {
  return facts.find((r) => gh.keys.some((k) => r.note.includes(k)));
}
export function ghostSt(gh: (typeof GHOSTS)[number], facts: Fact[]): "ghost" | "pin" | "build" | "done" {
  const hit = ghostFact(gh, facts);
  if (!hit) return "ghost";
  const near = (re: RegExp) => gh.keys.some((k) => {
    const i = hit.note.indexOf(k);
    return i >= 0 && re.test(hit.note.slice(Math.max(0, i - 25), i + k.length + 25));
  });
  return near(DONE_RE) ? "done" : near(BUILD_RE) ? "build" : near(PIN_RE) ? "pin" : "build";
}

const fmtMoney = (v: number) =>
  "$" + v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const TO_LABEL: Record<string, string> = { "/": "机赚如何", "/device": "哪病了", "/decision": "何事要裁" };

type Hot = (tt: Omit<Tip, "x" | "y">, x: number, y: number, z?: number, cls?: string) => Record<string, unknown>;

// 器官工位排——2×4 桌阵（前排水泥区）。位=小桌+显示器+转椅+人仔，
// 态随事实推：毛坯=桌椅人全淡影，立项=钢蓝人仔+桌📌旗，
// 在建=金人仔呼吸在敲，完工=绿人仔名牌✓。
const GH_TONE: Record<string, string> = { ghost: "#8a93a3", pin: "#7fa8d0", build: "#d9a62e", done: "#3ddc84" };
export function GhostRow(props: { hot: Hot; facts: Fact[] }) {
  return (
    <>
      {GHOSTS.map((gh, i) => {
        const x = 1.5 + (i % 4) * 2.45, y = i < 4 ? 7.35 : 8.8, cx = x + 0.75;
        const st = ghostSt(gh, props.facts);
        const fact = ghostFact(gh, props.facts);
        const done = st === "done", pin = st === "pin", dim = st === "ghost";
        const stLb = done ? "已完工" : st === "build" ? "在建" : pin ? "已立项" : "毛坯";
        const [nx, ny] = P(cx, y + 1.62);
        return (
          <g key={gh.name}
            {...props.hot({ k: "gh" + i, title: `工位 · ${gh.name}`, body: [gh.hint, stLb, fact ? `据 #${String(fact.id)}` : "无事实行"], to: "/decision" }, cx, y + 0.5, 0.9, st === "build" ? "fx-breathe" : "")}>
            <g opacity={dim ? 0.5 : 1}>
              <Sprite name="desk" x={cx} y={y + 0.62} w={66} />
              <Sprite name="screen" x={cx} y={y + 0.42} z={0.55} w={20} />
              {/* 人仔在桌前背对镜头——面对显示器在敲/占位影 */}
              <Person x={cx} y={y + 1.12} tone={GH_TONE[st]} breathe={st === "build"} />
            </g>
            {pin && <text x={P(cx - 0.3, y + 0.35, 0.72)[0]} y={P(cx - 0.3, y + 0.35, 0.72)[1]} fontSize="11">📌</text>}
            <text x={nx} y={ny + 8} className={`iso-cap${st === "build" ? " gold" : done ? "" : " dim"}`} textAnchor="middle">{gh.name}{done ? "✓" : ""}</text>
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
  facts: Fact[];
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
    const fact = ghostFact(gh, props.facts ?? []);
    const st = fact ? ghostSt(gh, props.facts ?? []) : "ghost";
    return (
      <div className="iso-focus-in">
        <div className="iso-focus-h">毛坯工位 · {gh.name}</div>
        <div className="iso-focus-row">{gh.hint}</div>
        <div className="iso-focus-big">{st === "done" ? "🟢 已完工" : st === "build" ? "🟡 在建" : st === "pin" ? "📌 已立项" : "⬚ 毛坯"}</div>
        {fact && <div className="iso-focus-row dim">据 #{String(fact.id)} · {plainText(fact.note).slice(0, 40)}</div>}
        {go}
      </div>
    );
  }
  return null;
}
