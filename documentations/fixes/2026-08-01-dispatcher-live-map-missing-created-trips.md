# Dispečerova live mapa nije prikazivala vozača koji je upravo krenuo

**Datum:** 2026-08-01
**Fajlovi:** `mobile/lib/screens/dispatcher_live_map_screen.dart`

## Simptom

Vozač je pokrenuo turu (i video svoju poziciju na sopstvenom `ActiveTripScreen`-u), ali se ta ista tura uopšte nije pojavljivala na `DispatcherLiveMapScreen`-u ("Vozila uživo") kod dispečera.

## Uzrok

Tura ostaje u statusu `"created"` sve dok asinhroni `trip.started` worker ne izračuna preporuku za pauzu i prebaci je u `"in_progress"` (`TripStore.UpdateAfterProcessing`) — to kašnjenje zavisi od RabbitMQ konzumera i može trajati primetno posle stvarnog polaska. `DispatcherLiveMapScreen._load()` je dohvatao SAMO `GET /trips?status=in_progress`, pa je tura u međuvremenu (ili ako worker kasni/ne stigne da obradi) bila potpuno nevidljiva na mapi — nije ni otvarao WS konekciju za nju.

Ovo je ista razlika koju su druge tačke u aplikaciji već ispravno tretirale (`TripStore.HasActiveTrip`, vozačev `TripListScreen`-ov "Pokrenute" tab) — obe uzimaju `created` I `in_progress` kao "aktivno".

## Popravka

`_load()` sad dohvata oba statusa paralelno (`Future.wait`, isti obrazac kao svuda drugde) i spaja rezultate pre nego što otvori `TripSocket` za svaku turu. WS relej pozicije ionako ne zavisi od DB statusa ture (`Gateway.ReportPosition` čuva poziciju u memoriji po `tripID`, bez obzira na `trips.status`), tako da nije trebalo ništa menjati na backend-u.

## Verifikacija

`flutter analyze`/`flutter test` čisti. Stvarno ponašanje (dispečer vidi vozača odmah po pritisku "Kreni") zahteva fizički uređaj — isto ograničenje kao za sav raniji UI/GPS rad u projektu.
