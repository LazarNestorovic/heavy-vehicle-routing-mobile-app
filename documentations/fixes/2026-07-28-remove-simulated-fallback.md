# Simulacija kretanja potpuno uklonjena

**Datum:** 2026-07-28
**Fajlovi:** `backend/internal/ws/gateway.go`, `backend/main.go`, `backend/internal/httpapi/trips.go`, `backend/internal/store/trip.go`, `mobile/lib/screens/{active_trip,dispatcher_live_map}_screen.dart`, `mobile/lib/services/{location_service,api_client}.dart`

Korisnik je prijavio da posle pokretanja ture, prvi klik na mapu ili na radijalni meni u uglu pokreće vidljivo simulirano kretanje vozila.

## Uzrok

`HandleTripStream` je (od [prethodne popravke](2026-07-26-trip-start-position-jump.md)) čekao do 3 sekunde na prvi pravi GPS ping pre pada na simulirani hod duž rute. Taj "grace period" je suštinski trka protiv toga koliko brzo telefon isporuči prvi GPS fix — na nekim uređajima/uslovima to traje duže od 3 sekunde, pa bi simulacija već krenula pre nego što stigne prava pozicija. Korisnička interakcija (klik na mapu/meni) nije uzrok, samo trenutak kad je promena postala primetna.

Umesto da se ponovo tjuning-uje trajanje grace perioda (privremena zakrpa, ne rešenje), korisnik je ovoga puta eksplicitno tražio da se simulacija u potpunosti ukloni — razumna odluka i zato što `StartProximityStatus` (postojeća provera pre "Kreni"/"Kreni na put") već zahteva da vozač bude fizički na polaznoj tački sa važećom GPS pozicijom da bi uopšte mogao da pokrene turu. Simulacija je do sada bila i dalje potencijalno dostupna (fallback), iako je u praksi realan GPS već garantovan pre starta — simulacija je time bila i krhka (trka) i suvišna.

## Popravka

**Backend (`internal/ws/gateway.go`)** — potpuno prepisan:
- `simulate()`, `simDuration`, `tickInterval`, `liveGraceDuration` i sav grace-period/timeout kod obrisani.
- `HandleTripStream` sad samo: učitaj turu → nadogradi na WS → pretplati se na turin `liveTrip` kanal → prosleđuj šta god stigne, dok se ne javi `"arrived"` ili se konekcija ne prekine. Nema više nikakve grane koja bi "izmišljala" poziciju.
- `liveTrip.hasData`/`isLive()` uklonjeni (postojali su samo da razlikuju "pretplatnik čeka" od "stvarno stigao ping" — potrebno SAMO za grace-period/fallback logiku koja više ne postoji).
- Pošto se pretplata na kanal dešava ODMAH pri konekciji (pre bilo kakve grane), ranija trka oko izgubljenog prvog broadcast-a (vidi prethodnu popravku) ostaje zatvorena — sad je to jedina i najjednostavnija putanja kroz kod, ne poseban slučaj.
- `Gateway.TripEvents` polje uklonjeno (korišćeno je samo unutar `simulate()` da upiše "arrived" trip_event pri `progress_fraction=1`; `handleCompleteTrip` u `trips.go` već nezavisno upisuje isti event za pravi GPS tok). `ws.New()` potpis skraćen sa `(trips, tripEvents)` na `(trips)`.

**Flutter (`active_trip_screen.dart`)**: traka `_gpsStatusBanner()` više ne pominje "simulirano kretanje" — sad kaže da se vozačeva pozicija ne prikazuje (dok se GPS ne uključi/dozvoli), sa istim dugmetom za rešavanje kao pre.

**Komentari** (bez funkcionalne izmene) ažurirani na više mesta (`trips.go`, `store/trip.go`, `api_client.dart`, `location_service.dart`, `dispatcher_live_map_screen.dart`) da više ne pominju nepostojeću simulaciju kao poređenje/fallback.

## Namerni obim-cut

Ako vozaču GPS postane nedostupan USRED ture (dozvola povučena, isključena lokacija), mapa jednostavno ne prikazuje ništa novo dok se ne reši — nema više tihog niti vidljivog fallback-a. `_gpsStatusBanner` i dalje objašnjava tačno zašto i nudi rešenje jednim dodirom, što je dovoljno s obzirom da je ovaj slučaj redak (dozvola je već morala biti data da bi tura uopšte krenula).

## Verifikacija

`go build`/`vet`/`test` i `flutter analyze`/`flutter test` čisti. Uživo (Node WS klijent protiv Docker stack-a), dva scenarija:
- Bez ikakvog GPS ping-a: WS konekcija ne dobija nijednu poruku tokom celog 8-sekundnog testa (ranije bi simulacija krenula posle ~3s) — potvrđeno da fallback-a više nema.
- Sa pravim GPS ping-om (+1024ms): WS klijent prima relay za manje od 20ms — live tok radi nepromenjeno.
