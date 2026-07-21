# Go backend skelet + Postgres/PostGIS u docker-compose

**Datum:** 2026-07-21
**Fajlovi:** [`backend/`](../../backend/), [`docker-compose.yml`](../../docker-compose.yml)

## Šta je dodato

Prvi deo backend-a iz `SPECIFIKACIJA.md` (nedelja 1, sekcija 3.4): Go servis koji prima zahtev za rutu i vraća truck-costed rutu iz Valhalla-e.

```
backend/
├── go.mod                      — modul heavy-vehicle-routing/backend, go 1.26, bez eksternih zavisnosti (samo stdlib)
├── main.go                     — wiring: config → valhalla.Client → httpapi.Server → http.ListenAndServe
├── Dockerfile                  — multi-stage build (golang:1.26-alpine → alpine:3.20)
├── internal/config/config.go   — čita PORT, VALHALLA_URL, DATABASE_URL iz env-a
├── internal/valhalla/client.go — HTTP klijent ka Valhalla /route, mapira vehicle profil (SI jedinice) u Valhalla truck costing_options (metri/metric tone)
└── internal/httpapi/handlers.go— GET /healthz, POST /api/v1/routes
```

`docker-compose.yml` dobio je dva nova servisa:
- **`postgres`** — `postgis/postgis:16-3.4-alpine`, kredencijali `hvr`/`hvr_dev_password`/db `hvr`, healthcheck preko `pg_isready`, named volume `pgdata`.
- **`backend`** — build iz `./backend`, čeka `valhalla` i `postgres` da budu `healthy` (`depends_on: condition: service_healthy`), port `8080`.

## API kontrakt (trenutno)

```
GET  /healthz
POST /api/v1/routes
  body: {
    "origin": {"lat": .., "lon": ..},
    "destination": {"lat": .., "lon": ..},
    "vehicle": {
      "height_m": .., "width_m": .., "length_m": ..,
      "weight_kg": .., "axle_load_kg": .., "hazmat": bool
    }
  }
  200: { "distance_km": .., "duration_min": .., "shape": "<polyline6>" }
  400: { "error": "..." }  — nevalidan JSON
  422: { "error": "..." }  — Valhalla nije mogla da nađe rutu za date parametre/lokacije
                              (namerno tretirano kao validan poslovni ishod, ne kao 500 —
                              "nema bezbedne rute za ovo vozilo" je legitiman odgovor sistema)
```

**Bitna napomena o jedinicama:** REST API prima `weight_kg`/`axle_load_kg` u kilogramima (SI, prirodno za JSON API), ali Valhalla `costing_options.truck` očekuje metričke tone — konverzija (`/1000`) se dešava u `internal/valhalla/client.go` (`Route()` funkcija). Visina/širina/dužina su već u metrima na obe strane, nema konverzije.

## Kako se testira

```bash
docker compose up -d postgres valhalla backend

curl -s -X POST http://localhost:8080/api/v1/routes -H "Content-Type: application/json" -d '{
  "origin": {"lat": 44.8, "lon": 20.4},
  "destination": {"lat": 45.25, "lon": 19.85},
  "vehicle": {"height_m": 4.0, "width_m": 2.55, "length_m": 16.5, "weight_kg": 40000, "axle_load_kg": 11500, "hazmat": false}
}'
```

Verifikovano ručno (2026-07-21):
- Beograd → Novi Sad sa realnim profilom teretnog vozila → `200`, `distance_km: 84.8`, validan polyline shape.
- Lokacija van pokrivenosti grafa (Pariz) → `422`, `"valhalla: No suitable edges near location"`.
- Nevalidan JSON telo → `400` sa opisom greške.

## Šta namerno NIJE urađeno (scope cut, vidi `SPECIFIKACIJA.md`)

- **Backend se još ne povezuje na Postgres.** `DATABASE_URL` je proosleđen kroz env ali se ne koristi u kodu — Postgres je podignut i dostupan (healthcheck prolazi), ali persistencija vozila/putovanja (`POST /api/v1/vehicles`, `POST /api/v1/trips`) je sledeći korak (nedelja 1-2 plana), ne ovaj.
- Nema `alternates`/risk-scoring sloja (nedelja 2, sekcija 3.3.1 specifikacije) — endpoint vraća samo jednu (najbolju po Valhalla-i) rutu.
- Nema auth-a — namerno, MVP za odbranu ne zahteva multi-usera (vidi `SPECIFIKACIJA.md` sekcija 6/7).
