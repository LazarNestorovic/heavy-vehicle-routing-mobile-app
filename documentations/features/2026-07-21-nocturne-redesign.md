# Nocturne redizajn + puna funkcionalnost iz mock-upa — Blok A (backend)

**Datum:** 2026-07-24
**Plan:** `/home/lazar/.claude/plans/lovely-cooking-puppy.md` (plan mode, odobren pre implementacije)
**Fajlovi:** `backend/internal/db/db.go`, `backend/internal/store/{vehicle,trip,trip_event,chat_message,driver}.go`, `backend/internal/httpapi/{server,vehicles,trips,chat}.go`, `backend/internal/queue/{queue,events}.go`, `backend/internal/ws/{gateway,chat_gateway}.go`, `backend/internal/worker/trip_worker.go`, `backend/main.go`

## Zašto

Korisnik je napravio Claude Design mock-up ("Cargo Routing App", Nocturne varijanta) koji pored vizuelnog stila uvodi četiri nova ekrana: Truck Status (gorivo/servis/radni sati), Cargo (podaci o tovaru), Trip Log (istorija dogadjaja) i Team Chat. Korisnik je eksplicitno tražio **pun set ekrana**, ne dekorativni presvlak — svaki od njih mora biti podržan stvarnim backend podacima. Ovaj zapis pokriva Blok A (backend); Blok B (Flutter tema + ekrani + FAB meni) je zaseban, naredni korak istog plana.

## Šema (`db.go`)

- `vehicles.fuel_percent` (default 100), `vehicles.next_service_km` — **ručno podesivi od strane vozača**, ne senzorski. Projekat nema telematiku, isto pošteno ograničenje kao ostatak sistema (npr. `DrivingHours` nije prava AETR evidencija, već izvedena iz sopstvenih trip-ova/trip_events vozila).
- `trips.cargo_description/cargo_weight_kg/cargo_temp_range/pickup_location/dropoff_location` — opciona, unose se pri kreiranju rute.
- `trip_events` (trip_id, event_type, description, occurred_at) — timeline za Trip Log ekran. `event_type` je jedan od `departed` | `rest_stop_suggested` | `arrived`.
- `chat_messages` (from_driver_id, to_driver_id, body, sent_at, read_at) — jednostavan 1:1 chat, bez fleet/team koncepta (svaki registrovan vozač je validan kontakt).

## Novi/prošireni endpoint-i

| Endpoint | Svrha |
|---|---|
| `GET /api/v1/drivers` | Kontakt lista za chat (svi vozači osim pozivaoca) |
| `PATCH /api/v1/vehicles/{id}/status` | Ručni upis fuel_percent/next_service_km |
| `GET /api/v1/vehicles/{id}/hours` | `since_last_break_min` (od poslednjeg `departed`/`rest_stop_suggested` eventa), `driving_today_min` (suma trajanja trip-ova danas) |
| `GET /api/v1/trips/{id}/events` | Trip Log timeline |
| `POST /api/v1/trips` | Prošireno opcionim cargo/pickup/dropoff poljima |
| `GET /api/v1/chats` | Lista razgovora (poslednja poruka + broj nepročitanih po sagovorniku, jedan agregatni upit sa 3 CTE-a) |
| `GET /api/v1/chats/{driverId}/messages` | Istorija + automatski `MarkRead` |
| `POST /api/v1/chats/{driverId}/messages` | Upis + publikovanje na `chat.events` |
| `GET /ws/chats/{counterpartId}` | Live isporuka novih poruka |

Sve rute su iza `RequireAuth` (WS iza `RequireAuthQuery`, isti obrazac kao `/ws/trips/{id}`).

## Chat preko RabbitMQ — nova arhitektura, ne samo novi routing key

Postojeći `queue.Client` je imao **jedan** exchange (`trip.events`) i durable, po-ulozi imenovane queue-ove (jedan worker, jedna svrha). Chat je drugačiji problem: potencijalno mnogo istovremenih WS konekcija, svaka zainteresovana samo za jedan razgovor, bez potrebe da "sačeka" poruke poslate dok je diskonektovana (REST `ListThread` je već trajni izvor istine za to).

