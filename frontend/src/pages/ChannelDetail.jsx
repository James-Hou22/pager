import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useParams, Link } from 'react-router-dom'
import { format } from 'date-fns'
import { apiFetch } from '../lib/api.js'
import { Button } from '../components/ui/button.jsx'
import { Textarea } from '../components/ui/textarea.jsx'

const CHANNEL_STATUS_STYLES = {
  inactive: 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400',
  active:   'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400',
  closed:   'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400',
}

const STATUS_HINT = {
  inactive: 'Open this channel to start broadcasting.',
  active:   'Channel is live — broadcast messages to subscribers.',
  closed:   'This channel is closed. No further messages can be sent.',
}

export default function ChannelDetail() {
  const navigate = useNavigate()
  const { eventId, channelId } = useParams()
  const [channel, setChannel] = useState(null)
  const [loading, setLoading] = useState(true)
  const [statusUpdating, setStatusUpdating] = useState(false)
  const [statusError, setStatusError] = useState('')
  const [messages, setMessages] = useState([])
  const [messageBody, setMessageBody] = useState('')
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState('')

  const fetchChannel = useCallback(async () => {
    const res = await apiFetch(`/events/${eventId}/channels`)
    if (res.ok) {
      const channels = await res.json()
      const found = channels.find(ch => ch.ID === channelId)
      setChannel(found ?? null)
    }
  }, [eventId, channelId])

  const fetchMessages = useCallback(async () => {
    const res = await apiFetch(`/events/${eventId}/channels/${channelId}/messages`)
    if (res.ok) setMessages(await res.json())
  }, [eventId, channelId])

  useEffect(() => {
    if (!localStorage.getItem('pager_token')) {
      navigate('/', { replace: true })
      return
    }

    async function init() {
      const res = await apiFetch(`/events/${eventId}/channels`)
      if (res.status === 401) {
        localStorage.removeItem('pager_token')
        navigate('/', { replace: true })
        return
      }
      if (res.ok) {
        const channels = await res.json()
        const found = channels.find(ch => ch.ID === channelId)
        setChannel(found ?? null)
      }

      const msgRes = await apiFetch(`/events/${eventId}/channels/${channelId}/messages`)
      if (msgRes.ok) setMessages(await msgRes.json())

      setLoading(false)
    }

    init()
  }, [eventId, channelId, navigate])

  async function updateStatus(status) {
    setStatusError('')
    setStatusUpdating(true)
    try {
      const res = await apiFetch(`/events/${eventId}/channels/${channelId}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      })
      if (!res.ok) {
        const json = await res.json().catch(() => ({}))
        setStatusError(json.error || 'Failed to update status.')
        return
      }
      await fetchChannel()
    } catch {
      setStatusError('Could not reach the server.')
    } finally {
      setStatusUpdating(false)
    }
  }

  async function handleSend() {
    if (!messageBody.trim()) return
    setSendError('')
    setSending(true)
    try {
      const res = await apiFetch(`/channel/${channelId}/blast`, {
        method: 'POST',
        body: JSON.stringify({ message: messageBody.trim() }),
      })
      if (!res.ok) {
        const json = await res.json().catch(() => ({}))
        setSendError(json.error || 'Failed to send message.')
        return
      }
      setMessageBody('')
      await fetchMessages()
    } catch {
      setSendError('Could not reach the server.')
    } finally {
      setSending(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-dvh flex items-center justify-center">
        <p className="text-muted-foreground text-sm">Loading…</p>
      </div>
    )
  }

  if (!channel) {
    return (
      <div className="min-h-dvh flex items-center justify-center px-4">
        <div className="text-center">
          <p className="text-muted-foreground text-sm mb-4">Channel not found.</p>
          <Link to={`/events/${eventId}`} className="text-sm underline underline-offset-4">
            Back to Event
          </Link>
        </div>
      </div>
    )
  }

  const isInactive = channel.Status === 'inactive'
  const isActive   = channel.Status === 'active'

  return (
    <div className="h-dvh flex flex-col bg-background overflow-hidden">
      <header className="shrink-0 border-b px-4 h-14 flex items-center">
        <Link
          to={`/events/${eventId}`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          ← Back to Event
        </Link>
      </header>

      <div className="flex-1 flex flex-col overflow-hidden w-full max-w-2xl mx-auto">

        {/* Sticky top zone: channel info + compose */}
        <div className="shrink-0 px-4 pt-6 pb-4 border-b">

          {/* Channel header */}
          <div className="mb-6">
            <div className="flex items-start justify-between gap-4 mb-2">
              <h1 className="text-2xl font-semibold leading-tight">{channel.Name}</h1>
              <span className={`shrink-0 mt-1 inline-block text-xs font-medium px-2 py-1 capitalize ${CHANNEL_STATUS_STYLES[channel.Status] ?? CHANNEL_STATUS_STYLES.inactive}`}>
                {channel.Status}
              </span>
            </div>

            <p className="text-sm text-muted-foreground mb-4">
              {STATUS_HINT[channel.Status]}
            </p>

            {/* Status control — demoted, context-sensitive */}
            <div className="flex items-center gap-3">
              {isInactive && (
                <Button
                  variant="outline"
                  size="sm"
                  className="rounded-none"
                  disabled={statusUpdating}
                  onClick={() => updateStatus('active')}
                >
                  {statusUpdating ? 'Opening…' : 'Open Channel'}
                </Button>
              )}
              {isActive && (
                <Button
                  variant="outline"
                  size="sm"
                  className="rounded-none"
                  disabled={statusUpdating}
                  onClick={() => updateStatus('closed')}
                >
                  {statusUpdating ? 'Closing…' : 'Close Channel'}
                </Button>
              )}
              {statusError && <p className="text-sm text-destructive">{statusError}</p>}
            </div>
          </div>

          {/* Compose — primary action */}
          <div className="flex flex-col gap-3">
            <Textarea
              placeholder={isActive ? 'Type a message to broadcast…' : 'Open the channel to send messages'}
              className="min-h-28"
              value={messageBody}
              onChange={e => setMessageBody(e.target.value)}
              disabled={!isActive || sending}
              onKeyDown={e => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleSend()
              }}
            />
            {sendError && <p className="text-sm text-destructive">{sendError}</p>}
            <div className="flex items-center justify-between gap-4">
              <Button
                className="rounded-none h-11 px-8"
                disabled={sending || !messageBody.trim() || !isActive}
                onClick={handleSend}
              >
                {sending ? 'Sending…' : 'Broadcast'}
              </Button>
            </div>
          </div>

        </div>

        {/* Broadcast log label — fixed, does not scroll */}
        <div className="shrink-0 px-4 pt-4 pb-2">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
            Broadcast Log
          </p>
        </div>

        {/* Scrollable broadcast log — no scrollbar */}
        <div className="flex-1 overflow-y-auto px-4 pb-4 [&::-webkit-scrollbar]:hidden [scrollbar-width:none]">
          <div className="border">
            {messages.length === 0 ? (
              <p className="text-muted-foreground text-sm py-8 text-center">
                No messages sent yet.
              </p>
            ) : (
              messages.map((msg, i) => (
                <div key={i} className="px-4 py-3 border-b last:border-b-0">
                  <p className="text-sm">{msg.Body}</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {format(new Date(msg.SentAt), 'MMM d \'at\' h:mm a')}
                  </p>
                </div>
              ))
            )}
          </div>
        </div>

      </div>
    </div>
  )
}
