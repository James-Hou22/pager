import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate, useParams, Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { format } from 'date-fns'
import { QRCodeCanvas } from 'qrcode.react'
import { apiFetch } from '../lib/api.js'
import { Button } from '../components/ui/button.jsx'
import { Input } from '../components/ui/input.jsx'
import { Label } from '../components/ui/label.jsx'
import { Textarea } from '../components/ui/textarea.jsx'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../components/ui/dialog.jsx'

function formatEventTime(iso) {
  if (!iso) return <span className="text-muted-foreground">Not set</span>
  return format(new Date(iso), 'MMM d, yyyy \'at\' h:mm a')
}

const EVENT_STATUS_STYLES = {
  draft:  'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400',
  active: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400',
  closed: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400',
}

const CHANNEL_STATUS_STYLES = {
  inactive: 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400',
  active:   'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400',
  closed:   'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400',
}

const CHANNEL_STATUS_LABEL = {
  inactive: 'Draft',
  active:   'Active',
  closed:   'Closed',
}

export default function EventDetail() {
  const navigate = useNavigate()
  const { eventId } = useParams()
  const [event, setEvent] = useState(null)
  const [channels, setChannels] = useState([])
  const [loading, setLoading] = useState(true)
  const [qrOpen, setQrOpen] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [createError, setCreateError] = useState('')
  const qrCanvasRef = useRef(null)

  const attendeeURL = event ? `${window.location.origin}/event/${event.AccessCode}` : ''

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors, isSubmitting },
  } = useForm()

  const opensAt = watch('opensAt')

  const fetchChannels = useCallback(async () => {
    const res = await apiFetch(`/events/${eventId}/channels`)
    if (res.ok) setChannels(await res.json())
  }, [eventId])

  useEffect(() => {
    if (!localStorage.getItem('pager_token')) {
      navigate('/', { replace: true })
      return
    }

    async function init() {
      const res = await apiFetch('/events')
      if (res.status === 401) {
        localStorage.removeItem('pager_token')
        navigate('/', { replace: true })
        return
      }
      const events = await res.json()
      const found = events.find(e => e.ID === eventId)
      if (found) setEvent(found)
      await fetchChannels()
      setLoading(false)
    }

    init()
  }, [eventId, navigate, fetchChannels])

  function downloadQr() {
    const canvas = document.querySelector('#event-qr-canvas canvas')
    if (!canvas) return
    const url = canvas.toDataURL('image/png')
    const a = document.createElement('a')
    a.href = url
    a.download = `${event.Name}-qr.png`
    a.click()
  }

  function openDialog() {
    reset()
    setCreateError('')
    setDialogOpen(true)
  }

  function handleDialogChange(open) {
    if (!open) {
      reset()
      setCreateError('')
    }
    setDialogOpen(open)
  }

  async function onSubmit(data) {
    setCreateError('')

    const body = { name: data.name }
    if (data.description?.trim()) body.description = data.description.trim()
    if (data.opensAt) body.opens_at = new Date(data.opensAt).toISOString()
    if (data.closesAt) body.closes_at = new Date(data.closesAt).toISOString()

    const res = await apiFetch(`/events/${eventId}/channels`, {
      method: 'POST',
      body: JSON.stringify(body),
    })

    if (!res.ok) {
      const json = await res.json().catch(() => ({}))
      setCreateError(json.error || 'Failed to create channel.')
      return
    }

    setDialogOpen(false)
    reset()
    await fetchChannels()
  }

  if (loading) {
    return (
      <div className="min-h-dvh flex items-center justify-center">
        <p className="text-muted-foreground text-sm">Loading…</p>
      </div>
    )
  }

  if (!event) {
    return (
      <div className="min-h-dvh flex items-center justify-center px-4">
        <div className="text-center">
          <p className="text-muted-foreground text-sm mb-4">Event not found.</p>
          <Link to="/dashboard" className="text-sm underline underline-offset-4">
            Back to Dashboard
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-dvh flex flex-col bg-background">
      <header className="border-b px-4 h-14 flex items-center">
        <Link
          to="/dashboard"
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          ← Back to Dashboard
        </Link>
      </header>

      <main className="flex-1 w-full max-w-2xl mx-auto flex flex-col">

        {/* Event header */}
        <div className="px-4 pt-6 pb-5 border-b">
          <div className="flex items-start justify-between gap-4 mb-2">
            <h1 className="text-2xl font-semibold leading-tight">{event.Name}</h1>
            <span className={`shrink-0 mt-0.5 inline-block text-xs font-medium px-2 py-1 capitalize ${EVENT_STATUS_STYLES[event.Status] ?? EVENT_STATUS_STYLES.draft}`}>
              {event.Status}
            </span>
          </div>
          <div className="flex flex-col gap-0.5 text-sm text-muted-foreground">
            <span>Starts: {formatEventTime(event.StartsAt)}</span>
            <span>Ends: {formatEventTime(event.EndsAt)}</span>
          </div>
        </div>

        {/* QR code accordion */}
        <div className="border-b">
          <button
            type="button"
            onClick={() => setQrOpen(o => !o)}
            className="w-full flex items-center justify-between px-4 py-3 text-sm font-medium hover:bg-muted/50 transition-colors"
          >
            <span>Event QR Code</span>
            <svg
              className={`w-4 h-4 text-muted-foreground transition-transform ${qrOpen ? 'rotate-180' : ''}`}
              fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          <div
            className={`grid transition-all duration-300 ease-in-out ${qrOpen ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'}`}
          >
            <div className="overflow-hidden">
              <div className="border-t px-4 py-5 flex flex-col items-center gap-4">
                <div id="event-qr-canvas" ref={qrCanvasRef}>
                  <QRCodeCanvas value={attendeeURL} size={200} />
                </div>
                <Button
                  type="button"
                  variant="outline"
                  className="rounded-none h-9 w-full max-w-xs"
                  onClick={downloadQr}
                >
                  Download PNG
                </Button>
              </div>
            </div>
          </div>
        </div>

        {/* Channels section */}
        <div className="flex items-center px-4 py-3 border-b border-border/40">
          <h2 className="text-sm font-semibold">Channels</h2>
        </div>

        <div className="flex flex-col">
          {channels.map(ch => (
            <Link
              key={ch.ID}
              to={`/events/${eventId}/channels/${ch.ID}`}
              className="flex items-center justify-between px-4 py-4 border-b hover:bg-muted/50 transition-colors"
            >
              <span className="font-medium text-sm">{ch.Name}</span>
              <div className="flex items-center gap-2.5">
                <span className={`text-xs font-medium px-2 py-1 ${CHANNEL_STATUS_STYLES[ch.Status] ?? CHANNEL_STATUS_STYLES.inactive}`}>
                  {CHANNEL_STATUS_LABEL[ch.Status] ?? ch.Status}
                </span>
                <svg className="w-4 h-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                </svg>
              </div>
            </Link>
          ))}
          <button
            type="button"
            onClick={openDialog}
            className="flex items-center justify-center gap-2 px-4 py-4 border-b border-dashed text-sm text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors w-full"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
            </svg>
            New Channel
          </button>
        </div>
      </main>

      <Dialog open={dialogOpen} onOpenChange={handleDialogChange}>
        <DialogContent className="rounded-none max-w-sm w-[calc(100vw-2rem)]">
          <DialogHeader>
            <DialogTitle>New channel</DialogTitle>
          </DialogHeader>

          <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4 mt-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="channel-name">Name</Label>
              <Input
                id="channel-name"
                placeholder="e.g. General announcements"
                className="rounded-none h-11"
                {...register('name', { required: 'Channel name is required' })}
              />
              {errors.name && (
                <p className="text-sm text-destructive">{errors.name.message}</p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="channel-description">Description (optional)</Label>
              <Textarea
                id="channel-description"
                placeholder="Shown to attendees when selecting channels"
                className="min-h-20"
                {...register('description')}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="opens-at">Opens at (optional)</Label>
              <Input
                id="opens-at"
                type="datetime-local"
                className="rounded-none h-11"
                {...register('opensAt')}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="closes-at">Closes at (optional)</Label>
              <Input
                id="closes-at"
                type="datetime-local"
                className="rounded-none h-11"
                {...register('closesAt', {
                  validate: val => {
                    if (!val || !opensAt) return true
                    return new Date(val) > new Date(opensAt) || 'Closes at must be after opens at'
                  },
                })}
              />
              {errors.closesAt && (
                <p className="text-sm text-destructive">{errors.closesAt.message}</p>
              )}
            </div>

            {createError && <p className="text-sm text-destructive">{createError}</p>}

            <Button type="submit" disabled={isSubmitting} className="rounded-none h-11 w-full">
              {isSubmitting ? 'Creating…' : 'Create channel'}
            </Button>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
