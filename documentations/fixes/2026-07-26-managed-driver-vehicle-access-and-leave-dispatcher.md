# Upravljani vozač nije mogao da upravlja vozilima + nije mogao da napusti dispečera

**Datum:** 2026-07-26
**Fajlovi:** `backend/internal/store/driver.go`, `backend/internal/httpapi/{auth,dispatcher,server}.go`, `mobile/lib/models/account_status.dart`, `mobile/lib/services/api_client.dart`, `mobile/lib/screens/{dispatcher_requests,offered_trips,vehicle_list}_screen.dart`

Korisnik je uživo testirao dispečer/vozač tok (Blok K, `2026-07-26-scope-cut-closures.md`) i prijavio da vozač, čim se poveže sa dispečerom, gubi svaki pristup upravljanju sopstvenim vozilima (dodavanje/izmena/brisanje) — i da bi bilo korisno da vozač sam može da raskine vezu.

## Uzrok

Kad se vozač poveže sa dispečerom, `homeScreenFor()` ga trajno preusmerava na `OfferedTripsScreen` (umesto `VehicleListScreen`, koji ostaje samo za samostalne vozače). `OfferedTripsScreen` nikad nije imao ni link ka listi vozila — pa je Blok M (izmena/brisanje vozila) bio potpuno nedostupan upravljanom vozaču, iako je backend odavno podržavao njegova lična vozila (dispečer ih je i pre mogao koristiti pri kreiranju ture, Blok K).

## Popravka

- **Pristup vozilima vraćen**: `OfferedTripsScreen` dobija ikonicu "Moja vozila" → `VehicleListScreen`. Ta lista već ima pun CRUD (dodavanje preko FAB-a, izmena/brisanje preko menija na svakom vozilu, Blok M) — samo joj je nedostajao ulaz za upravljanog vozača.
- Pošto backend odbija samostalno kreiranje ture za upravljanog vozača (`handleCreateTrip`: "your dispatcher creates trips for you"), tap na vozilo u ovom kontekstu više NE vodi na `RouteRequestScreen` (što bi vodilo direktno u grešku) — umesto toga prikazuje kratku poruku da dispečer kreira ture. Za samostalnog vozača ponašanje ostaje nepromenjeno.
- **Napuštanje dispečera**: nov `DriverStore.ClearDispatcher` + `POST /api/v1/driver/leave-dispatcher` (vozač sam sebi raskida vezu; `400` ako trenutno nije upravljan). Flotna vozila postaju odmah nedostupna (prirodna posledica `vehicleAccessible` provere trenutnog `dispatcher_id`), lična vozila i istorija tura ostaju netaknuti.
- `GET /api/v1/auth/me` sad vraća i `dispatcher_username` (dodatan lookup samo kad je `dispatcher_id` postavljen) — koristi se da se na ekranu "Zahtevi dispečera" prikaže "Trenutni dispečer: [ime]" + dugme "Napusti", sa potvrdom pre raskida.

## Verifikacija

Uživo protiv Docker stack-a: `/auth/me` pre raskida pokazuje `dispatcher_username`; `leave-dispatcher` na nepovezanom nalogu → `400`; na povezanom → `200`; `/auth/me` posle raskida više nema `dispatcher_id`; vozač odmah može ponovo da doda lično vozilo (`201`). `go build`/`vet`/`test` i `flutter analyze`/`test` čisti.
