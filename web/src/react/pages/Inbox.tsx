// Inbox.tsx —— 投件站(业主令 11-12:文件传前端落到可读盒,会话来取)。
// 上传 POST /inbox → 落 O_INBOX_DIR;清单 GET /inbox,取件 /inbox/f/<名>。
import { useCallback, useEffect, useState } from "react";

type InboxFile = { name: string; size: number; mtime: string };

const fmtSize = (n: number) =>
  n >= 1 << 20 ? (n / (1 << 20)).toFixed(1) + "MB" : n >= 1024 ? (n / 1024).toFixed(0) + "K" : n + "B";

export function InboxPage() {
  const [files, setFiles] = useState<InboxFile[]>([]);
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = await fetch("/inbox");
      const j = await r.json();
      setFiles(j.files ?? []);
    } catch { /* 静默——清单空 */ }
  }, []);
  useEffect(() => { load(); }, [load]);

  const up = async (f: File) => {
    setBusy(true); setMsg("");
    try {
      const fd = new FormData(); fd.append("file", f);
      const r = await fetch("/inbox", { method: "POST", body: fd });
      const j = await r.json();
      setMsg(r.ok ? `已落 ${j.path}` : `败 ${j.err ?? r.status}`);
      load();
    } catch { setMsg("败 网络"); }
    setBusy(false);
  };

  return (
    <div className="page v2">
      <section className="v2-hero">
        <div className="v2-eyebrow"><span className="v2-livedot" />递东西</div>
        <h1 className="v2-title">投件站</h1>
        <p className="v2-sub">文件落本机 inbox——会话直接读,别再手拷部署树</p>
      </section>

      <div className="v2-cols">
        <section className="v2-card">
          <h2>投件</h2>
          <label className={`inbox-drop${busy ? " busy" : ""}`}>
            <input type="file" style={{ display: "none" }} disabled={busy}
              onChange={(e) => { const f = e.target.files?.[0]; if (f) up(f); e.target.value = ""; }} />
            {busy ? "投件中…" : "选文件投——64MB 内,落 /opt/o-inbox/"}
          </label>
          {msg && <p className="inbox-msg">{msg}</p>}
        </section>

        <section className="v2-card">
          <h2>落件清单 <small>{files.length}</small></h2>
          {files.length === 0 && <p className="dim">空仓——还没投过件</p>}
          {files.map((f) => (
            <div key={f.name} className="inbox-row">
              <a href={`/inbox/f/${encodeURIComponent(f.name)}`} target="_blank" rel="noreferrer">{f.name}</a>
              <span className="dim">{fmtSize(f.size)} · {f.mtime}</span>
            </div>
          ))}
        </section>
      </div>
    </div>
  );
}
