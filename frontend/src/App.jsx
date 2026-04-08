import { Routes, Route } from 'react-router-dom'
import Landing from './pages/Landing.jsx'
import Attendee from './pages/Attendee.jsx'
import Organizer from './pages/Organizer.jsx'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Landing />} />
      <Route path="/channel/:id" element={<Attendee />} />
      <Route path="/organizer" element={<Organizer />} />
    </Routes>
  )
}
