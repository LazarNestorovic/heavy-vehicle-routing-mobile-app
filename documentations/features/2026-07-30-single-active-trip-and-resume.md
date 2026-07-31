# Jedna aktivna tura odjednom + povratak na nju sa početnog ekrana

**Datum:** 2026-07-30
**Fajlovi:** `backend/internal/store/trip.go`, `backend/internal/httpapi/trips.go`, `mobile/lib/services/api_client.dart`, `mobile/lib/widgets/active_trip_banner.dart`, `mobile/lib/screens/{vehicle_list,offered_trips,route_request,trip_detail}_screen.dart`

Dva povezana zahteva:
- Vozač treba da može da izađe sa ekrana aktivne ture (nazad dugme) i kasnije se vrati na MAPU SA ISTOM RUTOM.
- Vozač ne sme da može da pokrene novu turu dok je trenutna u toku.

## Izlazak i povratak — mehanički već radilo

`ActiveTripScreen` nema `PopScope`/`WillPopScope` niti bilo šta što bi blokiralo nazad dugme, i sav njegov state se gradi iznova iz `widget.trip`/`widget.vehicleId` pri svakom `initState`-u (backend je izvor istine za status/poziciju). Izlazak dakle već radi. Ono što je nedostajalo je **način da se vozač vrati** na aktivnu turu sa početnog ekrana pošto izađe — to je stvarni posao ovog fičera.

## Mehanizam

**Backend — `TripStore.HasActiveTrip(driverID)`** — vraća vozačevu turu sa statusom `created` ili `in_progress`, ili `nil`. Korišćeno na dva mesta da blokira POKRETANJE druge ture:
- `handleCreateTrip` (samostalan vozač — tura se kreira I odmah pokreće u jednom koraku) — `409` ako već ima aktivnu.
- `handleStartTrip` (upravljani vozač — `accepted` → pokrenuta) — `409` ako već ima aktivnu.

Namerno NIJE blokirano: dispečerovo NUĐENJE ture (`status=offered`) niti vozačevo PRIHVATANJE ponude — vozač sme da prihvati ponudu unapred, samo ne sme da je stvarno pokrene dok je zauzet.

**Flutter — `ApiClient.findActiveTrip()`** — poziva postojeći `GET /trips?status=created` pa `?status=in_progress` (trip brzo prelazi iz `created` u `in_progress` čim `trip.started` worker odradi svoje, pa se moraju proveriti oba statusa).

**Nov `ActiveTripBanner` widget** — samostalan (sam učitava svoje stanje), prikazan na oba vozačka početna ekrana (`VehicleListScreen` za samostalnog vozača, `OfferedTripsScreen` za upravljanog) — ako postoji aktivna tura, prikazuje traku "Imate aktivnu turu u toku" sa dugmetom "Nastavi" koje otvara `ActiveTripScreen` za TU turu. Eksplicitno preskače proveru za `role == 'dispatcher'` — dispečerov `GET /trips` vraća ture koje je ON dodelio (po `assigned_by_id`), ne ture koje ON vozi, pa bi inače pogrešno prikazivao nečiju tuđu aktivnu turu kao "svoju".

**Proaktivna blokada pre pokretanja** — `RouteRequestScreen` i `TripDetailScreen` takođe pozivaju `findActiveTrip()` pri otvaranju; ako postoji aktivna tura, prikazuje se traka/kartica sa objašnjenjem i dugmetom "Idi na nju", a dugme za pokretanje ("Kreni na put"/"Kreni") ostaje onemogućeno. Pregled rute ostaje dostupan (bezopasan, ne pokreće ništa) — blokiran je samo stvarni start. Ovo je čisto UX poboljšanje; backend-ova provera je i dalje jedina STVARNA garancija (proaktivna provera je best-effort — ako ona sama ne uspe, korisnik samo neće videti upozorenje unapred, ali backend će svejedno odbiti pokušaj).

## Verifikacija

`go build`/`vet`/`test` i `flutter analyze`/`flutter test` čisti. Uživo (curl, oba toka):
- Samostalan vozač: druga `POST /trips` dok je prva aktivna → `409 "you already have an active trip in progress"`; posle `POST /trips/{id}/complete` treća uspeva → `201`.
- Upravljani vozač: dispečer ponudi dve ture istom vozaču, vozač prihvati obe, pokrene prvu (`200`), pokušaj pokretanja druge dok je prva aktivna → `409`.

## Dopuna 2026-07-30 — dodir na VOZILO koje je na aktivnoj turi

Na `VehicleListScreen` (self-servisni vozač), dodir na vozilo je i dalje vodio na `RouteRequestScreen` čak i kad je TO ISTO vozilo trenutno na aktivnoj turi — vozač bi samo dobio blokadu opisanu iznad, umesto da ga aplikacija odmah prebaci na `ActiveTripScreen`.

`_VehicleListScreenState` sad drži sopstveni `_activeTrip` (zaseban poziv `findActiveTrip()`, namerno odvojen od `ActiveTripBanner`-a koji ostaje samostalan radi jednostavnosti — mala cena dodatnog GET poziva prihvatljiva je za nezavisnost oba dela). Dodir na vozilo prvo proverava `_activeTrip?.vehicleId == v.id`: ako se poklapa, ide direktno na `ActiveTripScreen` za tu turu (i osvežava `_activeTrip` posle povratka, u slučaju da je tura u međuvremenu završena); inače se ponaša kao pre (planiranje rute / poruka za upravljanog vozača / izmena za dispečera).

`flutter analyze`/`flutter test` čisti.
