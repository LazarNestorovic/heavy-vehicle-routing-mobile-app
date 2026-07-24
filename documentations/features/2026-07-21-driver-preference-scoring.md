# Driver nalozi + dinamičan preference-driven scoring (Faze 1-4)

**Datum:** 2026-07-21
**Plan:** `/home/lazar/.claude/plans/lovely-cooking-puppy.md` (plan mode, odobren pre implementacije)
**Fajlovi:** `backend/internal/auth/`, `backend/internal/store/{driver,preferences,favorite_stop}.go`, `backend/internal/httpapi/{auth,middleware,preferences,favorite_stops}.go`, `backend/internal/scoring/scoring.go`, `backend/internal/valhalla/client.go`

## Zašto

Konkretan, potvrđen bug: ruta Radalj→Klisa je birala variantu koja je **47% duža i 30% sporija** (209.4km/146.9min umesto 142.7km/113.1min) samo zato što je imala veći `highway_ratio` — stara formula (`maneuver_count*1.5 + (1-highway_ratio)*100 + ferry/toll`) nije imala nikakvu kaznu za dužinu/trajanje. Umesto uske zakrpe, odlučeno je (posle detaljnog planiranja uz korisnika, plan mode) da se ovo reši fundamentalnije: **pravi nalog sistem za vozače + podesivi prioriteti (1-5) po više dimenzija**, tako da se formula prirodno ispravi kao nusprodukt, a sistem postane fleksibilan za buduće potrebe.

## Auth (Faza 1)

- `drivers` tabela: `username` (UNIQUE), `password_hash` (bcrypt).
- JWT (ne session tabela) — `github.com/golang-jwt/jwt/v5`, HS256, claims `driver_id`/`username`/`exp` (30 dana). Potpisni ključ `JWT_SECRET` iz env-a (nasumično generisan u `docker-compose.yml`).
- **Namerno ograničenje:** nema server-side revokacije (nema blocklist tabele) — "logout" je samo brisanje tokena na klijentu. Standardno prihvatljivo pojednostavljenje za MVP.
- **Login obavezan svuda** (korisnikova eksplicitna odluka u plan-mode razgovoru) — `RequireAuth` middleware štiti sve postojeće endpoint-e (`/vehicles`, `/routes`, `/trips`, `/ws/trips/{id}`). WebSocket koristi `RequireAuthQuery` (token kao `?token=` query param, ne header — browseri ne mogu da postave custom header na WS handshake).
- `POST /api/v1/auth/register`, `POST /api/v1/auth/login` — oba vraćaju `{token, driver_id}`.

## Vlasništvo vozila (Faza 2)

- `vehicles.driver_id`, `trips.driver_id` — FK ka `drivers`.
- Vozač može imati **više vozila** (1:N, korisnikova eksplicitna odluka) — `GET /api/v1/vehicles` (nova, lista) pored postojećeg `GET /api/v1/vehicles/{id}`.
- Ownership provera na `GET /vehicles/{id}`, `POST /trips` (vehicle_id mora pripadati ulogovanom vozaču), `GET /trips/{id}`, i `GET /ws/trips/{id}` (403 ako ne pripada).

## Driver preference entitet (Faza 3)

`driver_preferences` (1:1 sa `drivers`, default `3` za sve — "neutralno", blisko starom ponašanju):
- `fuel_priority`, `cargo_priority`, `highway_priority`, `time_priority` (SMALLINT, CHECK 1-5)
- `preferred_fuel_brand` (nullable, za Fazu 5)

`driver_favorite_stops` (za Fazu 5) — sačuvane lokacije (lat/lon/name).

`GET`/`PUT /api/v1/preferences`, `POST`/`GET`/`DELETE /api/v1/favorite-stops(/{id})`.

## Nova scoring formula (Faza 4) — rešava Radalj bug

`backend/internal/scoring/scoring.go`, svaki član skaliran sa `priority/3` (3 = neutralno):

```go
timeTerm    := (c.DurationMin - fastestDurationMin) / fastestDurationMin * 150
highwayTerm := (1 - c.HighwayRatio) * 100
fuelTerm    := distanceKm*(1 + weightKg/40000*0.3) + maneuverCount*0.05  // prost proxy, bez elevation podataka
cargoTerm   := sharpManeuverCount * 3.0                                  // vidi ispod

score := (time/3)*timeTerm + (highway/3)*highwayTerm + (fuel/3)*fuelTerm + (cargo/3)*cargoTerm
       + maneuverCount*1.5 + ferryPenalty + tollPenalty
```

**`sharpManeuverCount`** (cargo-osetljivost) zahtevao je novi podatak — Valhalla `maneuver.type` polje se do sada nije parsiralo. Pre implementacije, **potvrđeni su stvarni kodovi live pozivom** (ne pretpostavka): `11`=oštro desno, `12`/`13`=U-okret, `14`=oštro levo, `26`=ulazak u rotor — isti princip provere kao za `highway`/`toll` polja ranije u projektu.

**Bitna posledica koja je otkrivena testom:** `explain.Explainer` je morao da se izmeni da i referentna/probna ruta prolaze kroz **isti** `scoring.Rank()` poziv (sa istim preferencama) kao izabrana ruta — inače bi objašnjenje rute lažno pripisivalo razliku ograničenjima vozila umesto scoring formuli (isti tip greške kao [ranije pronađeni bug](2026-07-21-route-explainability.md), sada ponovo relevantan jer je formula sada parametrizovana).

## Verifikacija (2026-07-21)

- `go test ./internal/auth/... ./internal/scoring/...` — 5 + 4 testa, uključujući `TestRank_DefaultPreferences_PicksDirectRouteOverRadaljDetour` (sintetička reprodukcija sa stvarnim brojevima iz Radalj slučaja) i `TestRank_HighwayLover_CanStillPreferTheDetour` (dokazuje da je formula stvarno podesiva, ne samo zakrpljena).
- Uživo, isti Radalj→Klisa poziv:
  - Default preference (3/3/3/3): **142.7km/113.1min biran** (ranije: 209.4km/146.9min) — bug potvrđeno rešen.
  - `fuel_priority=1, time_priority=1, highway_priority=5`: **209.4km/146.9min biran** — potvrđeno da je formula genuinski dinamična, ne hardkodovano "uvek biraj kraće".
- Ownership: dva vozača, svaki napravi vozilo, vozač B dobija 403 na vozilo/trip vozača A — potvrđeno.
- Preferences/favorite-stops CRUD (GET/PUT/POST/DELETE) — potvrđeno uživo.

## Šta namerno NIJE urađeno (za sada)

- Faza 5 (preferirane pumpe utiču na predlog pauze i na samu rutu) i Faza 6 (Flutter ekrani) — sledeći koraci istog plana, posebno dokumentovani kad se završe.
- `fuelProxy` nije kalibrisan na stvarnu potrošnju goriva (nema elevation podataka u Valhalla config-u) — čisto relativni signal za poređenje kandidata, isto kao i stari `risk_score`.
- Nema server-side JWT revokacije/blocklist-e.
