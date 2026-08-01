# "Sve ture" — dispečerova verzija liste tura sa tri taba

**Datum:** 2026-08-01
**Fajlovi:** `mobile/lib/screens/dispatcher_trip_list_screen.dart` (nov, zamenjuje `dispatcher_trips_screen.dart`), `mobile/lib/screens/dispatcher_home_screen.dart`, `mobile/lib/screens/trip_list_screen.dart`

Dopuna [`2026-07-31-trip-list-screen.md`](2026-07-31-trip-list-screen.md) (vozačka "Moje ture") — isti tri-tab obrazac sad i za dispečera, umesto starog `DispatcherTripsScreen`-a koji je prikazivao SVE dodeljene ture u jednoj neskrolabilnoj/nefiltriranoj listi.

## Zašto ne isto kao vozačka verzija

Dispečer ne vozi, pa dva mesta moraju da se razlikuju od `TripListScreen`-a:

- **Pokrenute** — dispečer može istovremeno imati VIŠE aktivnih tura (po jednu za svakog vozača na putu), za razliku od vozača kome backend garantuje najviše jednu (`HasActiveTrip`). Dodir na stavku zato ne vodi na `ActiveTripScreen` (ta mapa čita GPS SOPSTVENOG telefona kao poziciju vozila — za dispečera bi to bilo pogrešno), već na već postojeći `DispatcherLiveMapScreen` (agregirana, read-only mapa svih aktivnih tura preko WS-a).
- **Predstojeće** i **Završene** — dodir vodi na `TripLogScreen` (vremenska linija `departed`/`rest_stop_suggested`/`rerouted`/`arrived`). Bezbedno za bilo koji status/ulogu (`tripAccessible()` na backend-u već dozvoljava i dodeljenog dispečera, ne samo vozača) — za razliku od `TripDetailScreen`, koji ima dugmad "Prihvati/Odbij/Kreni" smislenu samo za vozača koji stvarno vozi tu turu.
- Stavke prikazuju `driverUsername` (dispečer upravlja više vozača, mora da zna čija je koja tura) — vozačka verzija to nije trebalo.

## Statusi po tabu

- **Pokrenute**: `created`/`in_progress`.
- **Predstojeće**: `offered`/`accepted`.
- **Završene**: `completed` **i** `rejected` — obe su terminalne, ništa se više ne dešava. Za dispečera je odbijena ponuda posebno bitna (signal da turu treba ponuditi drugom vozaču), pa nema smisla da nestane bez traga kao ranije.

## Dopuna vozačkoj verziji: odbijene ture

Dok sam radio na ovome, primetio sam da ni vozačka `TripListScreen` (od juče) nije nigde prikazivala `rejected` ture — ni ranije `OfferedTripsScreen` to nije radio. Dodato u oba: Završene tab sad dohvata `completed` + `rejected` (isti "dohvati svaki status, spoji" obrazac), sa zasebnom ikonicom/bojom za odbijenu turu.

## Verifikacija

`flutter analyze`/`flutter test` čisti. Backend nepromenjen (postojeći `GET /trips?status=` i `tripAccessible()` dovoljni). Stvarna navigacija (tabovi, sortiranje, dodir na svaki tab) zahteva fizički uređaj — isto ograničenje kao za sav raniji UI rad u projektu.
