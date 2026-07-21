# Changelog

Hronološki indeks svih zapisa u `documentations/`. Najnoviji na vrhu. Svaki red linkuje na detaljan zapis.

- **2026-07-21** — [Vehicle/Trip persistencija u Postgres](features/2026-07-21-vehicle-trip-persistence.md) — `POST /api/v1/vehicles`, `POST /api/v1/trips`; `handlers.go` razbijen po resursu.
- **2026-07-21** — [Risk-scoring sloj nad Valhalla alternativama](features/2026-07-21-risk-scoring-layer.md) — `POST /api/v1/routes` sada bira između 3 kandidata po heurističkom risk score-u, ne samo Valhalla-in default.
- **2026-07-21** — [Go backend skelet + Postgres/PostGIS u docker-compose](features/2026-07-21-go-backend-skeleton.md) — `POST /api/v1/routes` end-to-end preko Valhalla-e, testirano.
- **2026-07-21** — [Popravka osmium filtera: node tagovi i sync ka Valhalla build ulazu](fixes/2026-07-21-osmium-filter-node-tags.md) — rest-area/fuel/parking podaci su se odbacivali; graf rebuild-ovan.
