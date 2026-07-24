# Changelog

Hronološki indeks svih zapisa u `documentations/`. Najnoviji na vrhu. Svaki red linkuje na detaljan zapis.

- **2026-07-21** — [Flutter: nalozi, preference, izbor vozila (Faza 6)](features/2026-07-21-flutter-driver-accounts-preferences.md) — `flutter analyze`/`flutter test` čisti (Flutter sada dostupan u sesiji), UI vizuelno još nije potvrđen.
- **2026-07-21** — [Preferirane pumpe — brend i sačuvane lokacije (Faza 5)](features/2026-07-21-preferred-fuel-stations.md) — utiče na predlog pauze I na risk_score rute, potvrđeno uživo (-20 bonus tačno kako formula predviđa).
- **2026-07-21** — [Driver nalozi + dinamičan preference-driven scoring (Faze 1-4)](features/2026-07-21-driver-preference-scoring.md) — JWT auth, vlasništvo vozila, podesivi prioriteti (1-5), nova formula rešava Radalj bug (potvrđeno uživo + testovima).
- **2026-07-21** — [Flutter mobilna aplikacija (3.9)](features/2026-07-21-flutter-mobile-app.md) — 3 ekrana, pisano bez SDK-a, **nije pokrenuto/vizuelno provereno** — vidi napomenu u dokumentu.
- **2026-07-21** — [WebSocket gateway — simulacija pozicije uživo](features/2026-07-21-websocket-gateway.md) — `GET /ws/trips/{id}`, testirano preko Node ws klijenta.
- **2026-07-21** — [Objašnjenje predložene rute (3.10)](features/2026-07-21-route-explainability.md) — binding-constraint detekcija; pronađen i ispravljen bug (referentna ruta nije koristila isti scoring pipeline).
- **2026-07-21** — [Rest-stop lokacije iz OSM-a](features/2026-07-21-rest-stop-locations.md) — worker sada vraća stvarnu pumpu/parking, ne samo prag u minutima.
- **2026-07-21** — [Bounded A*/Dijkstra modul](features/2026-07-21-bounded-astar-dijkstra.md) — sopstveni algoritam nad OSM podacima, 9 testova (sintetički + stvarni koridor); Novi Banovci slučaj se namerno NE reprodukuje — vredan nalaz dokumentovan.
- **2026-07-21** — [RabbitMQ minimalni tok: trip.started → worker → trip.eta_updated](features/2026-07-21-rabbitmq-trip-worker.md) — async obrada potvrđena end-to-end na dve rute (kratka/duga).
- **2026-07-21** — [Vehicle/Trip persistencija u Postgres](features/2026-07-21-vehicle-trip-persistence.md) — `POST /api/v1/vehicles`, `POST /api/v1/trips`; `handlers.go` razbijen po resursu.
- **2026-07-21** — [Risk-scoring sloj nad Valhalla alternativama](features/2026-07-21-risk-scoring-layer.md) — `POST /api/v1/routes` sada bira između 3 kandidata po heurističkom risk score-u, ne samo Valhalla-in default.
- **2026-07-21** — [Go backend skelet + Postgres/PostGIS u docker-compose](features/2026-07-21-go-backend-skeleton.md) — `POST /api/v1/routes` end-to-end preko Valhalla-e, testirano.
- **2026-07-21** — [Popravka osmium filtera: node tagovi i sync ka Valhalla build ulazu](fixes/2026-07-21-osmium-filter-node-tags.md) — rest-area/fuel/parking podaci su se odbacivali; graf rebuild-ovan.
