import { useState, useEffect, useCallback } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { format } from 'date-fns'
import { apiFetch } from '../lib/api.js'
import { Button } from '../components/ui/button.jsx'
import { Card, CardContent } from '../components/ui/card.jsx'
import { Input } from '../components/ui/input.jsx'
import { Label } from '../components/ui/label.jsx'
import { Checkbox } from '../components/ui/checkbox.jsx'
import { Calendar } from '../components/ui/calendar.jsx'
import { Popover, PopoverContent, PopoverTrigger } from '../components/ui/popover.jsx'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../components/ui/dialog.jsx'

function DatePicker({ label, value, onChange, id }) {
  const [open, setOpen] = useState(false)
  return (
    <div className="flex flex-col gap-1.5 flex-1">
      {label && <Label htmlFor={id}>{label}</Label>}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            id={id}
            type="button"
            variant="outline"
            className="rounded-none h-11 w-full justify-start font-normal"
          >
            {value ? format(value, 'MMM d, yyyy') : 'Pick a date'}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0 rounded-none" align="start">
          <Calendar
            mode="single"
            selected={value}
            onSelect={d => { onChange(d); setOpen(false) }}
            initialFocus
          />
        </PopoverContent>
      </Popover>
    </div>
  )
}

function toISO(date, time) {
  if (!date || !time) return null
  const [hours, minutes] = time.split(':')
  const d = new Date(date)
  d.setHours(Number(hours), Number(minutes), 0, 0)
  return d.toISOString()
}

const EMPTY_FORM = {
  name: '',
  date: null,
  startTime: '',
  endTime: '',
  multiDay: false,
  endDate: null,
}

