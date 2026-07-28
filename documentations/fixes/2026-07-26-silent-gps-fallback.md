# "Kreni na put" i dalje simulira kretanje umesto pravog GPS-a

**Datum:** 2026-07-26
**Fajlovi:** `mobile/lib/services/location_service.dart`, `mobile/lib/screens/active_trip_screen.dart`

Korisnik je prijavio da posle "Kreni na put" mapa i dalje prikazuje simulirano kretanje, ne stvarnu GPS poziciju.

## Uzrok — tih fallback bez ikakve poruke

`_startLiveGps()` (Blok D, `2026-07-26-live-gps-tracking.md`) je namerno projektovan da NE prekine tok kad dozvola za lokaciju nije data — pada nazad na postojeću simulaciju. Problem je što je taj fallback bio potpuno TIH: `LocationService.ensurePermission()` je vraćao samo `bool`, pa aplikacija nije imala način da vozaču kaže ZAŠTO se koristi simulacija (GPS isključen na telefonu? dozvola odbijena? trajno odbijena od ranijeg testiranja, pa se dijalog više i ne pojavljuje?). Sa korisnikove strane, sve tri situacije izgledaju identično kao "app uopšte ne koristi pravi GPS" — što je tačno ono što je prijavljeno.

## Popravka

`LocationService.ensurePermission()` sad vraća `GpsStatus` enum (`granted` / `serviceDisabled` / `denied` / `deniedForever`) umesto golog bool-a. `ActiveTripScreen` prikazuje traku upozorenja (isti obrazac kao `EmailVerificationBanner`) kad status nije `granted`, sa porukom I dugmetom za rešavanje:
- **GPS isključen na telefonu** → "Uključi lokaciju" (`Geolocator.openLocationSettings()`)
- **Dozvola trajno odbijena** (Android više ni ne prikazuje dijalog) → "Podešavanja" (`Geolocator.openAppSettings()`)
- **Dozvola upravo odbijena** → "Pokušaj ponovo" (ponovo poziva `_startLiveGps()`, što će Android-u dati priliku da ponovo prikaže dijalog ako nije trajno odbijena)

Ovo ne menja SAMU logiku praćenja (i dalje: prvi pravi GPS ping prebacuje WS gateway iz simulacije u live režim) — samo čini fallback vidljivim i rešivim iz same aplikacije, umesto da vozač nagađa zašto mapa "samo simulira".

## Verifikacija

`flutter analyze`/`flutter test` čisti. Ovo je čisto Flutter izmena (backend nepromenjen, mehanizam prebacivanja simulacija→live iz Bloka D ostaje isti) — koji se TAČNO status prikazuje (isključen GPS vs trajno odbijena dozvola vs upravo odbijena) zavisi od stanja na korisnikovom telefonu i ne može se proveriti bez fizičkog uređaja. Sledeći test na uređaju treba da pokaže traku sa konkretnim razlogom umesto tihe simulacije.
