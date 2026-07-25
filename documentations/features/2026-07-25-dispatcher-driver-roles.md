# Dispečer i vozač — uloge, dodela tura, praćenje uživo

**Datum:** 2026-07-25
**Plan:** `/home/lazar/.claude/plans/lovely-cooking-puppy.md` (plan mode, odobren pre implementacije, uz dve runde pitanja korisniku i jednu izmenu plana nakon odbijanja prvog `ExitPlanMode` poziva)
**Fajlovi:** `backend/internal/db/migrations/`, `backend/internal/store/{driver,vehicle,trip,dispatcher_request}.go`, `backend/internal/httpapi/{auth,vehicles,trips,dispatcher,roles}.go`, `mobile/lib/screens/{register,offered_trips,dispatcher_*,entry_router}.dart`

## Zašto

Do sada je svaki vozač sam birao rutu i vozilo — realno retko van samostalnih (owner-operator) vozača. Korisnik je tražio novu ulogu **dispečer** koji upravlja flotom: bira rutu/vozilo/tovar i dodeljuje konkretnom vozaču; vozač zatim vidi ponuđenu turu i pokreće je klikom. Samostalni vozači (bez dispečera) zadržavaju potpuno nepromenjen tok.

Tri odluke iz razgovora bile su ključne za oblik implementacije:
1. **Veza dispečer↔vozač se NE uspostavlja pri registraciji** — korisnik je eksplicitno odbio prvi predlog (`dispatcher_username` polje na registraciji) i tražio zahtev/odobrenje tok: dispečer šalje zahtev, vozač odobrava/odbija tek nakon što se uloguje.
2. **Vozila su hibridna** — vozilo pripada ILI floti dispečera ILI pojedinačnom vozaču; dispečer pri kreiranju ture za vozača može koristiti bilo koje od njih.
3. **Scoring autoritet je dispečerov** — kada dispečer kreira turu, koristi se NJEGOV preference profil, ne profil dodeljenog vozača (postojeća `driver_preferences` tabela je već ključirana po ID-ju bilo kog naloga, pa ovo nije zahtevalo nov entitet).

## Migracioni sistem (nova infrastruktura, ne samo ova funkcionalnost)

Korisnik je dodatno zatražio uvođenje pravog sistema migracija ("kako bih imao bolji i organizovaniji uvid šta je urađeno nad bazom"), umesto dosadašnjeg pristupa — jedan rastući `const schema` string u `db.go`, primenjen u celini na svaki start. Prešlo se na `github.com/pressly/goose/v3`:

- `backend/internal/db/migrations/00001_initial_schema.sql` — kompletan POSTOJEĆI schema string prebačen bez sadržajne izmene (i dalje pun `IF NOT EXISTS`) — bezbedan bootstrap i za postojeću dev bazu (no-op, sve već postoji) i za novu instalaciju.
- `backend/internal/db/migrations/00002_dispatcher_roles.sql` — šema za OVU funkcionalnost, prvi migracioni fajl pisan kao čist (ne-idempotentan) SQL, jer goose sada garantuje jednokratnu primenu svakog fajla.
- `db.go` koristi `//go:embed migrations/*.sql` + `goose.Up(conn, "migrations")` umesto ručnog `ExecContext`.
- **Potvrđeno uživo**: rebuild Docker backend-a protiv postojeće, popunjene dev baze — `00001` primenjen bez greške (0 promena, sve već postojalo), `00002` dodao novu šemu, postojeći vozač/vozilo/status podaci netaknuti.

Ubuduće svaka šematska izmena = nov numerisan `.sql` fajl, ne dopisivanje u string.

## Šema (`00002_dispatcher_roles.sql`)

- `drivers.role` (`'driver'` | `'dispatcher'`, podrazumevano `'driver'`), `drivers.dispatcher_id` (samo-referentni FK, NULL = samostalan vozač).
- `vehicles.dispatcher_id` (uz već postojeći nullable `driver_id`) — tačno jedno od njih postavljeno po vozilu; ta provera je namerno samo u Go handleru, ne DB CHECK (dosledno projektnom stilu).
- `trips.assigned_by_id` — dispečer koji je dodelio turu (NULL za samostalne ture). `trips.driver_id` ostaje DODELJENI vozač kao i dosad, pa sav postojeći kod (ownership provere, WS auth, chat, trip_events) radi bez izmene i za dodeljene ture.
- `dispatcher_requests` — jedina tačka gde se veza dispečer↔vozač uspostavlja (zahtev → odobrenje → `drivers.dispatcher_id` postavljen).

## Tura: offered → start

