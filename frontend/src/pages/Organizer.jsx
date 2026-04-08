import { useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { QRCodeSVG } from 'qrcode.react'

export default function Organizer() {
  const [searchParams] = useSearchParams()
  const channelId = searchParams.get('c')
  const token = searchParams.get('t')

  const [error, setError] = useState('')
  const [ready, setReady] = useState(false)
  const [attendeeUrl, setAttendeeUrl] = useState('')
  const [message, setMessage] = useState('')
  const [blastStatus, setBlastStatus] = useState({ text: '', type: '' })
  const [sending, setSending] = useState(false)
  const textareaRef = useRef(null)

  useEffect(() => {
    if (!channelId || !token) {
      setError('Missing channel ID or organizer token in URL.')
      return
    }
    init()
  }, [channelId, token])

  async function init() {
    const res = await fetch(`/channel/${channelId}`)
    if (!res.ok) {
      setError('Channel not found.')
      return
    }
    setAttendeeUrl(`${location.origin}/channel/${channelId}`)
    setReady(true)
  }

  async function handleBlast() {
    const text = message.trim()
    if (!text) return

    setSending(true)
    setBlastStatus({ text: 'Sending…', type: '' })

    try {
      const res = await fetch(`/channel/${channelId}/blast`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Organizer-Token': token,
        },
        body: JSON.stringify({ message: text }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.error || `HTTP ${res.status}`)
      }

      setBlastStatus({ text: 'Sent ✓', type: 'ok' })
      setMessage('')
    } catch (err) {
      setBlastStatus({ text: `Failed: ${err.message}`, type: 'err' })
    } finally {
      setSending(false)
    }
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleBlast()
  }

  const statusColor = {
    ok: 'text-green-500',
    err: 'text-red-500',
    '': 'text-[#888]',
  }[blastStatus.type]

  return (
    <div className="min-h-dvh bg-[#0f0f0f] text-[#f0f0f0] flex flex-col items-center px-4 py-8 gap-6 font-sans">
      <h1 className="text-3xl font-bold tracking-tight">
        Pa<span className="text-brand">g</span>er{' '}
        <small className="text-[0.55em] text-[#666]">organizer</small>
      </h1>

      {error && (
        <p className="text-red-500 text-sm text-center">{error}</p>
      )}

      {ready && (
        <>
          <div className="w-full max-w-md bg-surface border border-border rounded-2xl p-5 flex flex-col gap-4">
            <h2 className="text-xs font-semibold text-[#aaa] uppercase tracking-widest">
              Attendee QR code
            </h2>
            <div className="flex justify-center">
              <QRCodeSVG
                value={attendeeUrl}
                size={220}
                bgColor="#1a1a1a"
                fgColor="#f0f0f0"
                className="rounded-lg"
              />
            </div>
            <p className="text-xs text-[#666] break-all text-center">{attendeeUrl}</p>
          </div>

          <div className="w-full max-w-md bg-surface border border-border rounded-2xl p-5 flex flex-col gap-4">
            <h2 className="text-xs font-semibold text-[#aaa] uppercase tracking-widest">
              Blast a message
            </h2>
            <textarea
              ref={textareaRef}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Doors open at 7 pm — grab a seat!"
              className="w-full bg-[#111] border border-[#333] rounded-lg text-[#f0f0f0] font-sans text-base px-3 py-3 resize-y min-h-20 outline-none focus:border-brand transition-colors"
            />
            <button
              onClick={handleBlast}
              disabled={sending}
              className="w-full bg-brand text-white rounded-xl py-3 text-base font-semibold transition-opacity hover:opacity-85 disabled:opacity-40 disabled:cursor-default cursor-pointer"
            >
              Send to everyone
            </button>
            {blastStatus.text && (
              <p className={`text-sm text-center ${statusColor}`}>
                {blastStatus.text}
              </p>
            )}
          </div>
        </>
      )}
    </div>
  )
}
