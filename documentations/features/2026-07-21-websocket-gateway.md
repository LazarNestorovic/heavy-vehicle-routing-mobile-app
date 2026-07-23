# WebSocket gateway — simulacija pozicije uživo

**Datum:** 2026-07-21
**Fajlovi:** [`backend/internal/ws/gateway.go`](../../backend/internal/ws/gateway.go)

## Šta je dodato

Implementacija `SPECIFIKACIJA.md` sekcije 3.7. `GET /ws/trips/{id}` — WebSocket endpoint koji gura simuliranu GPS poziciju duž rute putovanja, plus rest-stop predlog čim ga [worker](2026-07-21-rest-stop-locations.md) izračuna.

## Mehanizam

- Nema pravog vozila/telefona u ovom projektu (rok od mesec dana, vidi `SPECIFIKACIJA.md` sekciju 1) — pozicija se simulira dekodiranjem `trip.Shape` (`valhalla.DecodePolyline6`) i "šetanjem" duž te putanje.
- Cela simulacija traje **60 sekundi wall-clock vremena bez obzira na stvarno trajanje rute** (`simDuration` konstanta) — ruta od 320 minuta vožnje (Subotica-Vranje) i ruta od 68 minuta (Beograd-Novi Sad) obe se odigraju za ~60 sekundi u demo-u. Poruka na svakih 500ms (`tickInterval`).
- Svaka poruka: `{lat, lon, progress_fraction, eta_min, status, rest_stop?}`. `rest_stop` se šalje **jednom**, čim worker upiše lokaciju u bazu (gateway proverava DB na svakom tick-u dok se ne pojavi, pa prestaje da proverava).
- `status` prelazi u `"arrived"` na poslednjoj poruci (`progress_fraction: 1`), posle čega se konekcija zatvara.

## Zavisnost

`github.com/gorilla/websocket` — jedina eksterna zavisnost dodata u ovom koraku (ostatak backend-a namerno koristi samo stdlib gde je moguće, ali WebSocket protokol ručno nije vredelo implementirati).

## Verifikacija (2026-07-21)

Testirano preko Node.js `ws` klijenta (nema `wscat` interaktivnog izlaza u ovom okruženju, direktan Node skript je pouzdaniji):
- Beograd→Novi Sad trip: konekcija, `OPEN`, pozicija kreće sa `progress_fraction: 0` (44.799996, 20.399933 ≈ polazište) i pravilno napreduje (`eta_min` opada sa 68.5 nadole).
- Subotica→Vranje trip (duga ruta, rest-stop predlog postoji): WebSocket poruka je ispravno uključila `rest_stop: {"lat":43.1495558,"lon":21.8892843,"name":"Pan Ledi","amenity":"fuel"}` čim je worker završio obradu.

## Šta namerno NIJE urađeno

- `CheckOrigin` je potpuno permisivan (`return true`) — prihvatljivo za demo sa jednim poverljivim klijentom (Flutter app), ne za javan multi-tenant servis.
- Nema autentifikacije/autorizacije na WS konekciji — bilo ko sa `trip_id`-jem se može povezati.
- Nema reconnect/resume logike na serverskoj strani (ako se klijent diskonektuje pa vrati, simulacija kreće ispočetka, ne nastavlja od tačke prekida).
- `simDuration`/`tickInterval` su hardkodovani (ne mogu se podesiti po zahtevu) — dovoljno za demo.
