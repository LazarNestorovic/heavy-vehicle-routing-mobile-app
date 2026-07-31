# Automatsko prepoznavanje skretanja sa rute + automatska nova ruta

**Datum:** 2026-07-31
**Fajlovi:** `backend/internal/store/trip.go`, `backend/internal/httpapi/trips.go`, `mobile/lib/services/api_client.dart`, `mobile/lib/screens/active_trip_screen.dart`

Ako vozač na svoju ruku skrene sa planirane rute (drugi izlaz, zaobilazak, greška), aplikacija sama izračuna i primeni novu najbolju rutu od njegove TRENUTNE pozicije do ISTOG odredišta — ne ostavlja ga da gleda liniju na mapi koju više ne prati.

## Mehanizam

**Otkrivanje skretanja** (`ActiveTripScreen`, klijentska strana) — na svaki novi GPS fix (isti `positionStream` koji već izveštava poziciju backend-u), računa se najmanje rastojanje od trenutne pozicije do bilo koje tačke dekodirane rute (`_distanceToRouteMeters` — aproksimacija preko najbliže tačke, ne prava projekcija na segment; ruta iz Valhalla-e je dovoljno gusta da razlika nije bitna za ovu svrhu). Prag **300m** (`_offRouteThresholdMeters`) — heuristika u istom stilu kao `StartProximityStatus`-ov 500m prag, dovoljno velika da uobičajena GPS greška/širina puta ne izazove lažno pozitivan signal, dovoljno mala da stvarno skretanje bude uhvaćeno relativno brzo.

**Potpuno automatsko** — čim se pređe prag, `_reroute()` se pokreće bez ikakve potvrde vozača (vožnja dalje uz rutu koja više ne odgovara stvarnosti nije prava alternativa). Dok traje preračunavanje (~1-2s), prikazuje se kratka, neinteraktivna traka "Preračunavam rutu..." da vozač ne pomisli da je aplikacija zaglavila. Kad se završi, prikazuje se `explanation` nove rute ako postoji, inače generička poruka "Skrenuli ste sa rute - ruta je automatski preračunata.". `_rerouting` je jedina zaštita od ponovnog okidanja — dok je `true`, provere se preskaču, a čim nova ruta uspešno stigne, ona POČINJE tačno na vozačevoj trenutnoj poziciji, pa je sledeća provera prirodno "na ruti" bez potrebe za dodatnim stanjem.

**Backend — nov `POST /api/v1/trips/{id}/reroute`** (`{origin: {lat, lon}}`, samo vozač, samo dok je tura aktivna):
- Ponovo koristi TAČNO isti scoring pipeline kao kreiranje ture (`bestRoute`, iste preference/omiljene stanice/objašnjenje) — polazište je novo (vozačeva trenutna pozicija), odredište ostaje NEPROMENJENO (originalno odredište ture).
- `TripStore.Reroute()` upisuje novu rutu (shape/distance/duration/risk/explanation) i **briše staru preporuku za pauzu** (računata je za PUT koji se možda više uopšte ne vozi).
- Ponovo objavljuje `trip.started` na RabbitMQ — isti postojeći worker koji je izračunao originalnu preporuku pauze sad to uradi iznova za NOVU rutu, bez dupliranja te logike u reroute handler-u.
- Upisuje nov `trip_events` zapis tipa `"rerouted"` — vidljivo u Dnevniku putovanja pored postojećih `departed`/`arrived`/`rest_stop_suggested`.

## Namerni obim-cut

- Prag od 300m nije kalibrisan na stvarnim podacima — heuristika, isti stil kao svaki drugi prag u projektu.
- Provera je jednostavna "najbliža tačka rute", ne prava geometrijska projekcija na segment — dovoljno tačno za "jesam li očigledno skrenuo", ne za nešto precizno.
- Ako `POST .../reroute` ne uspe (mreža, itd.), samo se prikaže greška preko SnackBar-a — sledeći GPS fix (20m kretanja kasnije) će pokušati ponovo pošto je vozač i dalje van praga, bez posebne retry logike.

## Verifikacija

`go build`/`vet`/`test` i `flutter analyze`/`flutter test` čisti. Uživo (curl): tura kreirana (Beograd→Novi Sad, 92.4km), `POST .../reroute` sa novom pozicijom bliže odredištu vraća preračunatu rutu (47.7km) sa `status: in_progress`; `trip_events` ispravno pokazuje `departed` pa `rerouted`. Stvarno otkrivanje skretanja na mapi (GPS, prag, traka) zahteva fizički uređaj i stvarno kretanje — isto ograničenje kao za sav raniji GPS-zavisan rad.