Rešeno dodavanjem:
- **`chat.events`** — drugi topic exchange, deklarisan uz postojeći u `queue.Connect`.
- **`ChatRoutingKey(a, b)`** — `chat.<min>.<max>`, isti bez obzira ko šalje, tako da jedan binding pokriva ceo razgovor u oba smera.
- **`ConsumeChatEphemeral`** — za razliku od `Consume` (durable, imenovan queue za worker), ovo pravi **anoniman, exclusive, auto-delete** queue po WS konekciji na sopstvenom `amqp.Channel`-u (Channel nije bezbedan za konkurentno korišćenje izmedju gorutina, pa svaka WS konekcija dobija svoj).
- `ws/chat_gateway.go` — pattern koji do sada nije postojao u projektu: pisanje u WS ide u posebnoj gorutini koja čita iz RabbitMQ delivery kanala, dok glavna gorutina samo blokira na `conn.ReadMessage()` da detektuje disconnect klijenta (klijent ništa smisleno ne šalje preko ovog socket-a — slanje ide preko REST POST-a).

## Wiring dogadjaja u postojeći tok

- `handleCreateTrip` (nakon `Trips.Create`) → upisuje `departed`.
- `trip_worker.go` (nakon `UpdateAfterProcessing`, samo ako je `rest.AfterMinutes != nil`) → upisuje `rest_stop_suggested`.
- `ws/gateway.go simulate()` (kad `status=="arrived"`) → upisuje `arrived`. Gateway je dobio novo `TripEvents *store.TripEventStore` polje.

## Verifikacija uživo (curl + gorilla/websocket test klijent)

Docker image rebuild-ovan (`docker compose up -d --build backend`), migracija prošla bez greške. Testirano na živom stack-u:

1. Dva vozača registrovana/ulogovana, `GET /drivers` vraća kontakt listu.
2. Vozilo kreirano, `PATCH .../status` → `GET /vehicles/{id}` potvrđuje `fuel_percent`/`next_service_km` perzistovani.
3. Trip kreiran sa cargo/pickup/dropoff poljima (Radalj→Klisa, isti par koordinata kao originalni bug report) — sva polja se vraćaju u odgovoru i u `GET /trips/{id}`.
4. `GET /trips/{id}/events` odmah nakon kreiranja pokazuje `departed`; nakon što worker obradi (trajanje 87min < 270min prag) nema `rest_stop_suggested` — ispravno.
5. Duža ruta (Subotica→Vranje, 315min) → `rest_stop_suggested` se pojavljuje sa tačnim opisom praga.
6. `GET /vehicles/{id}/hours` menja se posle kreiranja trip-a (`driving_today_min` raste, `since_last_break_min` kratko posle `departed` eventa).
7. Chat: A šalje B, B šalje A, `GET /chats` pokazuje `unread_count`, `GET /chats/{id}/messages` ga nulira.
8. **Live WS isporuka**: pomoćni Go klijent (gorilla/websocket, pisan samo za ovaj test, van repozitorijuma) konektovan kao driver B na `/ws/chats/6`; dok je konektovan, driver A šalje poruku preko REST-a — B je prima uživo preko socket-a, potvrđujući da `chat.events` exchange + `ConsumeChatEphemeral` rade end-to-end.
9. `/ws/trips/{id}` odigran do kraja (60s simulacija) → `status:"arrived"` primljen, `GET /trips/{id}/events` pokazuje i `departed` i `arrived`.

Nijedan deo nije mokovan — svi testovi idu kroz stvarni Postgres/RabbitMQ/backend kontejner, isti standard kao ostatak projekta.

## Blok B (Flutter)

### B1/B5 — Nocturne tema i reskin

`mobile/lib/theme/nocturne_theme.dart` mapira `systems/nocturne/styles.css` tokene u jedan `ThemeData` (`google_fonts` v8.2.0 za Inter, `ColorScheme.fromSeed` sa `--color-accent` kao primary, tamna pozadina/površina/tekst tokeni direktno). Primenjeno u `main.dart` preko `MaterialApp(theme:)` — pokriva većinu postojećih ekrana automatski. Ručno zamenjeni hardkodovani `Colors.indigo/red/orange` pozivi u šest ekrana (`active_trip`, `route_request`, `preferences`, `login`, `register`, `vehicle_profile`) sa `NocturneColors.*` — mapa/marker pinovi (zeleno poreklo, crveno odredište, žuta zvezda za omiljeno) namerno ostavljeni kao univerzalne mapske konvencije, ne deo teme.

### B2 — Novi servisi i modeli

