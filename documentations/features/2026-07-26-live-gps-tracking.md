# Pravo GPS praćenje uživo (dopuna simulacije)

**Datum:** 2026-07-26
**Plan:** `/home/lazar/.claude/plans/lovely-cooking-puppy.md` (Blok D, runda 2 zatvaranja obim-cut-ova)
**Fajlovi:** `backend/internal/ws/gateway.go`, `backend/internal/httpapi/trips.go`, `backend/internal/store/trip.go`, `backend/internal/queue/events.go`, `backend/internal/worker/trip_worker.go`, `mobile/lib/services/{location_service,trip_socket,chat_socket}.dart`, `mobile/lib/screens/active_trip_screen.dart`

## Zašto

`GET /ws/trips/{id}` je od početka projekta simulirao poziciju hodajući duž Valhalla shape-a (60s wall-clock, fiksni tempo) — svesna, dokumentovana zamena za pravi GPS jer projekat nikad nije imao pravo teretno vozilo na terenu. Sada kad telefon postoji u ruci vozača tokom testiranja, korisnik je zatražio da se doda **stvarno** praćenje, uz eksplicitnu odluku da simulacija ostane automatski fallback (npr. ako vozač odbije GPS dozvolu) — niko ne gubi dosadašnju demo-sposobnost bez telefona.

## Mehanizam

**Backend (`ws.Gateway`)**: novi `live map[int64]*liveTrip` (mutex-zaštićen) — svaka tura koja je primila BAREM JEDAN pravi GPS ping prelazi u "live" režim za SVE svoje WS posmatrače (vozačev ekran i dispečerova flotna mapa, obe se konektuju na isti `/ws/trips/{id}`).

- `Gateway.ReportPosition(trip, lat, lon)` — poziva se iz novog `POST /api/v1/trips/{id}/position` (samo dodeljeni vozač). Progress/ETA se aproksimiraju preko preostale haversine udaljenosti do odredišta / originalno planiranog tempa (isti stil aproksimacije kao postojeći `PointAtFraction`).
- `HandleTripStream` grana se: ako tura već ima live stanje, pretplati se na njega; inače postojeće `simulate()` ponašanje bez izmene. `simulate()` sam proverava na svakom tick-u da li je u međuvremenu stigao pravi ping i, ako jeste, prepušta relay live kanalu (glatka predaja ako se GPS "uključi" usred simulacije).
- `Gateway.CompleteTrip(tripID)` (iz novog `POST /api/v1/trips/{id}/complete`) šalje finalni `{status:"arrived"}` svim posmatračima i briše live stanje. **Zašto poseban endpoint**: pravi GPS nema pouzdan auto-detekt dolaska kao simulacija (koja zna kad `progress_fraction=1`) — vozač eksplicitno potvrđuje dolazak.
- Novi `store.TripStatusCompleted` + `TripStore.MarkCompleted` (pokušava `in_progress→completed`, pa `created→completed` — GPS ping/complete teoretski mogu stići pre nego što worker obradi turu).
- Uklonjen mrtav kod: `trip.eta_updated` RabbitMQ poruka (`worker/trip_worker.go`) se publikovala ali niko je nije konzumirao (WS gateway čita direktno iz baze) — live GPS rad ovo čini još irelevantnijim.

**Flutter**: `geolocator` (v14) u `ActiveTripScreen` — `LocationService.ensurePermission()` pa `positionStream()` (foreground-only, BEZ `ACCESS_BACKGROUND_LOCATION`, praćenje samo dok je ekran otvoren). Svaka pozicija odmah ažurira vozačevu sopstvenu mapu (bez čekanja round-trip-a) i šalje se backend-u preko `api.reportPosition(...)` (fire-and-forget, dropped ping nije fatalan). Dozvola odbijena → ništa se ne menja, postojeći WS-simulacija tok radi identično kao pre ove izmene. Novo dugme "Stigao sam" (AppBar akcija, vidljivo samo dok je live GPS aktivan) zove `api.completeTrip(...)`.

`TripSocket`/`ChatSocket` dobili auto-reconnect (retry sa backoff-om 1s/2s/5s/10s) — prirodna dopuna, jer dispečer sad prati stvarno kretanje pa prekid konekcije više boli nego kod čiste simulacije.

## Napomena o prelaznom stanju

Odmah po startu ture, vozačev ekran može kratko prikazati DVE različite pozicije — sopstveni GPS fix (trenutan) naspram serverove simulacije (dok prvi pravi ping ne prebaci gateway u live režim). Jednokratan prelaz, ne bag — komentarisano direktno u kodu (`active_trip_screen.dart`).

## Verifikacija

Live curl protiv Docker stack-a: `POST /trips/{id}/position` → `204`; `POST /trips/{id}/complete` → `200` sa `status:"completed"`; ponovni `complete` → `409` "trip is not currently active". `go build`/`vet`/`test` i `flutter analyze`/`test` čisti. **Ono što ostaje korisniku**: stvarno WS relay ponašanje (dispečerova mapa prati pravu GPS poziciju), tačnost/UX GPS dozvole i dugmeta "Stigao sam" zahtevaju fizički uređaj — nisu proverljivi preko curl-a.
