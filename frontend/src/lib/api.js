const BASE = ''

export async function apiFetch(path, options = {}) {
  const token = localStorage.getItem('pager_token')

  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  })

  return res
}
