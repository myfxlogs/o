// Shell.tsx —— O 左侧栏导航壳（T-0113：nox 12-nox-shell 同款 DOM——
// 门面组=总览+TRX 自营盘，运维组预留空壳接号不加组；active 态按 path 显形）。
import type { ReactNode } from 'react'
import { useEffect, useState } from 'react'

const ICONS: Record<string, string> = {
  '/': 'M4 4h16v16H4z',
  '/trx': 'M12 2v20M2 12h20',
}

function ShellClock() {
  const [s, setS] = useState('')
  useEffect(() => {
    const f = () => setS(new Date().toISOString().slice(11, 19) + ' UTC')
    f()
    const t = setInterval(f, 1000)
    return () => clearInterval(t)
  }, [])
  return <span className="shell-clock">{s}</span>
}

export function Shell({ children }: { children: ReactNode }) {
  const [sideOpen, setSideOpen] = useState(false)
  const path = location.pathname === '/trx' || location.pathname === '/trx.html' ? '/trx' : '/'
  const groups: [string, [string, string][]][] = [
    ['门面', [['/', '总览'], ['/trx', 'TRX 自营盘']]],
    ['运维', []],
  ]
  return (
    <main className="app-layout">
      <header className="app">
        <button className="sidebar-toggle" onClick={() => setSideOpen(o => !o)} aria-label="菜单">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M3 6h18M3 12h18M3 18h18" /></svg>
        </button>
        <span className="brand">O</span>
        <span className="app-spacer" />
        <ShellClock />
      </header>
      <nav className={`sidebar${sideOpen ? ' open' : ''}`} aria-label="视图切换">
        {groups.map(([label, routes]) => (
          <div key={label}>
            <div className="grp-label">{label}</div>
            {routes.map(([p, title]) => (
              <a key={p} href={p === '/' ? '/' : p + '.html'}
                className={p === path ? 'sidebar-item active' : 'sidebar-item'}
                onClick={() => setSideOpen(false)}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d={ICONS[p] ?? 'M4 4h16v16H4z'} />
                </svg>
                <span>{title}</span>
              </a>
            ))}
          </div>
        ))}
      </nav>
      {sideOpen && <div className="sidebar-overlay" onClick={() => setSideOpen(false)} />}
      <div className="app-content">{children}</div>
    </main>
  )
}
