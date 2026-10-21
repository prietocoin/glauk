const CACHE = 'glauk-tasas-v1';
const ASSETS = [
  './', './index.html', './manifest.json', './js/main.js', './js/api.js',
  './js/views/monitor.js', './js/views/lotes.js', './js/views/socios.js',
  './js/views/hoo.js', './js/components/cartelera.js',
  '../shared/css/base.css', '../shared/css/components.css', '../shared/css/print.css',
  '../shared/js/dom.js', '../shared/js/http.js', '../shared/js/format.js',
  '../shared/js/toast.js', '../shared/js/reporte-canvas.js',
  '../shared/brand/favicon.svg', '../shared/brand/logo-fundablock.png'
];

self.addEventListener('install', (event) => {
  event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll(ASSETS)).then(() => self.skipWaiting()));
});

self.addEventListener('activate', (event) => {
  event.waitUntil(caches.keys().then((keys) => Promise.all(
    keys.filter((key) => key !== CACHE).map((key) => caches.delete(key))
  )).then(() => self.clients.claim()));
});

self.addEventListener('fetch', (event) => {
  const request = event.request;
  if (request.method !== 'GET') return;
  const url = new URL(request.url);
  if (url.origin !== location.origin || url.pathname === '/api' || url.pathname.startsWith('/api/')) return;
  event.respondWith(caches.match(request).then((cached) => cached || fetch(request).then((response) => {
    if (!response.ok || response.type !== 'basic') return response;
    const copy = response.clone();
    caches.open(CACHE).then((cache) => cache.put(request, copy));
    return response;
  }).catch(() => request.mode === 'navigate' ? caches.match('./index.html') : Response.error())));
});