`services/chat_socket.dart` (isti obrazac kao `trip_socket.dart`, receive-only — slanje ide preko REST-a). `api_client.dart` prošireno sa `listDrivers/updateVehicleStatus/getVehicleHours/listTripEvents/listChats/getChatMessages/sendChatMessage`, `createTrip` prima opciona cargo/pickup/dropoff polja. Novi modeli: `vehicle_hours.dart`, `trip_event.dart`, `driver.dart`, `chat_message.dart` (+ `ChatConversation`), `Trip`/`VehicleProfile` prošireni.

Pošto ne postoji `GET /api/v1/me`, `username` se ne dobija sa servera nakon cold-starta — čuva se lokalno (`AuthStorage`, uz token/driverId) od trenutka kada ga vozač otkuca na login/register ekranu, i drži se na `ApiClient.username/driverId` za sve ekrane koji ga koriste (Profile, chat bubble poravnanje po `fromDriverId == api.driverId`).

### B3 — Novi ekrani

`profile_screen.dart`, `truck_status_screen.dart` (gorivo/servis sa inline izmenom preko dijaloga + `PATCH .../status`, radni sati, "rest break recommended" baner koji **ponovo koristi** trenutni trip-ov `next_rest_suggestion_min`/`rest_stop`, ne nov mehanizam), `cargo_screen.dart` (read-only prikaz), `trip_log_screen.dart`, `chat_list_screen.dart` + `chat_thread_screen.dart`. Cargo unos dodat kao `ExpansionTile` na `RouteRequestScreen` pre "Kreni na put" (opciono, prazna polja se ne šalju).

### B4 — Draggable radijalni FAB meni (`widgets/radial_fab_menu.dart`)

Direktan port mock-up skripte: `CORNERS` tabela (4 ćoška, isti h/v predznaci za `arcOffset`), `arcOffset(angle, r, h, v)` formula, 5 pod-dugmadi na uglovima 78/60/42/24/6°, radijus 100px, `AnimatedPositioned` sa `Cubic(0.34, 1.3, 0.64, 1.0)` (ekvivalent CSS `cubic-bezier(.34,1.3,.64,1)` — y1>1 daje isti blagi "bounce" overshoot), tap-vs-drag razlikovanje preko pomeraja praga (6px slop, mock-up koristi "bilo koji pokret" ali to je nepouzdano na touch uređajima pa je dodat prag).

**Uhvaćen i ispravljen pravi bug tokom pisanja widget testa** (ne samo vizuelni nedostatak): `DragUpdateDetails.localPosition` je lokalan za render box na koji je gesture recognizer okačen — ovde mali 58px FAB koji se sam pomera svaki frejm tokom prevlačenja — korišćenje te vrednosti kao da je stabilna koordinata unutar celog ekrana bi na pravom uređaju pravilo nasumično/nekorektno prevlačenje. Ispravljeno konvertovanjem `globalPosition` u koordinate fiksnog kontejnera (`Stack` sa `GlobalKey`, `RenderBox.globalToLocal`) pre svakog poređenja. Ovo je uhvaćeno testom (`test/radial_fab_menu_test.dart`), ne uživo — dobar primer da widget testovi ipak hvataju neke klase gestural bagova, iako fino podešavanje "osećaja" i dalje zahteva fizički uređaj.

Ugrađen u `ActiveTripScreen` (postaje mock-upov "Main" ekran) kao `Stack` preko mape, zamenjujući privremenu AppBar navigaciju iz B3. `vehicle` se prosleđuje kroz `RouteRequestScreen` (već ga ima) do `ActiveTripScreen` da bi Truck Status mogao da prikaže gorivo/servis bez novog API poziva.

**Verifikacija:** `flutter analyze` (0 problema) i `flutter test` (uključujući 3 nova testa za FAB meni: otvaranje tapom, dispatch pod-dugmeta, prevlačenje NE otvara meni) prolaze čisto. Vizuelno/gesture "osećaj" (brzina animacije, veličina hitbox-a) ostaje na korisniku za proveru na fizičkom uređaju — ista napomena kao u odobrenom planu.

## Sledeći korak

Nema — Blok A i Blok B su kompletni prema odobrenom planu. Preostaje samo korisnikova provera na fizičkom uređaju (vizuelni izgled, osećaj FAB prevlačenja, chat u dva prozora).
