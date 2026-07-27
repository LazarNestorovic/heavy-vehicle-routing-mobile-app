# "Isto vozilo dva puta" u Mojim vozilima za upravljanog vozača + mogao je da menja tuđu flotu

**Datum:** 2026-07-26
**Fajlovi:** `backend/internal/httpapi/{roles,vehicles}.go`, `mobile/lib/screens/vehicle_list_screen.dart`

Korisnik je prijavio da se za vozača koji je u floti (upravljan od dispečera) u "Moja vozila" prikazuje isto vozilo dva puta.

## Uzrok — nije duplikat, nego dva vozila sa identičnim demo profilom

`GET /api/v1/vehicles` za upravljanog vozača namerno vraća UNIJU njegovih ličnih vozila i CELE dispečerove flote (Blok K). Provera uživo (curl) potvrdila je da backend vraća dva RAZLIČITA vozila (različit `id`), ne isti red dvaput — ali `VehicleProfileScreen`-ova forma je pre-popunjena istim "tipičnim EU šleperom" (4.0m/2.55m/16.5m/40000kg/11500kg) za SVAKO novo vozilo. Ako i vozač i dispečer sačuvaju formu bez izmene podrazumevanih vrednosti, dobijaju dva vozila sa identičnim prikazanim brojevima — lista ih nije nikako razlikovala, pa je izgledalo kao greška.

Dok sam ovo istraživao, primetio sam i pravu propust u ovlašćenjima: `handleUpdateVehicle`/`handleDeleteVehicle` (Blok M) su koristili isti `vehicleAccessible` kao pregled — što znači da je upravljani vozač preko iste liste mogao da **izmeni dimenzije ili obriše dispečerovo flotno vozilo**, ne samo svoje lično.

## Popravka

- **Vizuelno razdvajanje**: `VehicleListScreen` sad prikazuje drugačiju ikonicu (`Icons.warehouse_outlined` za flotu) i dodaje " · Flota" u podnaslov za flotna vozila (koristi već postojeće `is_fleet` polje iz Bloka K).
- **Meni za izmenu/brisanje sakriven za tuđu flotu**: prikazuje se samo ako je vozilo lično VLASNIŠTVO pozivaoca, ili ako je pozivalac dispečer nad SOPSTVENOM flotom.
- **Backend ojačan nezavisno od UI-ja**: nov `vehicleMutable()` (stroža verzija `vehicleAccessible()`) — dozvoljava izmenu/brisanje SAMO stvarnom vlasniku (lično vozilo → vozač; flotno vozilo → dispečer). Pregled (`GET`) i ažuriranje statusa goriva/servisa (`PATCH .../status`) ostaju na širem `vehicleAccessible` — vozač na putu i dalje sme da prijavi nivo goriva flotnog kamiona kojim vozi, samo ne sme da mu menja dimenzije ili ga obriše.

## Verifikacija

Uživo protiv Docker stack-a, upravljani vozač: `PUT`/`DELETE` na dispečerovom flotnom vozilu → `403` oba puta; `GET` na istom → i dalje `200` (pregled ostaje dozvoljen); `PUT` na sopstvenom ličnom vozilu → `200`. Dispečer i dalje slobodno menja sopstvenu flotu → `200`. `go build`/`vet`/`test` i `flutter analyze`/`test` čisti.
