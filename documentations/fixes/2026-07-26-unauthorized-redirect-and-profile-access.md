# Odjava sa svih uređaja nije vraćala na login + profil nedostupan bez aktivne ture

**Datum:** 2026-07-26
**Fajlovi:** `mobile/lib/services/api_client.dart`, `mobile/lib/main.dart`, `mobile/lib/screens/{entry_router,profile,vehicle_list,offered_trips,dispatcher_home}_screen.dart`

Korisnik je uživo testirao "Odjavi sve uređaje" (`2026-07-26-scope-cut-closures.md` Blok F) sa dva uređaja i prijavio dva problema.

## 1. Drugi uređaj je samo prikazivao grešku, ne vraćao na login

Kad je token na jednom uređaju invalidiran (bilo `token_version` neslaganje posle logout-all, bilo isticanje), backend ispravno vraća `401 "invalid or expired token"` — ali app na TOM uređaju je taj odgovor tretirao kao običnu `ApiException` na ekranu gde se desio, bez ikakve globalne reakcije. Korisnik je ostajao "zaglavljen" na ekranu sa greškom, jedini izlaz je bio ručno zatvoriti i ponovo pokrenuti app.

**Popravka**: `ApiClient` sad ima `onUnauthorized` callback i unutrašnji HTTP klijent (`_AuthAwareClient extends http.BaseClient`) koji presreće SVAKI odgovor — ako je `401` I zahtev je nosio `Authorization` zaglavlje, poziva `onUnauthorized`. Ovaj drugi uslov je bitan: `login`/`register`/`signInWithGoogle`/`forgotPassword` NIKAD ne šalju `Authorization` (nema još sesije), pa pogrešna lozinka na login ekranu i dalje ostaje obična greška na tom ekranu, ne globalna odjava.

`main.dart` sad ima `navigatorKey` (globalni, da bi `onUnauthorized` mogao da navigira bez sopstvenog `BuildContext`-a — 401 može stići sa bilo kog ekrana) i postavlja `api.onUnauthorized` da očisti sesiju i vrati na `LoginScreen`, sa guard-om (`handlingUnauthorized` flag) protiv višestrukog okidanja ako više paralelnih poziva istovremeno dobije 401.

Logika čišćenja sesije (7 istih linija ponovljenih u 4 ekrana) izdvojena u `entry_router.dart` → `clearSession(api)`, deljenu i sa svakim ekranovim "Odjava" dugmetom i sa novim globalnim handler-om.

## 2. Ekran profila dostupan samo preko aktivne ture

`ProfileScreen` (gde je Blok F dodao "Odjavi sve uređaje") bio je dostupan JEDINO preko `RadialFabMenu`-a na `ActiveTripScreen`-u — vozač bez pokrenute ture (ili dispečer, koji tu turu nikad ne "vozi") nije imao nikakav put do sopstvenog profila.

**Popravka**: dodata ikonica "Profil" u AppBar sva tri home ekrana (`VehicleListScreen`, `OfferedTripsScreen`, `DispatcherHomeScreen`), isti obrazac kao postojeće ikonice (Preference/Zahtevi dispečera/Odjava).

## Verifikacija

`flutter analyze`/`flutter test` čisti. Logička provera da diskriminator (`Authorization` zaglavlje) ispravno razdvaja "sesija je invalidirana" od "pogrešna lozinka pri loginu" — nijedan od login/register/google/forgot-password poziva ne šalje to zaglavlje. Puna provera sa dva fizička uređaja (jedan aktivan, drugi triggeruje logout-all, prvi treba da se automatski vrati na login pri sledećem API pozivu) ostaje na korisniku.
