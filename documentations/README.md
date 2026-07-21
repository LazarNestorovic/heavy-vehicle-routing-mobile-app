# Dokumentacija projekta

Ovaj folder prati **svaku promenu** napravljenu na projektu (popravke, arhitektonske odluke, operativna uputstva), odvojeno od `SPECIFIKACIJA.md` u root-u koji je "living" pregled arhitekture i plana rada. Dokumentacija je organizovana po tipu sadržaja, ne hronološki u jednom fajlu, da bi bila lakša za pretragu:

```
documentations/
├── CHANGELOG.md      — hronološki indeks svih zapisa (jedan red = jedna promena, sa linkom na detalje)
├── fixes/             — konkretne popravke/ispravke postojećeg koda, jedan fajl po popravci
├── features/          — nove funkcionalnosti/komponente dodate u sistem, jedan fajl po feature-u
├── decisions/         — arhitektonske/tehničke odluke i zašto su donete (ADR stil)
└── guides/            — operativna uputstva "kako uraditi X" (npr. kako rebuild-ovati graf)
```

## Kada šta pisati

| Situacija | Folder | Primer |
|---|---|---|
| Ispravljam bug ili grešku u postojećem kodu/konfiguraciji | `fixes/` | "osmium filter je odbacivao node tagove" |
| Dodajem novu komponentu/mogućnost koja ranije nije postojala | `features/` | "Go backend skelet sa POST /routes" |
| Biram između dve tehničke opcije i beležim zašto | `decisions/` | "Zašto Valhalla umesto GraphHopper-a" |
| Pišem uputstvo kako se nešto radi/pokreće | `guides/` | "Kako rebuild-ovati Valhalla graf" |

## Konvencija imenovanja

- `fixes/YYYY-MM-DD-kratak-opis.md`
- `features/YYYY-MM-DD-kratak-opis.md`
- `decisions/NNNN-kratak-opis.md` (rastući broj, npr. `0001-...`)
- `guides/kratak-opis.md` (bez datuma — guide se ažurira in-place kad zastari)

Svaki novi zapis se **mora** dodati i u [`CHANGELOG.md`](CHANGELOG.md) kao jedan red, najnoviji na vrhu.

## Template za `fixes/`

```markdown
# Naslov popravke

**Datum:** YYYY-MM-DD
**Fajlovi:** putanja/do/fajla.ext:linija

## Problem
...

## Popravka
...(referenca na tačan kod, pre/posle)

## Verifikacija
...(kako je potvrđeno da radi)
```
