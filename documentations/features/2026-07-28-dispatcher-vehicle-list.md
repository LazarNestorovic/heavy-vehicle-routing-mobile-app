# Dispečer dobija listu vozila flote (ne samo "dodaj")

**Datum:** 2026-07-28
**Fajlovi:** `mobile/lib/screens/{vehicle_list,dispatcher_home}_screen.dart`

Dispečer je do sada iz svog menija mogao samo da DODA novo vozilo u flotu ("Dodaj vozilo u flotu" → pravo na formu). Nije mogao da vidi listu postojećih flotnih vozila niti da ih izmeni/obriše bez zaobilaznog puta.

## Mehanizam

Backend je već potpuno podržavao dispečera: `GET /api/v1/vehicles` vraća `Vehicles.ListFleet(dispatcherID)` za nalog sa `role=dispatcher` (`handleListVehicles`), a `PUT`/`DELETE /vehicles/{id}` već rade preko `vehicleMutable` provere koja dispečeru dozvoljava upravljanje sopstvenom flotom (`handleUpdateVehicle`/`handleDeleteVehicle`, uvedeno u ranijoj popravci). Nedostajao je samo Flutter ekran koji to koristi za dispečera — nije trebalo ništa dirati na backend-u.

`VehicleListScreen` (već postojeći ekran za vozača) je proširen da ispravno radi i za dispečera, pošto je najveći deo logike (razlikovanje sopstvenog/flotnog vozila preko `is_fleet`, `canManage` provera) već bio tu:
- Naslov: "Vozila u floti" umesto "Moja vozila" kad je `role == 'dispatcher'`.
- Prazna lista: poruka prilagođena ("Nemate još nijedno vozilo u floti.").
- Dodir na vozilo: za dispečera sad otvara izmenu (isto što i "Izmeni" iz popup menija) umesto pokušaja da otvori `RouteRequestScreen` (koji ima smisla samo za vozača koji sam vozi).
- `RadialFabMenu` na ovom ekranu se sad krije i za dispečera, ne samo za upravljanog vozača — dispečerov home ekran već ima identičan meni, drugi bi bio zbunjujući duplikat.
- "Dodaj vozilo" ostaje fiksni FAB, isti za sve slučajeve.

`DispatcherHomeScreen`-ova stavka u meniju "Dodaj vozilo u flotu" preimenovana u "Vozila" i sad vodi na `VehicleListScreen` umesto pravo na formu za dodavanje — dodavanje se sada radi IZ tog ekrana (FAB), kako je i traženo.

## Verifikacija

`flutter analyze`/`flutter test` čisti. Uživo (curl, dispečerski nalog): `GET /vehicles` prazno pre dodavanja, `POST` kreira flotno vozilo (`is_fleet:true`), `GET` ga vraća, `PUT` menja dimenzije, `DELETE` vraća `204` — pun CRUD tok koji Flutter ekran sada izlaže potvrđen end-to-end.
