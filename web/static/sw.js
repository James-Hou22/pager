// Service worker — handles background push events and shows notifications.

// ── IndexedDB helpers ─────────────────────────────────────────────────────────

function openDB() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open('pager', 1);
    req.onupgradeneeded = () => req.result.createObjectStore('kv');
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

function idbSet(key, value) {
  return openDB().then(db => new Promise((resolve, reject) => {
    const tx = db.transaction('kv', 'readwrite');
    tx.objectStore('kv').put(value, key);
    tx.oncomplete = resolve;
    tx.onerror = () => reject(tx.error);
  }));
}

function idbGet(key) {
  return openDB().then(db => new Promise((resolve, reject) => {
    const tx = db.transaction('kv', 'readonly');
    const req = tx.objectStore('kv').get(key);
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  }));
}

// ── Activate — claim all clients immediately so this SW controls the page on
//    first install without requiring a second navigation. This ensures
//    postMessage calls from the page reach this SW right away.
self.addEventListener("activate", (event) => {
  event.waitUntil(clients.claim());
});

// ── Message handler — page tells SW which event URL to remember ───────────────

self.addEventListener("message", (event) => {
  if (event.data?.type === 'STORE_EVENT_URL') {
    console.log('SW received event URL:', event.data.url)
    idbSet('eventUrl', event.data.url);
  }
});

// ── Push handler ──────────────────────────────────────────────────────────────

self.addEventListener("push", (event) => {
  const text = event.data ? event.data.text() : "New message";
  event.waitUntil(
    self.registration.showNotification("Pager", {
      body: text,
      icon: "/icon-192.png",
      badge: "/icon-192.png",
      tag: "pager-message",        // replaces previous notification instead of stacking
      renotify: true,
    })
  );
});

// ── Notification click — open or focus the attendee's event page ──────────────

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    idbGet('eventUrl').then((url) => {
      console.log("url:", url)
      const target = url || '/';
      return clients.matchAll({ type: "window", includeUncontrolled: true }).then((list) => {
        for (const client of list) {
          if (client.url === target && 'focus' in client) return client.focus();
        }
        if (clients.openWindow) return clients.openWindow(target);
      });
    })
  );
});
