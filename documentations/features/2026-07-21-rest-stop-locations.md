# Rest-stop lokacije (stvarna OSM mesta, ne samo prag u minutima)

**Datum:** 2026-07-21
**Fajlovi:** [`backend/internal/reststop/`](../../backend/internal/reststop/), [`backend/internal/valhalla/polyline.go`](../../backend/internal/valhalla/polyline.go), [`backend/internal/worker/trip_worker.go`](../../backend/internal/worker/trip_worker.go)

## Šta je dodato

Nadogradnja [RabbitMQ worker-a](2026-07-21-rabbitmq-trip-worker.md), koji je do sada računao samo **broj minuta** posle kojih vozaču treba pauza, bez ikakve lokacije. Sada worker pronalazi **stvarnu** benzinsku pumpu/parking/odmaralište iz OSM podataka.

```
backend/data/serbia-rest-stops.osm       — 2061 node-a (1291 fuel + 738 parking + 32 rest_area) za celu Srbiju, ~1.4MB
backend/internal/reststop/reststop.go    — Load() parsira OSM XML, Finder.Nearest() linearna haversine pretraga
backend/internal/valhalla/polyline.go    — DecodePolyline6, PointAtFraction (nova, deljena sa ws paketom)
```

## Mehanizam

1. Worker (već postojeći `trip.started` handler) računa `fraction = 270 / duration_min` kad je `duration_min > 270`.
2. `valhalla.PointAtFraction` dekodira `trip.Shape` (polyline6) i pronalazi tačku na toj frakciji dužine rute — aproksimacija "gde će vozilo biti posle 4.5h", uz pretpostavku približno konstantne brzine (pojednostavljenje, vidi `SPECIFIKACIJA.md` 3.8).
3. `reststop.Finder.Nearest()` pronalazi najbližu pumpu/parking/odmaralište toj tački (linearna pretraga nad 2061 stavki — dovoljno brzo, ne treba prostorni indeks na ovoj skali).
4. Lokacija (lat/lon/name/amenity) se upisuje u `trips` tabelu i uključuje u `trip.eta_updated` event.

## Zašto podaci nisu u `testdata/`

Prvobitno stavljeno u `internal/reststop/testdata/`, premešteno u `backend/data/` kad je postalo jasno da fajl **mora biti u produkcionom Docker image-u** (worker ga čita u runtime-u), ne samo u test okruženju — Go-ova `testdata/` konvencija namerno se izostavlja iz built binary-ja, i originalni `Dockerfile` je kopirao samo binarni fajl, ne i podatke. `Dockerfile` je ažuriran da kopira `backend/data/` u finalni image (`COPY --from=build /src/data /app/data`).

## Verifikacija (2026-07-21)

- `reststop_test.go`: učitano 2061 stanica, `TestFinder_Nearest` nalazi pumpu na 2.5km od tačke na A1 auto-putu (realno).
- End-to-end: trip Subotica → Vranje (320 min) → worker upisuje `next_rest_suggestion_min: 270` **i** `rest_stop: {"lat":43.1495558,"lon":21.8892843,"name":"Pan Ledi","amenity":"fuel"}` — stvarna benzinska pumpa iz OSM-a na putu ka jugu Srbije.

## Šta namerno NIJE urađeno

- Ne uzima u obzir tip vozila (npr. hazmat vozilo možda ne sme da stane na obično parkiralište) — bira se prosto najbliža stanica bilo kog tipa.
- `PointAtFraction` pretpostavlja konstantnu brzinu duž cele rute — stvarna brzina varira (grad vs auto-put); dovoljno za MVP predlog, ne za precizno predviđanje vremena dolaska na stanicu.
- Nema filtriranja po tome da li je stanica uopšte na putu vozila (ili blizu njega) vs. vazdušna udaljenost — `Nearest()` je čisto geografska najbliža tačka.
