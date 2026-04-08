import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

export default function Landing() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  async function handleCreate() {
    setLoading(true)
    setError('')

    try {
      const res = await fetch('/channel', { method: 'POST' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const { id, organizer_token } = await res.json()
      navigate(`/organizer?c=${id}&t=${organizer_token}`)
    } catch (err) {
      setError('Failed to create event. Please try again.')
      setLoading(false)
    }
  }

  return (
    <div className="min-h-dvh bg-[#0a0a0a] text-[#f0f0f0] flex flex-col items-center justify-center px-6 gap-10">
      <div className="flex flex-col items-center gap-6 text-center max-w-lg">
        <div className="flex items-center gap-2">
          <span className="text-5xl font-black tracking-tighter">
            Pa<span className="text-brand">g</span>er
          </span>
        </div>

        <p className="text-xl text-[#999] leading-snug font-light max-w-sm">
          Instant broadcast notifications for live events.
          <br />
          No accounts. No noise. Just signal.
        </p>
      </div>

      <div className="flex flex-col items-center gap-3">
        <button
          onClick={handleCreate}
          disabled={loading}
          className="relative bg-brand text-white font-bold text-lg rounded-2xl px-10 py-4 cursor-pointer transition-all duration-150 hover:opacity-90 hover:scale-[1.02] active:scale-[0.98] disabled:opacity-50 disabled:cursor-default disabled:scale-100 shadow-[0_0_40px_rgba(108,71,255,0.35)]"
        >
          {loading ? (
            <span className="flex items-center gap-2">
              <Spinner />
              Creating…
            </span>
          ) : (
            'Create Event'
          )}
        </button>

        {error && (
          <p className="text-red-500 text-sm">{error}</p>
        )}
      </div>

      <p className="absolute bottom-8 text-xs text-[#444]">
        Channels expire after 72 hours.
      </p>
    </div>
  )
}

function Spinner() {
  return (
    <svg
      className="animate-spin h-4 w-4"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        className="opacity-25"
        cx="12" cy="12" r="10"
        stroke="currentColor" strokeWidth="4"
      />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
      />
    </svg>
  )
}
