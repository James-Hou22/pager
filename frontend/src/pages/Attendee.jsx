import { useEffect, useRef, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'

function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(base64)
  return Uint8Array.from([...raw].map((c) => c.charCodeAt(0)))
}

export default function Attendee() {
  const { id: paramId } = useParams()
  const [searchParams] = useSearchParams()
  const channelId = paramId || searchParams.get('c')

  const [status, setStatus] = useState('Loading…')
  const [messages, setMessages] = useState([])
  const [subState, setSubState] = useState('idle') // idle | subscribed | unsupported
  const regRef = useRef(null)

  useEffect(() => {
    if (!channelId) {
      setStatus('No channel ID found in URL.')
      return
    }
    init()
  }, [channelId])

  async function init() {
    setStatus('Connecting…')

    const sse = new EventSource(`/channel/${channelId}/sse`)
    sse.addEventListener('message', (e) => {
      setMessages((prev) => [{ text: e.data, time: new Date() }, ...prev])
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
      await sendSubscriptionToServer(existing)
      setSubState('subscribed')
    } else {
      setSubState('idle')
    }
  }

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

      await sendSubscriptionToServer(sub)
      setSubState('subscribed')
    } catch (err) {
      console.error('Subscribe failed:', err)
      setSubState('error')
    }
  }

  async function sendSubscriptionToServer(sub) {
    await fetch(`/channel/${channelId}/sub`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(sub),
    })
  }

  const buttonLabel = {
    idle: 'Enable notifications',
    loading: 'Subscribing…',
    subscribed: 'Notifications enabled ✓',
    unsupported: 'Notifications not supported',
    error: 'Subscribe failed — try again',
  }[subState]

  const buttonDisabled = subState === 'subscribed' || subState === 'unsupported' || subState === 'loading'

  return (
    <div className="min-h-dvh bg-[#0f0f0f] text-[#f0f0f0] flex flex-col items-center px-4 py-8 gap-6 font-sans">
      <h1 className="text-3xl font-bold tracking-tight">
        Pa<span className="text-brand">g</span>er
      </h1>

      <p className="text-sm text-[#888] text-center">{status}</p>

      {subState !== 'unsupported' && (
        <button
          onClick={handleSubscribe}
          disabled={buttonDisabled}
          className="bg-brand text-white rounded-xl px-8 py-3 text-base font-semibold transition-opacity hover:opacity-85 disabled:opacity-40 disabled:cursor-default cursor-pointer"
        >
          {buttonLabel}
        </button>
      )}

      <div className="w-full max-w-md flex flex-col gap-3">
        {messages.length === 0 && (
          <p className="text-[#555] text-sm">No messages yet.</p>
        )}
        {messages.map((msg, i) => (
          <div
            key={i}
            className="bg-surface border border-border rounded-xl px-4 py-3 text-[0.95rem] leading-relaxed animate-[slide-in_0.2s_ease]"
          >
            {msg.text}
            <time className="block text-xs text-[#666] mt-1">
              {msg.time.toLocaleTimeString()}
            </time>
          </div>
        ))}
      </div>

      <style>{`
        @keyframes slide-in {
          from { opacity: 0; transform: translateY(-6px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  )
}
