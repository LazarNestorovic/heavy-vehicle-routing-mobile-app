# Vehicle/Trip persistencija u Postgres

**Datum:** 2026-07-21
**Fajlovi:** [`backend/internal/db/`](../../backend/internal/db/), [`backend/internal/store/`](../../backend/internal/store/), [`backend/internal/httpapi/`](../../backend/internal/httpapi/)

## Šta je dodato

Backend je povezan na Postgres (podignut još u [Go backend skeletu](2026-07-21-go-backend-skeleton.md), ali do sada nekorišćen). Preduslov za RabbitMQ `trip.started` tok i WebSocket simulaciju pozicije iz nedelje 3 (`SPECIFIKACIJA.md`).

```
backend/internal/db/db.go        — Connect() + Migrate() (idempotentan schema.sql inline, bez eksternog migration alata)
backend/internal/store/vehicle.go — VehicleStore: Create, Get
backend/internal/store/trip.go    — TripStore: Create
```

`handlers.go` je razbijen u `server.go` (wiring/Server struct), `vehicles.go`, `routes.go` (postojeća logika izvučena u `bestRoute()` helper, sada deljen sa trips.go), `trips.go` — fajl je narastao dovoljno da razdvajanje po resursu ima smisla.

## Šema (bez eksternog migration alata — `CREATE TABLE IF NOT EXISTS` na svaki start)

```sql
vehicles: id, height_m, width_m, length_m, weight_kg, axle_load_kg, hazmat, created_at
  CHECK (axle_load_kg <= weight_kg)  -- isti uslov kao app-level validacija, kao odbrana u dubinu

trips: id, vehicle_id (FK), origin_lat/lon, destination_lat/lon,
       distance_km, duration_min, risk_score, shape, status, created_at
```

## API

```
POST /api/v1/vehicles        {height_m, width_m, length_m, weight_kg, axle_load_kg, hazmat} -> 201 {id, ...}
GET  /api/v1/vehicles/{id}   -> 200 {id, ...} | 404
POST /api/v1/trips           {vehicle_id, origin, destination} -> 201 {id, vehicle_id, status, distance_km, duration_min, shape, risk_score, candidates[]} | 404 (vehicle_id ne postoji) | 422 (nema validne rute)
```

`POST /api/v1/trips` interno radi tačno ono što `POST /api/v1/routes` radi (učitava profil, zove Valhalla-u, rangira kandidate — `bestRoute()`), samo profil čita iz baze po `vehicle_id` umesto iz tela zahteva, i **perzistira** rezultat. `POST /api/v1/routes` ostaje nepromenjen kao stateless preview (korisno za Flutter da prikaže rutu pre nego što se vozač "obaveže" na trip).

**Odstupanje od originalne API skice u `SPECIFIKACIJA.md` (sekcija 5):** tamo je `/trips` trebalo da prima `route_id` (referencu na prethodno izračunatu, perzistiranu rutu). Pojednostavljeno je u jedan poziv (`vehicle_id` + `origin`/`destination` → odmah računa i perzistira) jer još nema potvrđene UX potrebe za odvojenim "preview pa potvrdi" korakom — lako se doda kasnije ako Flutter tok to zahteva.

## Verifikacija (2026-07-21)

- `POST /api/v1/vehicles` → `201`, vraća `id`.
- `GET /api/v1/vehicles/{id}` → `200` sa istim podacima; `404` za nepostojeći id.
- `POST /api/v1/trips` sa validnim `vehicle_id` → `201`, red upisan u `trips` tabeli (potvrđeno direktno preko `psql`).
- `POST /api/v1/trips` sa `vehicle_id: 9999` → `404 "vehicle not found"`.
- `POST /api/v1/vehicles` sa `axle_load_kg > weight_kg` → `400` (app-level validator, isti kao [ranija provera](2026-07-21-risk-scoring-layer.md)) — DB `CHECK` konstrejnt postoji kao dodatna linija odbrane, nije još direktno pogođen testom (app validacija ga uvek preduhitri).

## Šta namerno NIJE urađeno

- Nema `PUT`/`DELETE` za vozila/putovanja — nije još potrebno.
- Nema auth-a — `vehicle_id` je prost integer, bilo ko može da ga pogodi/enumeriše. Prihvatljivo za MVP bez multi-usera (`SPECIFIKACIJA.md`), postaje problem čim se doda pravi korisnik/vozač koncept.
- `Driver`/`RestStop`/`LocationPing` tabele iz `SPECIFIKACIJA.md` (sekcija 4) nisu dodate — čekaju RabbitMQ/WebSocket tok (sledeći korak).
