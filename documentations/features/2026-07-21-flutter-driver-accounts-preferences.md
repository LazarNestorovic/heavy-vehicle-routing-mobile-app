# Flutter: nalozi, preference, izbor vozila (Faza 6)

**Datum:** 2026-07-21
**Plan:** `/home/lazar/.claude/plans/lovely-cooking-puppy.md` (Faza 6, poslednja)
**Fajlovi:** `mobile/lib/screens/{login,register,vehicle_list,preferences}_screen.dart`, `mobile/lib/services/{api_client,auth_storage,trip_socket}.dart`, `mobile/lib/main.dart`

## Šta je dodato

Dovršava [driver preference sistem](2026-07-21-driver-preference-scoring.md) i [preferirane pumpe](2026-07-21-preferred-fuel-stations.md) na mobilnoj strani:

- **Login/Register ekrani** — sada prvi ekran u app-u (pre je to bio profil vozila).
- **VehicleListScreen** — novi "hub" posle logina: lista vozačevih vozila (1:N), tap → `RouteRequestScreen`, "+" → nova forma za vozilo, ikonica za `PreferencesScreen`, ikonica za odjavu.
- **PreferencesScreen** — 4 slajdera (1-5) za ušteda goriva/osetljivost tovara/udeo auto-puta/brzinu, polje za preferirani brend pumpe, lista sačuvanih omiljenih lokacija (dodaj preko tap-na-mapu ekrana, obriši).
- **VehicleProfileScreen** — pojednostavljen: više nije "prvi ekran" niti čuva lokalno "trenutno" vozilo (`VehicleStorage` obrisan, više ne odgovara 1:N modelu) — čista forma za dodavanje novog vozila koja se vraća na listu.

## VAŽNA RAZLIKA od ranije Flutter faze: ovog puta je stvarno provereno

Za razliku od [prvobitne Flutter faze](2026-07-21-flutter-mobile-app.md) (pisana bez SDK-a, nikad pokrenuta), **Flutter je u međuvremenu postao dostupan u ovoj sesiji** (korisnik ga je instalirao, uspešno pokrenuo aplikaciju na svom fizičkom Android telefonu). Za ovu fazu je stvarno pokrenuto:

```
flutter analyze   → No issues found!
flutter test      → 1/1 passed
```

Ovo je mnogo jača provera od pukog ručnog pregleda koda — hvata prave sintaksne/tipske greške. **I dalje nije vizuelno potvrđeno** (ne mogu da renderujem UI ovde) — korisnik treba da pokrene `flutter run` na svom telefonu i potvrdi da login→lista vozila→preference tok stvarno izgleda i radi kako treba.

## Bitna tehnička izmena: token propagacija kroz ekrane

Ranije je svaki ekran pravio sopstveni `ApiClient()`. Sada, pošto skoro svaki endpoint zahteva JWT (Faza 1), **jedan `ApiClient` sa `token` poljem se pravi jednom u `main()`** (učitava sačuvan token iz `AuthStorage` pre `runApp`) i prosleđuje kroz konstruktore kroz ceo lanac ekrana (`LoginScreen` → `VehicleListScreen` → `RouteRequestScreen` → `ActiveTripScreen`, itd.) — svaki `Navigator.push` sada nosi `api: widget.api`.

**WebSocket takođe zahteva auth**, ali preko `?token=` query parametra, ne header-a (browseri ne mogu da postave custom header na WS handshake — backend `RequireAuthQuery` postoji tačno za ovo, videti [websocket-gateway.md](2026-07-21-websocket-gateway.md)). `TripSocket.connect()` sada prima token kao drugi argument.

## Šta namerno NIJE urađeno

- Nema "zaboravljena lozinka" toka.
- Nema logout sa svih uređaja / server-side token revokacije (isto ograničenje kao backend, videti [driver-preference-scoring.md](2026-07-21-driver-preference-scoring.md)).
- Picker za omiljenu lokaciju je minimalan (tap-na-mapu + ime) — bez pretrage adresa, isti obrazac kao `RouteRequestScreen`.

## Bug pronađen tek na stvarnom uređaju (2026-07-21) — tačno zašto je uređaj-test bio neophodan

Korisnik je odmah po prvom loginu na fizičkom Android telefonu dobio: `setState() callback argument returned Future` u `VehicleListScreen`. Uzrok: `_reload()` je pozivao

```dart
setState(() => _vehiclesFuture = widget.api.listVehicles());
```

U Dart-u, izraz dodele (`a = b`) se evaluira na vrednost `b` — arrow funkcija je time implicitno **vraćala** `Future<List<VehicleProfile>>`. Flutter `State.setState()` eksplicitno baca grešku kad callback vrati `Future` (skoro uvek znak da je neko zaboravio da je funkcija sinhrona). **Ni `flutter analyze` ni `flutter test` ovo nisu uhvatili** — Dart-ov tip sistem dozvoljava non-void povratnu vrednost gde se očekuje `void Function()`, provera je isključivo runtime unutar `setState()`, i widget test nije stvarno pozivao `_reload()` sa živim Future-om. Ispravka: blok telo umesto arrow-a (`setState(() { _vehiclesFuture = ...; })`), čime callback vraća `void`.

Ovo je tačno vrsta greške koju analyzer/test ne mogu da uhvate, a stvarno pokretanje na uređaju može — potvrđuje da je odluka da se ne proglašava "gotovo" bez vizuelne provere bila ispravna.
