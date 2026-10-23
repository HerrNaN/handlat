// Retire previously installed offline workers and their caches. Keep serving this
// file so returning clients can update. No fetch handler: private APIs must never
// be stored in a service-worker cache. This does not touch IndexedDB data.

self.addEventListener("install", (event) => {
  event.waitUntil(self.skipWaiting());
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      // Cleanup caches
      const cacheNames = await caches.keys();
      await Promise.all(cacheNames.map((name) => caches.delete(name)));

      // Unregister service worker
      await self.registration.unregister();

      // Reload all active clients
      const clients = await self.clients.matchAll({ type: "window" });
      await Promise.allSettled(clients.map((client) => client.navigate(client.url)));
    })(),
  );
});