export default function Dashboard() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [events, setEvents] = useState([])
  const [loadingAuth, setLoadingAuth] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [form, setForm] = useState(EMPTY_FORM)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState('')

  const set = (key, val) => setForm(f => ({ ...f, [key]: val }))

  const logout = useCallback(() => {
    localStorage.removeItem('pager_token')
    navigate('/', { replace: true })
  }, [navigate])

  const fetchEvents = useCallback(async () => {
    const res = await apiFetch('/events')
    if (res.ok) setEvents(await res.json())
  }, [])

  useEffect(() => {
    if (!localStorage.getItem('pager_token')) {
      navigate('/', { replace: true })
      return
    }
    async function init() {
      const res = await apiFetch('/auth/me')
      if (res.status === 401) {
        localStorage.removeItem('pager_token')
        navigate('/', { replace: true })
        return
      }
      const me = await res.json()
      setEmail(me.email || '')
      await fetchEvents()
      setLoadingAuth(false)
    }
    init()
  }, [navigate, fetchEvents])

  function openDialog() {
    setForm(EMPTY_FORM)
    setCreateError('')
    setDialogOpen(true)
  }

  async function handleCreateEvent(e) {
    e.preventDefault()
    setCreateError('')

    const { name, date, startTime, endTime, multiDay, endDate } = form

    // Validate date(s)
    if (!date) return setCreateError('Please pick a date.')
    if (multiDay && !endDate) return setCreateError('Please pick an end date.')

    // Validate times
    if (!startTime) return setCreateError('Please enter a start time.')
    if (!endTime) return setCreateError('Please enter an end time.')

    const startsAtISO = toISO(date, startTime)
    const endsAtISO = toISO(multiDay ? endDate : date, endTime)

    // End must be after start
    if (new Date(endsAtISO) <= new Date(startsAtISO)) {
      return setCreateError('End date/time must be after start date/time.')
    }

    setCreating(true)
    try {
      const res = await apiFetch('/events', {
        method: 'POST',
        body: JSON.stringify({ name, starts_at: startsAtISO, ends_at: endsAtISO }),
      })
      if (!res.ok) {
        const data = await res.json()
        setCreateError(data.error || 'Failed to create event.')
        return
      }
      setDialogOpen(false)
      await fetchEvents()
    } catch {
      setCreateError('Could not reach the server.')
    } finally {
      setCreating(false)
    }
  }

  if (loadingAuth) {
    return (
      <div className="min-h-dvh flex items-center justify-center">
        <p className="text-muted-foreground text-sm">Loading…</p>
      </div>
    )
  }

  return (
    <div className="min-h-dvh flex flex-col bg-background">
      {/* Nav */}
      <header className="border-b px-4 h-14 flex items-center justify-between gap-4">
        <span className="text-xl font-black tracking-tighter">
          Pa<span className="text-primary">g</span>er
        </span>
        <div className="flex items-center gap-3 min-w-0">
          {email && (
            <span className="text-sm text-muted-foreground truncate hidden sm:block">{email}</span>
          )}
          <Button variant="outline" size="sm" className="rounded-none shrink-0" onClick={logout}>
            Log out
          </Button>
        </div>
      </header>

      {/* Body */}
      <main className="flex-1 px-4 py-6 w-full max-w-2xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-xl font-semibold">Events</h1>
          <Button className="rounded-none h-11" onClick={openDialog}>New Event</Button>
        </div>

        {events.length === 0 ? (
          <p className="text-muted-foreground text-sm py-8 text-center">
            You have no events yet. Create your first one.
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            {events.map(event => (
              <Link key={event.ID} to={`/events/${event.ID}`}>
                <Card className="rounded-none hover:bg-muted/50 transition-colors cursor-pointer">
                  <CardContent className="py-4 px-4 flex items-center justify-between">
                    <span className="font-medium">{event.Name}</span>
                    <span className="text-xs text-muted-foreground capitalize">{event.Status}</span>
                  </CardContent>
                </Card>
              </Link>
            ))}
          </div>
        )}
      </main>

      {/* New Event Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="rounded-none max-w-sm w-[calc(100vw-2rem)]">
          <DialogHeader>
            <DialogTitle>New event</DialogTitle>
          </DialogHeader>

          <form onSubmit={handleCreateEvent} className="flex flex-col gap-4 mt-2">
            {/* Name */}
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="event-name">Name</Label>
              <Input
                id="event-name"
                value={form.name}
                onChange={e => set('name', e.target.value)}
                placeholder="e.g. React Conf 2026"
                required
                className="rounded-none h-11"
              />
            </div>

            {/* Single-day: one date + start/end times */}
            {!form.multiDay ? (
              <>
                <DatePicker
                  label="Date"
                  id="event-date"
                  value={form.date}
                  onChange={d => set('date', d)}
                />
                <div className="flex gap-2">
                  <div className="flex flex-col gap-1.5 flex-1">
                    <Label htmlFor="start-time">Start time</Label>
                    <Input
                      id="start-time"
                      type="time"
                      value={form.startTime}
                      onChange={e => set('startTime', e.target.value)}
                      className="rounded-none h-11"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5 flex-1">
                    <Label htmlFor="end-time">End time</Label>
                    <Input
                      id="end-time"
                      type="time"
                      value={form.endTime}
                      onChange={e => set('endTime', e.target.value)}
                      className="rounded-none h-11"
                    />
                  </div>
                </div>
              </>
            ) : (
              /* Multi-day: separate date+time for start and end */
              <>
                <div className="flex flex-col gap-1.5">
                  <Label>Starts</Label>
                  <div className="flex gap-2">
                    <DatePicker value={form.date} onChange={d => set('date', d)} />
                    <Input
                      type="time"
                      value={form.startTime}
                      onChange={e => set('startTime', e.target.value)}
                      className="rounded-none h-11 w-28"
                    />
                  </div>
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label>Ends</Label>
                  <div className="flex gap-2">
                    <DatePicker value={form.endDate} onChange={d => set('endDate', d)} />
                    <Input
                      type="time"
                      value={form.endTime}
                      onChange={e => set('endTime', e.target.value)}
                      className="rounded-none h-11 w-28"
                    />
                  </div>
                </div>
              </>
            )}

            {/* Multi-day toggle */}
            <div className="flex items-center gap-2">
              <Checkbox
                id="multi-day"
                checked={form.multiDay}
                onCheckedChange={v => set('multiDay', v)}
              />
              <Label htmlFor="multi-day" className="font-normal cursor-pointer">
                Multi-day event
              </Label>
            </div>

            {createError && <p className="text-sm text-destructive">{createError}</p>}

            <Button type="submit" disabled={creating} className="rounded-none h-11">
              {creating ? 'Creating…' : 'Create event'}
            </Button>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
