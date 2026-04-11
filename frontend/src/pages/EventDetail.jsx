import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useParams, Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { apiFetch } from '../lib/api.js'
import { Button } from '../components/ui/button.jsx'
import { Input } from '../components/ui/input.jsx'
import { Label } from '../components/ui/label.jsx'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../components/ui/dialog.jsx'

export default function EventDetail() {
  const navigate = useNavigate()
  const { eventId } = useParams()
  const [event, setEvent] = useState(null)
  const [channels, setChannels] = useState([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [createError, setCreateError] = useState('')

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
      const [eventsRes] = await Promise.all([apiFetch('/events')])
      if (eventsRes.status === 401) {
        localStorage.removeItem('pager_token')
        navigate('/', { replace: true })
        return
      }
      const events = await eventsRes.json()
      const found = events.find(e => e.ID === eventId)
      if (found) setEvent(found)
      await fetchChannels()
      setLoading(false)
    }

    init()
  }, [eventId, navigate, fetchChannels])

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

      <main className="flex-1 px-4 py-6 w-full max-w-2xl mx-auto">
        <div className="flex items-center gap-3 mb-6">
          <h1 className="text-2xl font-semibold">{event.Name}</h1>
          <span className="text-xs border px-2 py-1 capitalize text-muted-foreground">
            {event.Status}
          </span>
        </div>

        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-semibold">Channels</h2>
          <Button className="rounded-none h-9" onClick={openDialog}>New Channel</Button>
        </div>

        {channels.length === 0 ? (
          <p className="text-muted-foreground text-sm py-8 text-center">
            No channels yet. Create your first one.
          </p>
        ) : (
          <div className="flex flex-col gap-2">
            {channels.map(ch => (
              <div
                key={ch.ID}
                className="border px-4 py-3 flex items-center justify-between"
              >
                <span className="font-medium text-sm">{ch.Name}</span>
                <span className="text-xs text-muted-foreground capitalize">{ch.Status}</span>
              </div>
            ))}
          </div>
        )}
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
