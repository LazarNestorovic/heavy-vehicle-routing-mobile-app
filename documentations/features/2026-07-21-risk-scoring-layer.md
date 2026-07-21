# Risk-scoring sloj nad Valhalla alternativama

**Datum:** 2026-07-21
**Fajlovi:** [`backend/internal/scoring/scoring.go`](../../backend/internal/scoring/scoring.go), [`backend/internal/valhalla/client.go`](../../backend/internal/valhalla/client.go), [`backend/internal/httpapi/handlers.go`](../../backend/internal/httpapi/handlers.go)

## Šta je dodato

Prvi deo nedelje 2 iz `SPECIFIKACIJA.md` (sekcija 3.3.1). `POST /api/v1/routes` sada traži od Valhalla-e primarnu rutu + 2 alternative (`"alternates": 2`), rangira ih po heurističkom "risk score"-u, i vraća najbolju — uz transparentan prikaz svih razmatranih kandidata.

## Zašto baš ovi kriterijumi (i ne drugi)

Pre pisanja koda proverio sam šta Valhalla `/route` odgovor **stvarno** izlaže:
- Na nivou `summary`: `time`, `length`, `has_ferry`, `has_toll`, `has_highway`, `has_time_restrictions`.
- Na nivou `maneuvers`: `highway` (bool), `toll` (bool), `length`, ali **ne** `bridge`/`tunnel`/`surface`.

Podaci o mostovima/tunelima/podlozi po segmentu (koje smo filtrirali iz OSM-a — vidi [osmium filter fix](../fixes/2026-07-21-osmium-filter-node-tags.md)) nisu dostupni preko `/route`; zahtevali bi `/trace_attributes` ili čitanje OSM-a direktno (to je posao bounded A*/Dijkstra modula, sledeći korak, sekcija 3.3.2 specifikacije). Scoring formula ovde namerno koristi samo ono što je realno dostupno — nije glumljena preciznost.

## Formula (`scoring.go`)

```
risk_score = maneuver_count * 1.5
           + (1 - highway_ratio) * 100
           + (has_ferry ? 50 : 0)
           + (has_toll ? 5 : 0)
```

Niže je bolje. Težine su **prva heuristička procena, nisu kalibrisane** na stvarnim podacima o incidentima — vredi ih podesiti kad postoji evaluacioni dataset (pomenuto u `SPECIFIKACIJA.md` sekcija 8, "Testiranje i evaluacija").

## API promena

`POST /api/v1/routes` odgovor je proširen (ranije samo `distance_km`/`duration_min`/`shape`):

```json
{
  "distance_km": 92.364,
  "duration_min": 68.52,
  "shape": "...",
  "risk_score": 41.87,
  "candidates": [
    {"distance_km": 92.364, "duration_min": 68.52, "risk_score": 41.87, "maneuver_count": 15, "highway_ratio": 0.856, "has_ferry": false, "has_toll": true, "chosen": true},
    {"distance_km": 84.814, "duration_min": 68.37, "risk_score": 61.89, "maneuver_count": 10, "highway_ratio": 0.581, "has_ferry": false, "has_toll": true, "chosen": false},
    {"distance_km": 92.743, "duration_min": 75.50, "risk_score": 80.24, "maneuver_count": 20, "highway_ratio": 0.548, "has_ferry": false, "has_toll": true, "chosen": false}
  ]
}
```

`candidates` je uvek sortiran po `risk_score` rastuće; prvi element (`chosen: true`) je i dalje ujedno vrednosti u `distance_km`/`duration_min`/`shape`/`risk_score` na vrhu odgovora (zgodno za klijenta koji ne mari za alternative).

## Verifikacija (2026-07-21, Beograd → Novi Sad)

Scoring je stvarno promenio izbor u odnosu na Valhalla-in podrazumevani (najkraći/najbrži) predlog:
- Valhalla-in prvi predlog (bez scoring-a): 84.8 km, 58% na auto-putu.
- Naš izbor posle scoring-a: 92.4 km (~9% duže), ali 86% na auto-putu, uz **praktično isto trajanje** (68.5 vs 68.4 min).

Ovo je konkretan, merljiv primer "prilagođene cost funkcije" za poglavlje 5 rada (`SPECIFIKACIJA.md` sekcija 8) — bira se ruta koja favorizuje auto-put (manje raskrsnica/manevara za veliko vozilo) uz zanemarljivu cenu u vremenu.

Regresija proverena: 422 (van pokrivenosti) i 400 (axle_load > weight, [prethodni fix](../fixes/2026-07-21-osmium-filter-node-tags.md) — ovde referenca na validaciju, ne na taj fix) i dalje rade nepromenjeno.

## Šta namerno NIJE urađeno

- Nema kalibracije težina formule na stvarnim podacima — future work.
- Nema bridge/surface/hazmat-proximity signala (zahteva `/trace_attributes` ili bounded custom graf — sledeći korak plana).
- `numAlternates = 2` je hardkodovano (`handlers.go`), nije parametar API-ja — dovoljno za MVP, lako se otvori kao query param kasnije ako zatreba.
