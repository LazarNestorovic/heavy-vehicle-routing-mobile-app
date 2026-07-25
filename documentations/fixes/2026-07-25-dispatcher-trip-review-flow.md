# Ispravke posle prvog uživo testiranja dispečer/vozač toka

**Datum:** 2026-07-25
**Fajlovi:** `backend/internal/store/trip.go`, `backend/internal/httpapi/trips.go`, `mobile/lib/screens/{route_request,dispatcher_create_trip,offered_trips,dispatcher_home,chat_thread}_screen.dart`, `mobile/lib/screens/{trip_detail,dispatcher_trips}_screen.dart` (nova), `mobile/lib/models/trip.dart`, `mobile/lib/services/api_client.dart`

Korisnik je uživo isprobao dispečer/vozač funkcionalnost (`documentations/features/2026-07-25-dispatcher-driver-roles.md`) i prijavio četiri problema. Jedan (prikaz "zahteva" na glavnom ekranu) je zahtevao pojašnjenje pre popravke — ispostavilo se da nije bag u prikazu zahteva, već nedostatak funkcionalnosti: dodeljena tura se odmah mogla samo pokrenuti, bez uvida u rutu/tovar/vozilo i bez mogućnosti odbijanja.

## 1. Nov korak u statusnoj mašini tura: `accepted`

Dosad: `offered` → (klik "Kreni") → `created`. Sada: `offered` → **`accepted`** (vozač pregleda rutu/tovar/vozilo i prihvati ili odbije) → (tek posle, posebnim klikom) `created`. `rejected` je terminalno stanje za odbijene ponude.

- `store/trip.go` — dodate `TripStatusAccepted`/`TripStatusRejected` konstante, `MarkAccepted`/`MarkRejected`/`MarkStarted` dele zajedničku `markStatusTransition` (uslovni UPDATE, `ErrNotFound` ako trenutni status ne odgovara).
- `httpapi/trips.go` — nova `POST /api/v1/trips/{id}/accept` i `POST /api/v1/trips/{id}/reject`; `handleStartTrip` sada zahteva `accepted` (ne više `offered`) pre pokretanja. Novi `loadOwnTrip` helper (striktna provera — samo dodeljeni vozač, ne i dispečer koji je dodelio, za razliku od `tripAccessible` koji dozvoljava oboje za gledanje/WS).
- **Potvrđeno uživo**: pokušaj `start` na `offered` turi → 409; `accept` → status `accepted`, i dalje nema `departed` eventa; `start` → sada radi, `departed` se pojavljuje; `reject` na drugoj turi → status `rejected`, nestaje iz `?status=offered` liste, naknadni `accept` na njoj → 409.

## 2. Dispečer sada vidi status svake poslate ture

`tripResponse` dobija `driver_id`/`driver_username` (ranije nije bilo nikakvog vozača u odgovoru, iako je `store.Trip.DriverID` uvek postojao). `handleListTrips` obogaćuje `driver_username` samo za dispečera (jedan upit po jedinstvenom vozaču u listi). Novi `screens/dispatcher_trips_screen.dart` — sve ture koje je dispečer dodelio, sa statusnom oznakom (Ponuđena/Prihvaćena/Pokrenuta/Odbijena) i imenom vozača; dostupno preko nove ikonice na `DispatcherHomeScreen`.

## 3. Vozač sada vidi rutu, tovar i vozilo pre nego što odluči

Novi `screens/trip_detail_screen.dart` — statična mapa (`CameraFit.bounds` da se cela ruta uklopi), kartice sa distancom/trajanjem/risk score-om, tovarom (ako postoji), i vozilom (učitanim preko `ApiClient.getVehicle`). Dugmad se menjaju po statusu: `offered` → [Odbij] [Prihvati]; `accepted` → [Kreni] (koji tek tada zove `startTrip` i vodi na `ActiveTripScreen`). `OfferedTripsScreen` sad povlači i `offered` i `accepted` ture (dva paralelna `listMyTrips` poziva) i za obe navigira na detalje umesto direktnog dugmeta "Kreni" na listi.

## 4. Tastatura je prekrivala polja za tovar

`RouteRequestScreen`/`DispatcherCreateTripScreen`: donji deo ekrana (greška, pregled rute, forma za tovar, dugmad) je bio u istom, neskrolabilnom `Column`-u kao mapa — fokusirano polje se nije moglo automatski skrolovati iznad tastature. Rešeno razdvajanjem na dva `Expanded` regiona (mapa `flex: 3`, donji deo `flex: 2` unutar sopstvenog `SingleChildScrollView`) — mapa zadržava svoj prostor za gestove (bez konflikta sa skrolovanjem strane), a fokusirano polje se sad automatski skroluje u donjem regionu (Flutter-ov podrazumevani `ensureVisible` na fokus, radi samo unutar `Scrollable` pretka).

## 5. Duplirane poruke u čet-u

Uzrok: `chat.<min>.<max>` routing key ne razlikuje smer, pa i POŠILJAOČEVA sopstvena otvorena WS konekcija (na `/ws/chats/{counterpartId}`) prima nazad poruku koju je sam poslao — `_send()` je već optimistički dodavao poruku lokalno, pa se ista poruka pojavljivala i drugi put kroz `_listen()`. Ispravljeno: zajednička `_addMessage()` metoda proverava `id` pre dodavanja (koriste je i `_send()` i `_listen()`) — poruka se prikaže tačno jednom bez obzira koji put stigne prvi.

## Verifikacija

Backend: `go build`/`vet`/`test` čisti, uživo potvrđen ceo `offered→accept→start`/`reject` tok preko curl-a (uključujući 409 provere na pogrešnim prelazima) i `driver_username` obogaćivanje na dispečerovoj listi. Flutter: `flutter analyze` (0 problema) i `flutter test` (uključujući postojeće FAB meni testove) prolaze čisto. Vizuelno testiranje (izgled mape na `trip_detail_screen`, tastatura preko formi, dupliranje u čet-u) ostaje na korisniku za potvrdu na uređaju.
