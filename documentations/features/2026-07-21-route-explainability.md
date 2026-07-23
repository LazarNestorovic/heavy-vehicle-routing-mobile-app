# Objašnjenje predložene rute (3.10)

**Datum:** 2026-07-21
**Fajlovi:** [`backend/internal/explain/`](../../backend/internal/explain/), [`backend/internal/valhalla/client.go`](../../backend/internal/valhalla/client.go)

## Šta je dodato

Implementacija `SPECIFIKACIJA.md` sekcije 3.10 — kad izabrana ruta odstupa od "idealne" (bez ograničenja vozila), `POST /api/v1/routes` i `POST /api/v1/trips` sada vraćaju `explanation` polje koje objašnjava **zašto**, ne samo **šta**.

```
backend/internal/explain/explain.go — Explainer.Explain(): binding-constraint detekcija
```

## Mehanizam

1. Zatraži se "referentna" ruta sa svim dimenzijama vozila oslobođenim na nerealno velike vrednosti (height=100m, weight=900t, itd.) — najmanje ograničena moguća putanja.
2. Ako je razlika u distanci između izabrane i referentne rute manja od 1km, nema smislenog zaobilaska → `explanation: null`.
3. Ako postoji razlika, redom se oslobađa **po jedna** dimenzija zahtevanog profila (height → weight → axle_load → width → hazmat) i ponovo traži ruta; prva dimenzija čije oslobađanje vrati distancu blisku referentnoj je vezujuće ograničenje.
4. Lokacija se aproksimira kao ime prve ulice na kojoj se manevri izabrane i referentne rute razilaze (`firstDivergentStreetName`).

## Bitan bug pronađen i ispravljen tokom testiranja

Prva verzija je referentnu rutu dobijala preko sirovog `RouteSingle` poziva (Valhalla-in "sirovi" najbolji predlog), dok je izabrana ruta prolazila kroz [naš risk-scoring](2026-07-21-risk-scoring-layer.md) (koji namerno bira drugačiju, ne nužno najkraću rutu). Rezultat: za **normalno, neograničeno vozilo** (height=4.0m, isti profil koji je testiran u risk-scoring-layer.md) sistem je javljao `"Ruta skreće kod M11 jer visina vozila (4.0m) ne zadovoljava ograničenje..."` — potpuno netačno objašnjenje, jer je stvarni uzrok razlike bila **naša sopstvena scoring formula** (preferira veći `highway_ratio`), ne ograničenje vozila.

**Ispravka:** i referentna i sve probne rute sada prolaze kroz identičan `RouteAlternates` + `scoring.Rank` pipeline kao produkciona ruta (`Explainer.rankedBest`), tako da se poredi jabuka sa jabukom — svaka razlika je sada zaista posledica ograničenja vozila, ne razlike u metodologiji izbora rute.

## Verifikacija (2026-07-21, posle ispravke)

- `height=4.0` (Beograd→Novi Sad): `explanation: null`, `distance_km: 92.364` — ista ruta kao referentna (identičan izbor scoring-a), nema pravog ograničenja.
- `height=4.7`: `explanation: "Ruta skreće kod Партизанске авијације jer visina vozila (4.7m) ne zadovoljava ograničenje na toj deonici."` — tačno identifikuje `height` kao vezujuće ograničenje.
- `weight=60000kg`: `explanation: null` — ova konkretna ruta nema aktivno ograničenje težine (A1 auto-put nosi i teža vozila); dokazuje da sistem ne izmišlja objašnjenje kad ga nema.
- Test suite: `TestFirstDivergentStreetName` (4 slučaja, čista logika, bez zavisnosti od žive Valhalla instance).

## Poznato ograničenje: lokacija može pokazivati na početak rute, ne na tačno mesto prepreke

`firstDivergentStreetName` pronalazi **prvi** manevar gde se imena ulica razlikuju — radi dobro kad izabrana i referentna ruta dele zajednički deo pa se lokalno razdvoje (npr. Banovci slučaj). Ali kad su rute **globalno različite od samog početka** (Valhalla bira potpuno drugu strategiju za jako ograničeno vs neograničeno vozilo), prijavljena lokacija može biti blizu polazišta, ne blizu stvarne prepreke. Primećeno u testu: `Партизанске авијације` je ulica blizu Beograda (početak rute), ne kod stvarnog nadvožnjaka. Poznato, dokumentovano ograničenje — future work bi bio pravi geometrijski presek putanja (uporediti dekodirane `shape` tačke, ne samo nizove imena ulica).

## Šta namerno NIJE urađeno

- Do 6 dodatnih Valhalla poziva po zahtevu kad postoji detour (1 referentna + do 5 probnih) — prihvatljivo za demo/tezu, ne za produkcioni saobraćaj velikog obima.
- Nema keširanja referentne rute između zahteva sa istim origin/destination.
- Lokacija je ime ulice, ne GPS koordinata niti tačan opis prepreke (npr. "nadvožnjak").
