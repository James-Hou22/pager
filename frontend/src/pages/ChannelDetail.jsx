import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useParams, Link } from 'react-router-dom'
import { format } from 'date-fns'
import { apiFetch } from '../lib/api.js'
import { Button } from '../components/ui/button.jsx'
import { Label } from '../components/ui/label.jsx'
import { Textarea } from '../components/ui/textarea.jsx'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../components/ui/dialog.jsx'

const CHANNEL_STATUS_STYLES = {
  inactive: 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400',
  active:   'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400',
  closed:   'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400',
}

export default function ChannelDetail() {
  const navigate = useNavigate()
  const { eventId, channelId } = useParams()
  const [channel, setChannel] = useState(null)
  const [loading, setLoading] = useState(true)
  const [statusUpdating, setStatusUpdating] = useState(false)
  const [statusError, setStatusError] = useState('')

  // Message history state
  // TODO: replace with real data once GET /events/:eventId/channels/:channelId/messages exists
  const [messages, setMessages] = useState([])

  // Send message dialog
  const [dialogOpen, setDialogOpen] = useState(false)
  const [messageBody, setMessageBody] = useState('')
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState('')

  // TODO: no single-channel endpoint exists yet — fetching all channels and filtering by channelId
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
      // TODO: PATCH /events/:eventId/channels/:channelId/status does not exist yet
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

  function openSendDialog() {
    setMessageBody('')
    setSendError('')
    setDialogOpen(true)
  }

  function handleDialogChange(open) {
    if (!open) {
      setMessageBody('')
      setSendError('')
    }
    setDialogOpen(open)
  }

  async function handleSendMessage(e) {
    e.preventDefault()
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
      setDialogOpen(false)
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
  const isClosed   = channel.Status === 'closed'

  return (
    <div className="min-h-dvh flex flex-col bg-background">
      <header className="border-b px-4 h-14 flex items-center">
        <Link
          to={`/events/${eventId}`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          ← Back to Event
        </Link>
      </header>

      <main className="flex-1 px-4 py-6 w-full max-w-2xl mx-auto">
        {/* Channel header */}
        <div className="flex flex-col gap-2 mb-8">
          <h1 className="text-2xl font-semibold">{channel.Name}</h1>
          <div>
            <span className={`inline-block text-xs font-medium px-2 py-1 capitalize ${CHANNEL_STATUS_STYLES[channel.Status] ?? CHANNEL_STATUS_STYLES.inactive}`}>
              {channel.Status}
            </span>
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex flex-col gap-3 mb-8">
          <Button
            className="rounded-none h-11 w-full"
            disabled={isActive || statusUpdating}
            onClick={() => updateStatus('active')}
          >
            Open Channel
          </Button>
          <Button
            variant="outline"
            className="rounded-none h-11 w-full"
            disabled={isClosed || statusUpdating}
            onClick={() => updateStatus('closed')}
          >
            Close Channel
          </Button>
          <Button
            variant="outline"
            className="rounded-none h-11 w-full"
            onClick={openSendDialog}
          >
            Send Message
          </Button>
          {statusError && <p className="text-sm text-destructive">{statusError}</p>}
        </div>

        {/* Message history */}
        <h2 className="text-base font-semibold mb-4">Messages</h2>

        {messages.length === 0 ? (
          <p className="text-muted-foreground text-sm py-8 text-center">
            No messages sent yet.
          </p>
        ) : (
          <div className="flex flex-col">
            {messages.map((msg, i) => (
              <div key={i} className="border-b py-4 first:border-t">
                <p className="text-sm">{msg.Body}</p>
                <p className="text-xs text-muted-foreground mt-1">
                  {format(new Date(msg.SentAt), 'MMM d, yyyy \'at\' h:mm a')}
                </p>
              </div>
            ))}
          </div>
        )}
      </main>

      {/* Send message dialog */}
      <Dialog open={dialogOpen} onOpenChange={handleDialogChange}>
        <DialogContent className="rounded-none max-w-sm w-[calc(100vw-2rem)]">
          <DialogHeader>
            <DialogTitle>Send message</DialogTitle>
          </DialogHeader>

          <form onSubmit={handleSendMessage} className="flex flex-col gap-4 mt-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="message-body">Message</Label>
              <Textarea
                id="message-body"
                placeholder="Type your message…"
                className="min-h-30"
                value={messageBody}
                onChange={e => setMessageBody(e.target.value)}
                required
              />
            </div>

            {sendError && <p className="text-sm text-destructive">{sendError}</p>}

            <Button
              type="submit"
              disabled={sending || !messageBody.trim()}
              className="rounded-none h-11 w-full"
            >
              {sending ? 'Sending…' : 'Send'}
            </Button>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
