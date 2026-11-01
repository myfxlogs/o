// ui.tsx —— 卡面原语（T-0170：Card/骨架/错误态/状态点——诚实空态律：
// 错误显文案+重试钮，禁假零；骨架=卡面独立不整页灰）。
import type { ReactNode } from "react";

export function Card(props: {
  title: ReactNode;
  extra?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={"card " + (props.className ?? "")}>
      <div className="card-h">
        <h2>{props.title}</h2>
        {props.extra}
      </div>
      {props.children}
    </section>
  );
}

export function Skeleton(props: { rows?: number }) {
  return (
    <div className="skel" aria-hidden>
      {Array.from({ length: props.rows ?? 3 }, (_, i) => (
        <div key={i} className="skel-row" />
      ))}
    </div>
  );
}

export function ErrorState(props: { msg: string; onRetry: () => void }) {
  return (
    <div className="err-state" role="alert">
      <span>读件缺——{props.msg}</span>
      <button className="btn" onClick={props.onRetry}>
        重试
      </button>
    </div>
  );
}

// StateDot 态点（0绿/1黄/2红/3灰——语义在旁注文字，色点不带字面）。
export function StateDot(props: { state: number; title?: string; breathe?: boolean }) {
  const cls = props.state === 0 ? "s0" : props.state === 1 ? "s1" : props.state === 3 ? "s3" : "s2";
  return (
    <span
      className={"dot " + cls + (props.breathe ? " breathe" : "")}
      title={props.title ?? ""}
    />
  );
}

export function EmptyRow(props: { text: string }) {
  return <div className="empty-row">{props.text}</div>;
}

// EmptyState 空态（业主八条⑧：插画+引导，禁白板——插画=抽象线稿随主题墨色）。
export function EmptyState(props: { title: string; hint?: string; children?: ReactNode }) {
  return (
    <div className="empty-state">
      <svg viewBox="0 0 96 64" fill="none" aria-hidden>
        {/* 托盘 + 悬停圆点 + 虚线轨迹——「等一件事发生」的抽象形 */}
        <ellipse cx="48" cy="50" rx="30" ry="6" stroke="currentColor" strokeWidth="1.4" opacity="0.5" />
        <path d="M28 44h40l-4-14H32l-4 14Z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
        <circle cx="48" cy="22" r="7" stroke="currentColor" strokeWidth="1.6" />
        <path d="M48 29v1" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
        <path d="M14 18c8-6 18-8 26-6" stroke="currentColor" strokeWidth="1.2" strokeDasharray="3 4" opacity="0.6" />
        <path d="M82 46c-6 5-15 7-23 5" stroke="currentColor" strokeWidth="1.2" strokeDasharray="3 4" opacity="0.6" />
        <circle cx="76" cy="14" r="2" fill="currentColor" opacity="0.55" />
        <circle cx="20" cy="52" r="1.6" fill="currentColor" opacity="0.4" />
      </svg>
      <div className="es-title">{props.title}</div>
      {props.hint ? <div className="es-hint">{props.hint}</div> : null}
      {props.children}
    </div>
  );
}
