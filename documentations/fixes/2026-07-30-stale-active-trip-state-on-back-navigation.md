# Stanje aktivne ture ostajalo zastarelo posle povratka nazad (bez restarta app-a)

**Datum:** 2026-07-30
**Fajlovi:** `mobile/lib/services/route_observer.dart` (nov), `mobile/lib/main.dart`, `mobile/lib/widgets/active_trip_banner.dart`, `mobile/lib/screens/{route_request,vehicle_list,offered_trips}_screen.dart`

Korisnik je prijavio: posle osvežavanja (restarta) aplikacije dok je tura aktivna, sve radi ispravno. Ali ako tu turu završi i pokrene NOVU, pa izađe nazad dugmetom — ekran za pravljenje rute ne zna da je nova tura aktivna (nema blokade), a kad se vrati sve do "Moja vozila", traka za aktivnu turu se uopšte ne prikazuje, i dodir na vozilo ga ponovo vodi na ekran za pravljenje rute (gde TEK TADA piše da ima aktivnu turu). Tek pun restart aplikacije ispravlja stanje.

## Uzrok

`RouteRequestScreen`, `VehicleListScreen`, `OfferedTripsScreen` i `ActiveTripBanner` su svoje "da li postoji aktivna tura" stanje učitavali JEDNOM, u `initState()`. Flutter-ov `Navigator` ne poziva `initState()` ponovo za ekran koji je već postojao u steku i samo je ponovo otkriven pošto se ekran iznad njega ukloni (`pop`) — isti Widget/State objekat ostaje živ. Zato je:

- `RouteRequestScreen` (instanca A) proverila "nema aktivne ture" PRE nego što je tura kroz nju pokrenuta, pa ostala zaglavljena na tom (sad netačnom) stanju kad se vozač vrati na nju iz `ActiveTripScreen`-a.
- `VehicleListScreen` (instanca sa dna steka) proverila "nema aktivne ture" davno pre svega ovoga i nikad se nije osvežila kad se vozač vratio nazad sve do nje — dodir na vozilo je zato otvarao NOVU (svežu) instancu `RouteRequestScreen`-a, koja je TEK TADA prvi put ispravno videla aktivnu turu i prikazala blokadu.

## Popravka

Standardno Flutter rešenje za "osveži me kad se vratim na ovaj ekran": `RouteObserver`/`RouteAware`. Nov `routeObserver` (app-wide `RouteObserver<PageRoute>`, registrovan na `MaterialApp.navigatorObservers` u `main.dart`). Svaki od pogođenih ekrana/widget-a sad:
- U `didChangeDependencies()` se pretplati: `routeObserver.subscribe(this, ModalRoute.of(context) as PageRoute)`.
- U `dispose()` se odjavi: `routeObserver.unsubscribe(this)`.
- Implementira `didPopNext()` (poziva se kad ruta iznad ovog ekrana bude uklonjena i ekran ponovo postane vidljiv) — poziva isto osvežavanje koje se inače dešava u `initState()`.

Pogođeno:
- `ActiveTripBanner` — `_load()` refaktorisan da uslov za dispečera (nema sopstvenih tura) bude UNUTAR `_load()`, ne samo u `initState()`, pa se ista logika ispravno primenjuje i pri `didPopNext()`.
- `RouteRequestScreen` — `didPopNext()` poziva `_checkActiveTrip()`.
- `VehicleListScreen` — `didPopNext()` poziva `_reload()` (osvežava i listu vozila i `_activeTrip` korišćen za tap-rutiranje).
- `OfferedTripsScreen` — `didPopNext()` poziva `_reload()` (ista klasa problema i tu, npr. povratak iz `VehicleListScreen`-a preko menija).

## Verifikacija

`flutter analyze`/`flutter test` čisti. Ovo je čisto Flutter lifecycle ponašanje (ne postoji preko curl-a) — mehanizam je standardan/dobro poznat Flutter obrazac za tačno ovaj problem, ali stvarno ponašanje na uređaju (povratak nazad kroz više ekrana, provera da se traka/blokada ažurira bez restarta) ostaje na korisniku da potvrdi na fizičkom uređaju, isto ograničenje kao za sav raniji UI rad u ovom projektu.
