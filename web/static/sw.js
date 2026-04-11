// Service worker — handles background push events and shows notifications.

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

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    clients.matchAll({ type: "window", includeUncontrolled: true }).then((list) => {
      for (const client of list) {
        if ("focus" in client) return client.focus();
      }
      if (clients.openWindow) return clients.openWindow("/");
    })
  );
});
