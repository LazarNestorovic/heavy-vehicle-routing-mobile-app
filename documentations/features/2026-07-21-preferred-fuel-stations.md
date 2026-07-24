# Preferirane pumpe — brend i sačuvane lokacije (Faza 5)

**Datum:** 2026-07-21
**Plan:** `/home/lazar/.claude/plans/lovely-cooking-puppy.md` (Faza 5)
**Fajlovi:** `backend/internal/reststop/reststop.go`, `backend/internal/worker/trip_worker.go`, `backend/internal/scoring/scoring.go`, `backend/internal/httpapi/routes.go`

## Šta je dodato

Nastavak [driver preference sistema](2026-07-21-driver-preference-scoring.md). Vozač može da postavi `preferred_fuel_brand` (npr. "НИС Петрол") i sačuva konkretne omiljene lokacije (`driver_favorite_stops`, iz Faze 3). Ovo sada utiče na **oba** mesta koja je korisnik tražio u plan-mode razgovoru:

1. **Predlog pauze** (`trip.started` worker) — bira preferiranu stanicu umesto proizvoljno najbliže.
2. **Sama ruta** (scoring) — ruta koja prolazi blizu omiljene/brend stanice dobija bonus u `risk_score`.

## Mehanizam

- `reststop.Stop` dobio `Brand` polje (OSM `brand` tag, već postojao u podacima — potvrđeno ranije).
- `Finder.NearestPreferred(lat, lon, brand, favorites, maxRadius)` — redosled pokušaja: (1) najbliža sačuvana omiljena lokacija, (2) najbliža stanica traženog brenda, (3) fallback na običan `Nearest` (bilo koji brend/tip) — sve u okviru `DefaultPreferredRadiusM` (15km), da vozača ne šalje na veliki zaobilazak radi brenda.
- `Finder.ByBrand(brand)` — vraća SVE stanice tog brenda (za scoring, koji proverava blizinu rute svim kandidatima, ne samo jednoj tačci).
- `scoring.Rank`/`score` dobili novi parametar `preferredStops []valhalla.LatLon` — `preferredStopDiscount` dekodira `Shape` kandidata i, ako bilo koja preferirana tačka padne unutar 3km od bilo koje (svake 5-te, radi performansi) tačke rute, oduzima fiksnih 20 poena od `risk_score`. **Nije skalirano nijednim 1-5 prioritetom** (nije bilo posebne dimenzije za ovo u planu) — flat bonus.
- `httpapi.resolvePreferredStops(ctx, driverID)` — spaja sačuvane favorite + `ByBrand()` rezultate u listu koordinata, prosleđuje se i `bestRoute` i `Explain` (mora biti isti skup za oba, isti razlog kao [ranije](2026-07-21-route-explainability.md) — konzistentnost scoring pipeline-a).

## Verifikacija (2026-07-21, uživo)

- **Route scoring:** Beograd→Novi Sad, bez omiljene lokacije: `risk_score: 166.03`. Dodata omiljena lokacija tačno na početnu tačku rute (44.799996, 20.399933): `risk_score: 146.03` — tačno -20, kako formula predviđa.
- **Rest-stop worker:** postavljen `preferred_fuel_brand: "НИС Петрол"`, dugačak trip Subotica→Vranje → `rest_stop: {"name": "НИС Петрол", "amenity": "fuel", ...}` (ranije, bez preference, isti trip je predlagao "Pan Ledi" — drugi brend, prosto najbliži).
- `go test ./internal/reststop/... ./internal/scoring/...` — novi testovi (`TestFinder_NearestPreferred_*`, `TestRank_PreferredStopNearRoute_GetsDiscount`) uz sve postojeće.

## Zanimljiva greška uhvaćena usput

Prva verzija testa `TestRank_PreferredStopNearRoute_GetsDiscount` je koristila **ručno skraćen** polyline6 string (uzet prvih ~60 karaktera od pravog Valhalla shape-a) — to je preseklo enkodirani token nasred, i `DecodePolyline6` je pukao sa index-out-of-range panikom umesto greške. Ispravka: uzet je **pun, stvaran** shape direktno sa žive Valhalla instance (`curl` poziv), ne ručno seckan. Vredna lekcija: polyline6 nije bezbedno seckati na proizvoljnoj granici karaktera — svaki token mora biti kompletan.

## Šta namerno NIJE urađeno

- `preferredStopDiscount` uzorkuje samo svaku 5. tačku dekodirane rute (performanse) — teoretski može promašiti veoma kratak segment blizu preferirane stanice na dugoj ruti sa retkim tačkama; prihvatljivo za MVP.
- Nema UI indikacije *zašto* je ruta dobila bonus (npr. "ruta prolazi blizu vaše omiljene pumpe X") — samo utiče na broj, ne generiše poruku kao `explain` paket.
- Bonus (20 poena) i radijus (3km/15km) nisu kalibrisani — heuristika, isti princip kao ostatak formule.
