import { useEffect, useRef, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'

const isIOS     = /iPad|iPhone|iPod/.test(navigator.userAgent)
const isAndroid = /Android/.test(navigator.userAgent)

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
  const [showInstallSteps, setShowInstallSteps] = useState(false)
  const [canGoBack, setCanGoBack] = useState(false)

  // Tell the SW which event URL to open when a notification is tapped.
  // Uses controller directly (rather than .ready) so the message goes to the SW
  // that's actually controlling this page. Also re-sends on controllerchange so
  // a freshly-activated SW (via clients.claim()) gets the URL without a reload.
  useEffect(() => {
    if (!('serviceWorker' in navigator)) return
    const payload = { type: 'STORE_EVENT_URL', url: window.location.pathname }
    const send = () => navigator.serviceWorker.controller?.postMessage(payload)
    send()
    navigator.serviceWorker.addEventListener('controllerchange', send)
    return () => navigator.serviceWorker.removeEventListener('controllerchange', send)
  }, [])

  useEffect(() => {
    if (!event) return

    document.title = event.name

    let link = document.querySelector("link[rel='manifest']")
    if (!link) {
      link = document.createElement('link')
      link.rel = 'manifest'
      document.head.appendChild(link)
    }
    link.href = `/manifest/${eventId}`
  }, [event, eventId])

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
        const savedToken = localStorage.getItem('pager_attendee_token')
        const savedChanRaw = localStorage.getItem(`pager_channels_${eventId}`)
        if(savedToken && savedChanRaw && Notification.permission === 'granted'){
          try {
            const vRes = await fetch('/attendee/verify', {
              headers: {'X-Attendee-Token': savedToken},
            })
            if(vRes.ok){
              const savedIds = JSON.parse(savedChanRaw)

              for(const chanId of savedIds){
                await init(chanId)
              }

              await loadMessageHistory(savedIds, savedToken)
              setSubState('subscribed')
              setView('subscribed')
              return
            }
          } catch {
            //network error. fall through to landing
          }
          //If the token is rejected or the request fails clear the stale state
          localStorage.removeItem('pager_attendee_token')
          localStorage.removeItem(`pager_channels_${eventId}`)
        }
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

  async function loadMessageHistory(channelIds, token) {
    const results = await Promise.all(
      channelIds.map(chanId =>
        fetch(`/attendee/channel/${chanId}/messages`, {
          headers: { 'X-Attendee-Token': token },
        }).then(r => (r.ok ? r.json() : []))
      )
    )
    setMessages(results.flat().map(msg => ({
      ...msg,
      time: msg.sent_at ? new Date(msg.sent_at) : undefined,
    })))
  }

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
    setSubState(existing ? 'subscribed' : 'idle')
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
        localStorage.setItem(`pager_channels_${eventId}`, JSON.stringify([...selectedIds]))
      }

      // Open SSE connection per selected channel using the existing init() function
      for (const chanId of ids) {
        await init(chanId)
      }

      // Fetch message history for all subscribed channels and populate the feed
      if (token) {
        await loadMessageHistory(ids, token)
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
      <div className="min-h-dvh bg-[#0f0f0f] text-[#f0f0f0] flex flex-col items-center px-5 pt-16 pb-12 font-sans">
        {/* Confirmation header */}
        <div className="flex flex-col items-center text-center gap-3 mb-8">
          <div className="text-5xl">🔔</div>
          <h2 className="text-2xl font-bold">You're all set</h2>
          <p className="text-[#aaa] text-base max-w-xs leading-relaxed">
            You'll receive updates directly on your lock screen. No need to keep this page open.
          </p>
          <p className="text-xs text-[#555]">{status}</p>
        </div>

        {/* Message feed */}
        {messages.length > 0 && (
          <div className="w-full max-w-md">
            <p className="text-xs text-[#555] font-medium uppercase tracking-widest mb-1">Messages</p>
            <div className="flex flex-col divide-y divide-[#1e1e1e]">
              {messages.map((msg, i) => (
                <div key={i} className="py-3">
                  <p className="text-[0.95rem] leading-relaxed text-[#e0e0e0]">{msg.text}</p>
                  {msg.time && (
                    <time className="block text-xs text-[#555] mt-1">
                      {msg.time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </time>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        <button
          onClick={() => { setCanGoBack(true); setView('landing') }}
          className="text-sm text-[#555] mt-8 underline underline-offset-4 cursor-pointer hover:text-[#888] transition-colors"
        >
          Manage subscriptions
        </button>
      </div>
    )
  }

  // ── View 1: Landing / channel selection ──────────────────────────────────

  return (
    <div className="min-h-dvh bg-[#0f0f0f] text-[#f0f0f0] flex flex-col px-5 py-10 gap-6 font-sans max-w-lg mx-auto">
      {/* Back button — only shown when navigating from the messages view */}
      {canGoBack && (
        <button
          onClick={() => setView('subscribed')}
          className="self-start text-sm text-[#555] underline underline-offset-4 cursor-pointer hover:text-[#888] transition-colors"
        >
          ← Back to messages
        </button>
      )}

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
          {channels.map(ch => {
            const checked = selectedIds.has(ch.id)
            return (
              <label
                key={ch.id}
                className={`flex items-start gap-4 rounded-2xl px-4 py-4 cursor-pointer border transition-colors ${
                  checked
                    ? 'bg-[#1a1a1a] border-[#3a3a3a]'
                    : 'bg-[#141414] border-[#222]'
                }`}
              >
                {/* Custom checkbox */}
                <div className={`mt-0.5 w-5 h-5 shrink-0 rounded-md border-2 flex items-center justify-center transition-colors ${
                  checked ? 'bg-brand border-brand' : 'border-[#444] bg-transparent'
                }`}>
                  {checked && (
                    <svg className="w-3 h-3 text-white" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth={2.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M2 6l3 3 5-5" />
                    </svg>
                  )}
                </div>
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={() => toggleChannel(ch.id)}
                  className="sr-only"
                />
                <div>
                  <p className="font-medium text-[0.95rem] leading-snug">{ch.name}</p>
                  {ch.description && (
                    <p className="text-sm text-[#888] mt-1 leading-snug">{ch.description}</p>
                  )}
                </div>
              </label>
            )
          })}
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