Nova statusna vrednost `'offered'` (pre `'created'`). Kada dispečer kreira turu (`POST /api/v1/trips` sa `driver_id`), ruta se odmah računa/skoruje (koristeći DISPEČEROV preference profil — `scoringPreferences(ctx, callerID)`, gde je `callerID` uvek pozivalac), ali se **ne** pali `departed` event niti `trip.started` RabbitMQ poruka — worker još ne obrađuje turu. Tek na `POST /api/v1/trips/{id}/start` (vozač, samo dodeljeni), status prelazi u aktivan i pale se isti side-effect-i koji kod samostalne ture pale odmah pri kreiranju (izdvojeno u `startTripSideEffects`, deljeno između oba puta).

**Potvrđeno uživo**: dispečer kreira turu za upravljanog vozača → `GET /trips/{id}/events` prazno → vozač je vidi u `GET /trips?status=offered` → pokreće (`POST /start`) → `departed` event se sada pojavljuje → dispečer se poveže na `/ws/trips/{id}` i prima live pozicije (isti gateway, samo proširena provera vlasništva da uključi `assigned_by_id`).

## Vozila — hibridno vlasništvo

`store.Vehicle.DriverID`/`DispatcherID` su oba `*int64`; tačno jedno postavljeno. `vehicleAccessible(v, account)` (u novom `httpapi/roles.go`) pokriva tri slučaja: sopstveno lično, sopstvena flota (dispečer), ili flota SVOG dispečera (upravljani vozač). `GET /vehicles` za upravljanog vozača vraća UNIJU ličnih i dispečerove flote.

**Potvrđeno uživo**: dispečer kreira flotno vozilo, upravljani vozač kreira lično → `GET /vehicles` kao vozač vraća oba; kao dispečer vraća samo flotu.

## Zahtevi za povezivanje

`store/dispatcher_request.go` — `Create` odbija ako je vozač već upravljan ili već postoji `pending` zahtev od istog dispečera; `Respond` (u transakciji) menja status i, ako je odobreno, postavlja `drivers.dispatcher_id`. Endpoint-i: `GET /dispatcher/available-drivers`, `POST /dispatcher/requests`, `GET /dispatcher/requests`, `GET /driver/requests`, `POST /driver/requests/{id}/respond`.

**Potvrđeno uživo**: dispečer vidi vozača u `available-drivers` → šalje zahtev → vozač ga vidi (sa dispečerovim username-om) u `driver/requests` → odobrava → ponovni login potvrđuje `dispatcher_id` postavljen → vozač NESTAJE iz `available-drivers`, POJAVLJUJE se u `dispatcher/drivers`.

## Flutter

- `register_screen.dart` — `SegmentedButton` Vozač/Dispečer (bez polja za dispečera — veza se uspostavlja tek posle logovanja).
- `dispatcher_requests_screen.dart` (vozač) — pristigli zahtevi, Odobri/Odbij; na odobrenje ažurira `ApiClient.dispatcherId` + `AuthStorage` lokalno i navigira na `OfferedTripsScreen` bez ponovnog logovanja.
- `offered_trips_screen.dart` — home za upravljanog vozača, `GET /trips?status=offered`, tap → `start` → `ActiveTripScreen`.
- `dispatcher_home_screen.dart` / `dispatcher_available_drivers_screen.dart` / `dispatcher_create_trip_screen.dart` / `dispatcher_live_map_screen.dart` — roster, slanje zahteva, kreiranje ture za vozača (vozilo ograničeno na dispečerovu flotu — nema listing endpoint-a za "vozačeva lična vozila" iz dispečerove perspektive, iako backend to tehnički prihvata; namerno ostavljeno za kasnije), live mapa (N paralelnih `TripSocket` konekcija na jednoj `FlutterMap`, isti servis kao `ActiveTripScreen`, bez novog WS koda).
- `entry_router.dart` — `homeScreenFor(api)`, jedna tačka grananja (dispečer/upravljan/samostalan) korišćena i posle login-a/registracije i na cold-start u `main.dart`.
- **Ispravka postojećeg koda**: `ActiveTripScreen` je ranije zahtevao pun `VehicleProfile` kroz konstruktor (iz Nocturne redizajna) — za dodeljenu turu vozač taj objekat nikad nema u ruci (nije prošao kroz `RouteRequestScreen`). Promenjeno da prima `vehicleId` i sâm učita vozilo preko novog `ApiClient.getVehicle(id)`.

**Verifikacija**: `flutter analyze` (0 problema), `flutter test` (uključujući postojeće FAB meni testove) prolaze čisto. Vizuelno testiranje (dispečer/vozač tok na dva uređaja/naloga) ostaje na korisniku.

## Namerni obim-cut-ovi

- Nema "odbij ponudu" akcije za vozača (samo start ili ignorisanje) — lako dodati kasnije.
- Dispečerova mapa uživo prikazuje samo vozače TRENUTNO na aktivnoj turi — nema praćenja pozicije van ture (projekat nema pravu GPS integraciju, samo simulaciju po turi).
- Dispečerski ekran za kreiranje ture nudi samo flotna vozila u padajućem meniju (backend prihvata i vozačevo lično vozilo, ali nema endpoint-a da ga dispečer izlista za drugog vozača).
