// EarnToggle.tsx —— Earn 自动申购停用开关（T-0342——业主令：页面临时停用
// 自动转入 Simple Earn）。写径=SetEarnAutoPurchase 同源窄写（服务端 Origin
// 闸+只写一键）；按钮二次确认；文案钉死「只停止 O 自动转入，不赎回现有
// Earn；开仓需要时仍可赎回」——停用态禁声称已赎回。
import { useCallback, useEffect, useState } from "react";
import { client, errText } from "../api";

const CONFIRM_OFF =
  "临时停用自动转入 Simple Earn？\n\n" +
  "只停止 O 自动转入（闲余回存/资金户直购/残币归口）——\n" +
  "不赎回、不移动现有 Earn 持仓；开仓需要时仍可自动赎回。";
const CONFIRM_ON = "恢复自动转入 Simple Earn？";

export function EarnToggle() {
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [meta, setMeta] = useState({ by: "", at: "", note: "" });
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    client
      .getEarnAutoPurchase({})
      .then((r) => {
        setEnabled(r.enabled);
        setMeta({
          by: r.updatedBy,
          at: r.updatedAt
            ? new Date(Number(r.updatedAt.seconds) * 1000)
                .toISOString()
                .slice(5, 16)
                .replace("T", " ")
            : "",
          note: r.note,
        });
        setErr(null);
      })
      .catch((e: unknown) => setErr(errText(e)));
  }, []);

  useEffect(load, [load]);

  const flip = () => {
    if (enabled === null || busy) return;
    const next = !enabled;
    if (!window.confirm(next ? CONFIRM_ON : CONFIRM_OFF)) return; // 二次确认钦定
    setBusy(true);
    client
      .setEarnAutoPurchase({ enabled: next, reason: "业主前端手动翻位" })
      .then((r) => {
        setEnabled(r.enabled);
        setMeta((m) => ({ ...m, by: r.updatedBy }));
      })
      .catch((e: unknown) => setErr(errText(e)))
      .finally(() => {
        setBusy(false);
        load(); // 写后回读真态（updated_at/note 同步——非回显）
      });
  };

  return (
    <div className="earn-toggle">
      <span className="ink-3">自动转入 Simple Earn：</span>
      <span className={enabled === false ? "u-bad" : "u-good"}>
        {enabled === null ? "…" : enabled ? "开" : "临时停用"}
      </span>
      <button className="btn" disabled={enabled === null || busy} onClick={flip}>
        {enabled ? "停用" : "恢复"}
      </button>
      {meta.at ? (
        <span className="ink-3" title={meta.note || undefined}>
          {meta.at} {meta.by ? `by ${meta.by}` : ""}
          {meta.note ? ` · ${meta.note}` : ""}
        </span>
      ) : null}
      {err ? <span className="u-bad"> · {err}</span> : null}
    </div>
  );
}
