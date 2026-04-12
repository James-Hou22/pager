import { useEffect, useRef, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'

function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(base64)
  return Uint8Array.from([...raw].map((c) => c.charCodeAt(0)))
}

export default function Attendee() {
  const { id } = useParams()
  const [searchParams] = useSearchParams()
  const eventId = id || searchParams.get('c')

  // Existing state
  const [status, setStatus] = useState('Loading…')
  const [messages, setMessages] = useState([])
  const [subState, setSubState] = useState('idle') // idle | loading | subscribed | unsupported | error
  const regRef = useRef(null)

  // New state for two-view flow
  const [view, setView] = useState('landing')       // 'landing' | 'subscribed'
  const [event, setEvent] = useState(null)
  const [channels, setChannels] = useState([])
  const [selectedIds, setSelectedIds] = useState(new Set())
  const [loadError, setLoadError] = useState('')
  const [subError, setSubError] = useState('')

  useEffect(() => {
    if (!eventId) {
      setLoadError('No event ID found in URL.')
      return
    }

    async function loadEvent() {
      try {
        const [evRes, chRes] = await Promise.all([
          fetch(`/attendee/events/${eventId}`),
          fetch(`/attendee/events/${eventId}/channels`),
        ])
        if (!evRes.ok) {
          setLoadError('Event not found.')
          return
        }
        const ev = await evRes.json()
        const chs = chRes.ok ? await chRes.json() : []
        setEvent(ev)
        setChannels(chs)
        setSelectedIds(new Set(chs.map(ch => ch.id)))
      } catch {
        setLoadError('Could not load event details.')
      }

      // Register service worker early so regRef is ready before subscription
      if ('serviceWorker' in navigator && 'PushManager' in window) {
        try {
          const reg = await navigator.serviceWorker.register('/sw.js')
          regRef.current = reg
        } catch (err) {
          console.error('SW registration failed:', err)
        }
      }
    }

    loadEvent()
  }, [eventId])

  // Adapted to accept channelId as a parameter instead of reading from closure.
  // All other behaviour is unchanged.
  async function init(channelId) {
    setStatus('Connecting…')

    const sse = new EventSource(`/channel/${channelId}/sse`)
    sse.addEventListener('message', (e) => {
      setMessages((prev) => [...prev, { text: e.data, channel_id: channelId, time: new Date() }])
    })
    sse.addEventListener('open', () => setStatus('Connected.'))
    sse.addEventListener('error', () => setStatus('Connection lost — retrying…'))

    if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
      setStatus('Connected. (Push notifications not supported in this browser.)')
      setSubState('unsupported')
      return
    }

    const reg = await navigator.serviceWorker.register('/sw.js')
    regRef.current = reg

    const existing = await reg.pushManager.getSubscription()
    if (existing) {
      await sendSubscriptionToServer(existing, channelId)
      setSubState('subscribed')
    } else {
      setSubState('idle')
    }
  }

  // Kept for compatibility — not called by the new UI but preserved.
  // eslint-disable-next-line no-unused-vars
  async function handleSubscribe() {
    const reg = regRef.current
    if (!reg) return
    setSubState('loading')

    try {
      const permission = await Notification.requestPermission()
      if (permission !== 'granted') {
        setSubState('idle')
        return
      }

      const res = await fetch('/vapid-public-key')
      if (!res.ok) throw new Error('Could not fetch VAPID key')
      const { key } = await res.json()

      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(key),
      })

      await sendSubscriptionToServer(sub, null)
      setSubState('subscribed')
    } catch (err) {
      console.error('Subscribe failed:', err)
      setSubState('error')
    }
  }

  // Added channelId parameter; returns parsed JSON so the token can be read by callers.
  async function sendSubscriptionToServer(sub, channelId) {
    const res = await fetch(`/channel/${channelId}/sub`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(sub),
    })
    return res.json()
  }

  function toggleChannel(channelId) {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(channelId)) {
        next.delete(channelId)
      } else {
        next.add(channelId)
      }
      return next
    })
  }

  async function handleSubscribeAll() {
    setSubState('loading')
    setSubError('')

    try {
      // Ensure SW is registered before calling pushManager.subscribe
      if (!regRef.current) {
        if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
          setSubState('unsupported')
          return
        }
        regRef.current = await navigator.serviceWorker.register('/sw.js')
      }

      const permission = await Notification.requestPermission()
      if (permission !== 'granted') {
        setSubState('idle')
        return
      }

      const vapidRes = await fetch('/vapid-public-key')
      if (!vapidRes.ok) throw new Error('Could not fetch VAPID key')
      const { key } = await vapidRes.json()

      const sub = await regRef.current.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(key),
      })

      const ids = [...selectedIds]
      let token = null
      for (const chanId of ids) {
        const json = await sendSubscriptionToServer(sub, chanId)
        if (token === null && json?.token) token = json.token
      }

      if (token) {
        localStorage.setItem('pager_attendee_token', token)
      }

      // Open SSE connection per selected channel using the existing init() function
      for (const chanId of ids) {
        await init(chanId)
      }

      // Fetch message history for all subscribed channels and populate the feed
      if (token) {
        const historyResults = await Promise.all(
          ids.map(chanId =>
            fetch(`/attendee/channel/${chanId}/messages`, {
              headers: { 'X-Attendee-Token': token },
            }).then(r => (r.ok ? r.json() : []))
          )
        )
        setMessages(historyResults.flat())
      }

      setSubState('subscribed')
      setView('subscribed')
    } catch (err) {
      console.error('Subscribe failed:', err)
      setSubState('error')
      setSubError('Subscription failed. Please try again.')
    }
  }

  // ── Loading / error states ────────────────────────────────────────────────

  if (!event && !loadError) {
    return (
      <div className="min-h-dvh bg-[#0f0f0f] text-[#f0f0f0] flex items-center justify-center font-sans">
        <p className="text-[#888] text-sm">Loading…</p>
      </div>
    )
  }

  if (loadError) {
    return (
      <div className="min-h-dvh bg-[#0f0f0f] text-[#f0f0f0] flex items-center justify-center px-5 font-sans">
        <p className="text-red-400 text-sm text-center">{loadError}</p>
      </div>
    )
  }

  // ── View 2: Confirmation ──────────────────────────────────────────────────

  if (view === 'subscribed') {
    return (
      <div className="min-h-dvh bg-[#0f0f0f] text-[#f0f0f0] flex flex-col items-center justify-center px-5 gap-5 font-sans text-center">
        <div className="text-5xl">🔔</div>
        <h2 className="text-2xl font-bold">You're all set</h2>
        <p className="text-[#aaa] text-base max-w-xs leading-relaxed">
          You'll receive updates directly on your lock screen. No need to keep this page open.
        </p>
        <p className="text-xs text-[#555]">{status}</p>

        <button
          disabled
          className="text-sm text-[#555] cursor-default mt-2 underline underline-offset-4"
        >
          Manage subscriptions
        </button>

        {/* Message feed visible after subscribing */}
        {messages.length > 0 && (
          <div className="w-full max-w-md flex flex-col gap-3 mt-4">
            {messages.map((msg, i) => (
              <div
                key={i}
                className="bg-surface border border-border rounded-xl px-4 py-3 text-[0.95rem] leading-relaxed animate-[slide-in_0.2s_ease]"
              >
                {msg.text}
                {msg.time && (
                  <time className="block text-xs text-[#666] mt-1">
                    {msg.time.toLocaleTimeString()}
                  </time>
                )}
              </div>
            ))}
          </div>
        )}

        <style>{`
          @keyframes slide-in {
            from { opacity: 0; transform: translateY(-6px); }
            to   { opacity: 1; transform: translateY(0); }
          }
        `}</style>
      </div>
    )
  }

  // ── View 1: Landing / channel selection ──────────────────────────────────

  return (
    <div className="min-h-dvh bg-[#0f0f0f] text-[#f0f0f0] flex flex-col px-5 py-10 gap-6 font-sans max-w-lg mx-auto">
      {/* Brand */}
      <h1 className="text-2xl font-bold tracking-tight">
        Pa<span className="text-brand">g</span>er
      </h1>

      {/* Event header */}
      <div className="flex flex-col gap-2">
        <h2 className="text-2xl font-semibold leading-tight">{event.name}</h2>
        {event.welcome_description && (
          <p className="text-[#aaa] text-base leading-relaxed">{event.welcome_description}</p>
        )}
      </div>

      {/* Channel list */}
      {channels.length > 0 && (
        <div className="flex flex-col gap-3">
          <p className="text-xs text-[#888] font-medium uppercase tracking-widest">Channels</p>
          {channels.map(ch => (
            <label
              key={ch.id}
              className="flex items-start gap-3 bg-surface border border-border rounded-xl px-4 py-3 cursor-pointer"
            >
              <input
                type="checkbox"
                checked={selectedIds.has(ch.id)}
                onChange={() => toggleChannel(ch.id)}
                className="mt-0.5 accent-brand w-4 h-4 shrink-0"
              />
              <div>
                <p className="font-medium text-[0.95rem] leading-snug">{ch.name}</p>
                {ch.description && (
                  <p className="text-sm text-[#888] mt-0.5 leading-snug">{ch.description}</p>
                )}
              </div>
            </label>
          ))}
        </div>
      )}

      {subError && <p className="text-red-400 text-sm">{subError}</p>}

      <button
        onClick={handleSubscribeAll}
        disabled={selectedIds.size === 0 || subState === 'loading'}
        className="bg-brand text-white rounded-xl px-8 py-4 text-base font-semibold transition-opacity hover:opacity-85 disabled:opacity-40 disabled:cursor-default cursor-pointer"
      >
        {subState === 'loading' ? 'Subscribing…' : 'Subscribe and get updates'}
      </button>
    </div>
  )
}
