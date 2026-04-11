import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiFetch } from '../lib/api.js'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card.jsx'
import { Input } from '../components/ui/input.jsx'
import { Button } from '../components/ui/button.jsx'
import { Label } from '../components/ui/label.jsx'

export default function Auth() {
  const navigate = useNavigate()
  const [mode, setMode] = useState('login') // 'login' | 'register'
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (localStorage.getItem('pager_token')) {
      navigate('/dashboard', { replace: true })
    }
  }, [navigate])

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const path = mode === 'login' ? '/auth/login' : '/auth/register'
      const res = await apiFetch(path, {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      })

      const data = await res.json()

      if (!res.ok) {
        setError(data.error || 'Something went wrong.')
        return
      }

      localStorage.setItem('pager_token', data.token)
      navigate('/dashboard', { replace: true })
    } catch {
      setError('Could not reach the server. Try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-dvh flex items-center justify-center px-4 bg-background">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <span className="text-4xl font-black tracking-tighter">
            Pa<span className="text-primary">g</span>er
          </span>
        </div>

        <Card className="rounded-none shadow-none border">
          <CardHeader className="pb-4">
            <CardTitle className="text-lg">
              {mode === 'login' ? 'Sign in' : 'Create account'}
            </CardTitle>
          </CardHeader>

          <CardContent>
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                  required
                  className="rounded-none h-11"
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  required
                  className="rounded-none h-11"
                />
              </div>

              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}

              <Button
                type="submit"
                disabled={loading}
                className="rounded-none h-11 w-full mt-1"
              >
                {loading
                  ? 'Please wait…'
                  : mode === 'login' ? 'Sign in' : 'Create account'}
              </Button>
            </form>

            <p className="mt-4 text-center text-sm text-muted-foreground">
              {mode === 'login' ? (
                <>No account?{' '}
                  <button
                    type="button"
                    onClick={() => { setMode('register'); setError('') }}
                    className="underline underline-offset-4 text-foreground"
                  >
                    Register
                  </button>
                </>
              ) : (
                <>Already have an account?{' '}
                  <button
                    type="button"
                    onClick={() => { setMode('login'); setError('') }}
                    className="underline underline-offset-4 text-foreground"
                  >
                    Sign in
                  </button>
                </>
              )}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
