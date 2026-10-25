// main.tsx —— TRX 页 React 壳（T-0113：左栏导航 nox shell 同款+TrxPage 全件）。
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import TrxPage from './pages/trx'
import { Shell } from './Shell'
import './styles/00-base.css'
import './styles/03-table-chip.css'
import './styles/06-buttons.css'
import './styles/08-facts-ledger.css'
import './styles/11-account-responsive.css'
import './styles/12-nox-shell.css'
import './styles/13-daily.css'
import './styles/18-price-alert.css'
import './styles/20-semantic.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Shell><TrxPage /></Shell>
  </StrictMode>,
)
