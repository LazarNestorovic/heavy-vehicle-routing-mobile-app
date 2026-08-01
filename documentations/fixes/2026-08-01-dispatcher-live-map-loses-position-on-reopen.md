# Dispečer gubi vozačevu poziciju kad izađe i vrati se na live mapu

**Datum:** 2026-08-01
**Fajlovi:** `backend/internal/ws/gateway.go`, `backend/internal/ws/gateway_test.go` (nov)

## Simptom

Dispečer je video vozačevu poziciju na "Vozila uživo" mapi, ali čim izađe sa tog ekrana i vrati se, marker nestane i ne vraća se - ponekad ni posle nekoliko minuta.

## Uzrok

`liveTrip` (in-memory objekat po turi u `ws.Gateway`) je bio čist fan-out bez pamćenja: `broadcast()` je samo prosleđivao update trenutno pretplaćenim kanalima i ništa nije čuvao. Kad dispečer zatvori ekran, `DispatcherLiveMapScreen.dispose()` zatvara WS konekciju (odjava iz `subscribers` mape); kad se vrati, otvara se POTPUNO NOVA WS konekcija koja se pretplaćuje "od nule" i ne dobija ništa dok ne stigne SLEDEĆI `ReportPosition` poziv sa vozačevog telefona.

Vozačev telefon pak ne šalje GPS non-stop - `LocationService.positionStream()` koristi `distanceFilter: 20` (fix samo na svakih ≥20m kretanja), pa ako vozač u tom trenutku stoji (semafor, pauza, parking), dispečer može ostati bez markera proizvoljno dugo, sve dok se vozilo ponovo ne pomeri.

## Popravka

`liveTrip` sad pamti poslednji poznati `positionUpdate` (`last *positionUpdate`, upisuje se u `broadcast()`), a `subscribe()` ga odmah (non-blocking, isti stil kao `broadcast()`-ova sigurnosna `select`/`default`) ubacuje u kanal novog pretplatnika ako postoji. Svaki novi WS gledalac (dispečerova mapa nakon ponovnog ulaska, ili bilo koja buduća druga upotreba istog gateway-a) odmah dobija poslednju poznatu poziciju, ne čeka sledeći GPS fix.

Nema izmene na Flutter strani - `TripSocket`/`DispatcherLiveMapScreen` već ispravno crtaju šta god prvo stigne sa WS-a.

## Verifikacija

`go build`/`vet`/`test` čisti. Novi `gateway_test.go` direktno testira `liveTrip` logiku (4 testa): replay poslednje pozicije novom pretplatniku, nema lažnog replay-a kad ništa još nije prijavljeno, broadcast posle pretplate i dalje stiže, i da je replay NEZAVISAN po pretplatniku (drugi dispečer ne "pojede" replay prvog). Backend je rebuildovan i pokrenut u Docker-u, logovi čisti bez grešaka. Stvarno ponašanje na dva telefona (vozač + dispečer) zahteva fizičke uređaje - isto ograničenje kao za sav raniji GPS/live rad u projektu.
