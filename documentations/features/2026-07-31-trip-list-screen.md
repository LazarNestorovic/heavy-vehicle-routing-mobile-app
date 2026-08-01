# "Moje ture" — lista tura sa tri taba (pokrenute/predstojeće/završene)

**Datum:** 2026-07-31
**Fajlovi:** `mobile/lib/models/trip.dart`, `mobile/lib/screens/trip_list_screen.dart` (nov), `mobile/lib/screens/{vehicle_list,offered_trips}_screen.dart`

Nijedan tip vozača (samostalan ni upravljan) do sada nije imao NIKAKAV način da vidi istoriju svojih ZAVRŠENIH tura — jedina vidljivost aktivne ture bila je `ActiveTripBanner` (najviše jedna, pošto backend dozvoljava samo jednu aktivnu turu odjednom). Ovo zatvara tu prazninu, standardnim obrascem za ovakav ekran: tri taba.

## Tri taba

- **Pokrenute** — status `created`/`in_progress`. Najviše jedna stavka (backend garantuje preko `HasActiveTrip`). Dodir odmah otvara `ActiveTripScreen` (live mapa).
- **Predstojeće** — status `offered`/`accepted`. Uvek prazno za samostalnog vozača (kod njega tura ide pravo u `created`, nema faze ponude/prihvatanja) — smisleno samo za upravljanog vozača, isti sadržaj koji `OfferedTripsScreen` već prikazuje kao svoju glavnu listu. Dodir otvara `TripDetailScreen` (isto odredište kao i sa `OfferedTripsScreen`-a).
- **Završene** — status `completed`. Jedino mesto u aplikaciji gde se uopšte vidi istorija završenih tura. Dodir otvara `TripLogScreen` (već postojeći, prikazuje `departed`/`rest_stop_suggested`/`rerouted`/`arrived` vremensku liniju te ture).

## Sortiranje (samo Završene)

Padajući meni: Datum (najnovije/najstarije), Rastojanje (najduže/najkraće). Potpuno na klijentskoj strani (`_sortCompleted`) — lista je u obimu teze mala (desetine, ne hiljade tura), nema potrebe za backend sort parametrom.

## Dohvatanje podataka

`GET /trips?status=X` već postoji i filtrira po TAČNO jednom statusu — isti obrazac kao `ApiClient.findActiveTrip()` (dva/tri poziva paralelno preko `Future.wait`, pa spajanje rezultata) ponovo iskorišćen ovde za "Pokrenute" (created+in_progress) i "Predstojeće" (offered+accepted). Nema potrebe za novim backend parametrom za višestruke statuse na ovoj skali.

Flutter `Trip` model dobija `createdAt` (`DateTime`, iz `created_at` polja koje je backend već slao, samo se ranije nije parsiralo) — potrebno za prikaz datuma i sortiranje.

## Ulazne tačke

Nova stavka "Moje ture" u `RadialFabMenu`-u na oba vozačka početna ekrana:
- `VehicleListScreen` (samostalan vozač).
- `OfferedTripsScreen` (upravljani vozač) — ovde postoji delimično preklapanje sa ekranovim postojećim glavnim sadržajem (i sam prikazuje offered/accepted), ali "Moje ture" dodaje ono što tamo ne postoji: kompletnu istoriju završenih tura, i pregled aktivne ture na istom mestu.

## Namerni obim-cut

- "Predstojeće" tab je uvek prisutan i za samostalnog vozača, iako će uvek biti prazan — jednostavnije nego uslovno sakrivati tab po ulozi, i tačno odražava stvarnost (nema takvih tura, ne greška).
- Sortiranje je samo za Završene — Pokrenute (najviše jedna stavka) i Predstojeće nemaju očiglednu potrebu za tim.

## Verifikacija

`flutter analyze`/`flutter test` čisti. Backend nepromenjen (postojeći `GET /trips?status=` dovoljan). Stvarno ponašanje (tabovi, sortiranje, navigacija sa svakog taba) zahteva fizički uređaj — isto ograničenje kao za sav raniji UI rad u projektu.
