# Flutter mobilna aplikacija (3.9)

**Datum:** 2026-07-21
**Fajlovi:** [`mobile/`](../../mobile/)

## Šta je dodato

Implementacija `SPECIFIKACIJA.md` sekcije 3.9 — tri ekrana iz plana (profil vozila → mapa/zahtev za rutu → aktivna vožnja), povezana na sve backend funkcionalnosti izgrađene u ovoj sesiji (`/routes`, `/vehicles`, `/trips`, `/ws/trips/{id}`).

```
mobile/
├── pubspec.yaml                        — http, web_socket_channel, flutter_map, latlong2, shared_preferences
└── lib/
    ├── main.dart                       — MaterialApp, početni ekran
    ├── config.dart                     — apiBaseUrl/wsBaseUrl (platform-zavisno, vidi guide)
    ├── models/                         — VehicleProfile, RouteResult, RouteCandidate, Trip, RestStop, PositionUpdate
    ├── services/
    │   ├── api_client.dart             — REST pozivi (createVehicle, previewRoute, createTrip, getTrip)
    │   ├── trip_socket.dart            — WebSocket kao Stream<PositionUpdate>
    │   ├── polyline.dart               — polyline6 dekoder (Dart port backend/internal/valhalla/polyline.go)
    │   └── vehicle_storage.dart        — lokalno čuvanje profila (shared_preferences, bez auth-a)
    └── screens/
        ├── vehicle_profile_screen.dart — forma profila, POST /api/v1/vehicles
        ├── route_request_screen.dart   — mapa (tap-to-pick), POST /api/v1/routes i /trips
        └── active_trip_screen.dart     — live pozicija preko WS, ETA, rest-stop alert, explanation banner
```

## VAŽNO: kod nije pokrenut niti vizuelno proveren

Flutter/Dart SDK nije instaliran u okruženju gde je ovaj kod pisan (nema `flutter`/`dart` komande, nema Android emulatora). Korisnik je eksplicitno odabrao opciju **"Napravi kod, ti testiraš lokalno"** kad je ovo postavljeno kao pitanje — kod je pisan pažljivo (ručno pregledan red-po-red za Dart sintaksu, sve `flutter_map`/paket verzije proverene protiv pub.dev API-ja i CHANGELOG-a za breaking promene), ali **nijedan ekran nije stvarno renderovan, nijedan HTTP/WS poziv iz aplikacije nije stvarno izveden**. Prvo pravo pokretanje mora biti na mašini sa Flutter SDK-om — vidi [`documentations/guides/run-flutter-app.md`](../guides/run-flutter-app.md).

Ovo je direktno kršenje uobičajene prakse "pokreni i vizuelno proveri UI pre nego što javiš da je gotovo" — namerno flagovano ovde umesto prećutano, jer je transparentnost o ovome bitnija od izgleda završenosti.

## Arhitektonske odluke

- **Bez state management biblioteke** (Provider/Riverpod/Bloc) — samo `StatefulWidget`/`setState`. Tri ekrana sa linearnim tokom (profil → mapa → aktivno putovanje) ne opravdavaju dodatnu zavisnost.
- **Bez geokodiranja/pretrage adresa** — polazak/odredište se biraju dodirom na mapu (prvi dodir = polazak, drugi = odredište, treći počinje ispočetka). Dodavanje pretrage adresa bi zahtevalo eksterni geocoding servis, van obima MVP-a.
- **Bez autentifikacije** — profil vozila se čuva lokalno (`shared_preferences`) i šalje kao prost `vehicle_id` bez ikakve zaštite, u skladu sa `SPECIFIKACIJA.md` odlukom da MVP ne zahteva multi-usera.
- **`polyline.dart` je ručni Dart port** Go dekodera iz `backend/internal/valhalla/polyline.go` — namerno bez eksterne polyline biblioteke, isti razlog kao na backend-u (algoritam je mali i stabilan, dodatna zavisnost nije vredna toga).

## Šta namerno NIJE urađeno

- Nema pravog GPS-a — aktivni ekran prikazuje **simuliranu** poziciju koju gura WebSocket gateway (vidi [websocket-gateway.md](../features/2026-07-21-websocket-gateway.md)), ne stvarnu lokaciju telefona.
- Nema offline moda (mapa zahteva internet za OSM tile-ove) — future work po `SPECIFIKACIJA.md`.
- Nema prikaza alternativnih ruta (`candidates` niz iz API odgovora) na mapi — samo izabrana ruta se crta; podaci postoje u modelu (`RouteResult.candidates`) za budući rad.
- Nema testova (`flutter test`) — nije mogao da se pokrene `flutter test` bez SDK-a; ovo je priznata praznina, ne prećutana.

## Sledeći koraci (kad se pokrene na mašini sa Flutter-om)

1. Proći kroz [`run-flutter-app.md`](../guides/run-flutter-app.md) i prijaviti šta ne radi.
2. Ispraviti bilo koje sintaksne/API greške koje `flutter analyze`/`flutter run` otkrije (očekivano da postoji poneka sitnica — kod nikad nije prošao kroz pravi kompajler).
3. Dodati `flutter test` pokrivenost bar za `polyline.dart` (čista funkcija, laka za unit test, direktno uporediva sa Go verzijom).
