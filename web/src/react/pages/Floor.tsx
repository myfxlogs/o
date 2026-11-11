// Floor.tsx —— 交易大厅页（业主令 11-11 活物世界观：基金公司交易层。
// 大厅=O 的身体常驻，值班经理=唤知会话轮班（打卡进/读墙/落账/打卡走），
// 日记墙=ops_event 共享大脑，白板=工单面，毛坯工位=待长器官——缺什么画什么。
// 活性律：探针灯呼吸/经理进出/墙钉换/NAV 跳全由真数据驱动，
// 断流=厅灯暗（onLive 显形不假亮）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { useLive } from "../hooks";
import { onLive } from "../stream";
import { EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { pbDate } from "../components/fmt";
import { AGE } from "../components/panel";
import { plainText } from "../components/plain";

const fmtMoney = (v: number) =>
  "$" + v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

// 双钟——UTC+北京（业主令 11-11⑭）。
function Clock(props: { label: string; tz: string; now: Date }) {
  const s = props.now.toLocaleTimeString("zh-CN", {
    timeZone: props.tz, hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit",
  });
  return (
    <div className="h-clock">
      <span className="h-clock-label">{props.label}</span>
      <span className="h-clock-time num">{s}</span>
    </div>
  );
}

// 毛坯工位注册面——钦定清单（A-D 四件+成年四器官）。
// st: ghost=毛坯/build=在建/done=落成；驱动源=工单/升件面（#10489 自态账
// 落地后接真认领态——现凭 canon/事件已知进度标 building）。
const GHOSTS: { name: string; st: "ghost" | "build" | "done"; hint: string }[] = [
  { name: "B 元探", st: "ghost", hint: "缺仪器自动发探——自模型胚" },
  { name: "C 自态摘要", st: "ghost", hint: "会话冷启动账页" },
  { name: "D 目标发起", st: "build", hint: "进水口已通·7源已抓" },
  { name: "旧禁条作废", st: "build", hint: "种子行+引尸探在案" },
  { name: "做梦", st: "ghost", hint: "离线反刍成规律" },
  { name: "举一反三", st: "ghost", hint: "摔一次会一类" },
  { name: "连续自我", st: "ghost", hint: "破日记人格" },
  { name: "懂你", st: "ghost", hint: "业主意图模型" },
];

export function FloorPage() {
  const ov = useLive(["ev_position", "ev_account"], () => client.overview({}));
  const g = useLive(["ev_account", "ev_position", "ev_pipeline"], () =>
    client.getFaceGrid({})
  );
  const t = useLive(["ev_pipeline"], () => client.getDecisionTrail({ limit: 12 }));
  const [live, setLive] = useState(false);
  const [now, setNow] = useState(() => new Date());
  useEffect(() => onLive((v) => setLive(!!v)), []);
  useEffect(() => {
    const i = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(i);
  }, []);

  if (g.loading && !g.data) return <Skeleton rows={6} />;
  if (g.err && !g.data) return <ErrorState msg={g.err} onRetry={g.reload} />;
  const d = g.data!;
  const o = ov.data;
  const s = d.session;
  const openEsc = (t.data?.trail ?? []).filter((r) => r.open);
  const probes = d.probes ?? [];
  const openProbes = probes.filter((p) => p.open).length;
  const engCanon = Number(t.data?.mine?.engCanon ?? 0n);
  const econCanon = Number(t.data?.mine?.econCanon ?? 0n);
  const wallNotes = (t.data?.trail ?? []).slice(0, 4);

  return (
    <div className="page v2">
      <div className={`house-room${live ? " live" : ""}`}>
        {/* 天棚：双钟+屋灯（断流灯暗=显形） */}
        <div className="house-top">
          <span className={`house-lamp${live ? " on" : ""}`} title={live ? "数据流活着" : "数据流断了"} />
          <Clock label="UTC" tz="UTC" now={now} />
          <Clock label="北京" tz="Asia/Shanghai" now={now} />
        </div>

        {/* 后墙：日记墙（共享大脑）——经验架+墙钉+NAV 屏+身高尺 */}
        <div className="house-wall">
          <div className="wall-shelf">
            <div className="shelf-label ink-3">日记墙 · 经验架</div>
            <div className="shelf-row">
              {engCanon === 0 ? (
                <span className="shelf-empty">工程经验 空架</span>
              ) : (
                <>
                  <span className="shelf-k">工程</span>
                  {Array.from({ length: engCanon }).map((_, i) => (
                    <span key={i} className="shelf-box" title={`工程经验条目 ${i + 1}`} />
                  ))}
                  <span className="shelf-n num">{engCanon}</span>
                </>
              )}
            </div>
            <div className="shelf-row">
              {econCanon === 0 ? (
                <span className="shelf-empty">经济经验 空架——裁定 26 件无一入库</span>
              ) : (
                <>
                  <span className="shelf-k">经济</span>
                  {Array.from({ length: econCanon }).map((_, i) => (
                    <span key={i} className="shelf-box" />
                  ))}
                  <span className="shelf-n num">{econCanon}</span>
                </>
              )}
            </div>
            {/* 墙钉——最新上报行钉在墙上（流帧驱动换钉） */}
            <div className="wall-notes">
              {wallNotes.length === 0 ? (
                <span className="shelf-empty">墙上干净——无新钉</span>
              ) : (
                wallNotes.map((r) => (
                  <div key={String(r.escId)} className="wall-note" title={r.note}>
                    <span className={`pin${r.open ? " red" : ""}`} />
                    {plainText(r.note).slice(0, 42)}
                  </div>
                ))
              )}
            </div>
          </div>

          {/* NAV 显示屏——钱径大屏钉死 */}
          <div className="wall-screen">
            <div className="ws-label">收银屏 · NAV</div>
            <div className="ws-nav num">{o ? fmtMoney(o.navTotal) : "—"}</div>
            <div className="ws-sub">
              今日 {o ? (o.pnlToday >= 0 ? "+" : "−") + fmtMoney(Math.abs(o.pnlToday)) : "—"} ·
              累计 {o ? (o.pnlCum >= 0 ? "+" : "−") + fmtMoney(Math.abs(o.pnlCum)) : "—"}
            </div>
          </div>

          {/* 身高尺——婴→少年→成年刻度，验收两桩贴尺旁 */}
          <div className="wall-ruler">
            <span className="ruler-mark">成年</span>
            <span className="ruler-mark">少年</span>
            <span className="ruler-mark">婴</span>
            <span className="ruler-pin" style={{ bottom: "18%" }} title="现位：婴→少年（A-D 在建）" />
            <span className="ruler-goal">判据：点破率归零 + 年化≥8%</span>
          </div>
        </div>

        {/* 地板：哨兵岗 / 值班台 / 白板 */}
        <div className="house-floor">
          <div className="post sentry-post">
            <div className="post-label ink-3">
              哨兵岗 · {probes.length} 探在册{openProbes > 0 ? ` · ${openProbes} 报警` : ""}
            </div>
            <div className="sentry-line">
              {probes.slice(0, 6).map((p) => (
                <span key={p.escClass + "/" + p.escScope} className="sentry" title={p.escClass + "/" + p.escScope}>
                  <StateDot
                    state={p.open ? 1 : p.lastEsc ? 0 : 3}
                    breathe={p.open}
                  />
                </span>
              ))}
              {probes.length > 6 && <span className="ink-3">+{probes.length - 6}</span>}
            </div>
          </div>

          <div className="post manager-post">
            <div className="post-label ink-3">值班台 · 唤知会话</div>
            <div className={`manager${s?.running ? " on" : ""}`}>
              <span className="fig" />
              <span className="manager-state">
                {s?.running
                  ? "在班——正在干活"
                  : s?.lastSpawn
                    ? `下班了（末唤起 ${AGE(pbDate(s.lastSpawn)!)}）`
                    : "唤起机未写行"}
              </span>
            </div>
            <div className="post-sub ink-3">
              24h 唤{s?.spawns24h ?? 0} 收{s?.closes24h ?? 0}
              {(s?.fails24h ?? 0n) > 0n ? ` 败${s?.fails24h}` : ""}
            </div>
          </div>

          <div className="post board-post">
            <div className="post-label ink-3">白板 · 待办</div>
            {openEsc.length === 0 && (t.data?.condreqs ?? []).filter((r) => r.open).length === 0 ? (
              <EmptyState title="板上干净" hint="急件/工单落账后钉在这里" />
            ) : (
              <div className="board-rows">
                {openEsc.slice(0, 5).map((r) => (
                  <div key={String(r.escId)} className="board-row" title={r.note}>
                    <StateDot state={1} breathe />
                    <span>{plainText(r.note).slice(0, 30)}</span>
                  </div>
                ))}
                {(t.data?.condreqs ?? []).filter((r) => r.open).map((r) => (
                  <div key={String(r.id)} className="board-row" title={r.note}>
                    <StateDot state={1} />
                    <span>条件请求 #{String(r.id)}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* 前排：毛坯工位——待长器官钦定清单（暗=未长，黄=在建） */}
        <div className="ghost-row">
          {GHOSTS.map((gh) => (
            <div key={gh.name} className={`ghost-desk ${gh.st}`} title={gh.hint}>
              <span className="ghost-chair" />
              <span className="ghost-name">{gh.name}</span>
              <span className="ghost-st ink-3">
                {gh.st === "done" ? "落成" : gh.st === "build" ? "在建" : "毛坯"}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
