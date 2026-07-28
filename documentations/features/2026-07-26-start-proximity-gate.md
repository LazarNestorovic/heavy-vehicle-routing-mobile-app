# Pokretanje ture zahteva da vozač bude na polaznoj tački (kao Google Maps)

**Datum:** 2026-07-26
**Fajlovi:** `mobile/lib/services/location_service.dart`, `mobile/lib/widgets/start_proximity_status.dart`, `mobile/lib/screens/{route_request,trip_detail}_screen.dart`

## Zašto

Nakon Bloka D (`2026-07-26-live-gps-tracking.md`), vozačeva mapa prati STVARNU GPS poziciju čim stigne prvi pravi ping. Ako se ta pozicija ne poklapa sa polaznom tačkom koju je vozač (ili dispečer, za dodeljenu turu) izabrao pri planiranju rute, marker na mapi "skače" daleko od nacrtane linije — zbunjujuće, i nema smisla stvarno "voziti" turu koja kreće odnekud drugde.

Korisnik je eksplicitno tražio pristup po uzoru na Google Maps: **pregled rute radi uvek**, bez obzira gde se vozač nalazi; **stvarno pokretanje** ("Kreni na put" / "Kreni") je dozvoljeno SAMO kad se trenutna GPS pozicija poklapa sa polaznom tačkom.

## Mehanizam

Nov `LocationService.distanceToCurrentPosition(lat, lon)` — jednokratno (`Geolocator.getCurrentPosition`, ne stream) izračunava udaljenost od zadate tačke preko `Geolocator.distanceBetween`. Vraća `null` ako dozvola nije data ili pozicija nije dostupna (poziva se preko postojećeg `ensurePermission()` iz `2026-07-26-silent-gps-fallback.md` fix-a, pa se isti `GpsStatus` tok dozvola ponovo koristi).

Nov deljeni widget `StartProximityStatus` (`widgets/start_proximity_status.dart`) — prima ciljnu tačku (polazište), prikazuje status traku ("Proveravam...", "Na polaznoj tački.", "Udaljeni ste X km od polazišta...", ili "Nije moguće proveriti lokaciju" sa dugmetom za ponovni pokušaj) i javlja roditeljskom ekranu preko `onCanStartChanged(bool)` da li je vozač unutar praga od **500m** (`StartProximityStatus.thresholdMeters`).

Ugrađen na dva mesta:
- **`RouteRequestScreen`** (samostalan vozač) — traka se pojavljuje čim je polazna tačka postavljena (tap na mapu ili pretraga adrese), koristi TU tačku kao referentnu. "Pregled rute" ostaje uvek dostupan; "Kreni na put" je onemogućeno dok `_canStart` nije `true`.
- **`TripDetailScreen`** (upravljani vozač, dodeljena tura) — traka se pojavljuje samo u "accepted" stanju (pre dugmeta "Kreni"), koristi PRVU tačku dekodirane rute (`_routePoints.first`) kao polazište — nema potrebe za novim backend poljem, isti pristup kao postojeći marker za polazak na toj mapi.

`DispatcherCreateTripScreen` namerno nije dirana — dispečer ne vozi turu koju kreira, samo je nudi ("Ponudi turu"), pa provera blizine nema smisla na tom ekranu.

## Namerni obim-cut-ovi

- Prag od 500m nije kalibrisan — heuristika, isti stil kao ostale konstante u projektu (npr. `preferredStopRadiusM`).
- Provera je JEDNOKRATNA pri prikazu trake (plus ručno "Osveži" dugme) — ne prati kontinuirano da li se vozač približava dok gleda ekran; prihvatljivo, pošto je svrha "da li si SADA na polazištu", ne live navigacija do njega.
- Ako GPS dozvola nije data, traka prikazuje "Nije moguće proveriti lokaciju" i "Kreni" ostaje trajno onemogućeno (nema fallback-a na "dozvoli svakako") — dosledno odluci da provera stvarno mora da postoji da bi se pokrenulo, ne samo da postoji kad je dostupna.

## Verifikacija

`flutter analyze`/`flutter test` čisti. Backend nepromenjen (cela provera je klijentska, koristi već postojeći `trip.shape`/origin koordinate). Stvarno ponašanje (traka, prag od 500m, dugme "Osveži") zahteva fizički uređaj i stvarno kretanje - nije proverljivo bez toga.
