import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import './attendee.css'
import Attendee from './pages/Attendee.jsx'

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/channel/:id" element={<Attendee />} />
      </Routes>
    </BrowserRouter>
  </StrictMode>,
)
