import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import './attendee.css'
import Attendee from './pages/Attendee.jsx'
import PwaGate from './components/PwaGate.jsx'

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/event/:id" element={<PwaGate><Attendee /></PwaGate>} />
      </Routes>
    </BrowserRouter>
  </StrictMode>,
)
