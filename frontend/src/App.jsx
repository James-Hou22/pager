import { Routes, Route } from 'react-router-dom'
import Auth from './pages/Auth.jsx'
import Dashboard from './pages/Dashboard.jsx'
import EventDetail from './pages/EventDetail.jsx'
import Attendee from './pages/Attendee.jsx'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Auth />} />
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/events/:eventId" element={<EventDetail />} />
      <Route path="/channel/:id" element={<Attendee />} />
    </Routes>
  )
}
