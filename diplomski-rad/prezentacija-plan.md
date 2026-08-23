# Plan prezentacije za odbranu diplomskog rada

**Tema:** Razvoj aplikacije za rutiranje teretnih vozila prilagođene fizičkim ograničenjima vozila i terena
**Kandidat:** Lazar Nestorović · **Mentor:** Prof. dr Dušan Gajić
**Trajanje izlaganja:** 13:30 (rezerva do 15) · **Broj slajdova:** 17 + 4 backup
**Gotova prezentacija:** [prezentacija/odbrana.html](prezentacija/odbrana.html) — otvoriti u pregledaču

## Raspodela vremena (nosivi deo je algoritamski — poglavlja 6 i 9)

| Blok | Slajdovi | Vreme |
|---|---|---|
| Uvod i problem | 1–3 | ~2:00 |
| Arhitektura i podaci | 4–6 | ~2:30 |
| **Algoritamski doprinos** | 7–10 | **~4:30** |
| Sistem (backend, mobilna) | 11–13 | ~2:00 |
| Evaluacija i zaključak | 14–17 | ~2:30 |

Zlatno pravilo: slajdovi 8, 9 i 14 su srce odbrane — na njima se ne žuri. Slajdovi 5, 12, 13 se mogu skratiti ako vreme izmiče.

---

## Slajd 1 — Naslovna (0:15)

- Naslov rada, ime kandidata i broj indeksa, mentor, fakultet, datum.
- Pozadina: jedan screenshot mape sa rutom, zatamnjen — vizuelno postavlja temu pre nego što progovoriš.

**Reći:** samo predstavljanje i naslov teme. Ne čitati naslov doslovno.

---

## Slajd 2 — Problem (1:00)

- Slika/ilustracija: kamion ispred nadvožnjaka sa oznakom visine.
- Tri stavke, bez rečenica:
  - Teretno vozilo ≠ putnički automobil: visina, masa, osovinsko opterećenje, širina, dužina, opasan teret.
  - Google Maps / Waze te podatke ne uzimaju u obzir → ruta može biti fizički neprohodna ili nedozvoljena.
  - Komercijalna rešenja (PTV, TomTom Truck) su zatvorena i plaćena — vozač ne vidi **zašto** je ruta izabrana.

**Reći:** konkretan primer — vozilo visine 4,7 m i nadvožnjak od 4,5 m; standardna navigacija ga vodi pravo u prepreku.

---

## Slajd 3 — Cilj rada i doprinosi (0:45)

- Cilj u jednoj rečenici: *objašnjiv* sistem rutiranja teretnih vozila nad potpuno otvorenim podacima (OSM) i otvorenim engine-om (Valhalla).
- Četiri doprinosa (iz poglavlja 10.1), po jedna linija:
  1. Sloj rangiranja alternativa po riziku specifičnom za teretni saobraćaj.
  2. Automatsko objašnjenje zašto ruta odstupa od najkraćeg puta.
  3. Sopstvena Dijkstra/A* implementacija nad realnim OSM podgrafom.
  4. Kompletan distribuiran sistem (REST + RabbitMQ + WebSocket) sa ulogama vozač/dispečer.

**Reći:** ovo je „mapa puta" cele prezentacije — najaviti da će naglasak biti na 1–3.

---

## Slajd 4 — Arhitektura sistema (1:00)

- Dijagram `diagrams/architecture.pdf` iz rada, preko cele širine slajda.
- Pet procesa: Flutter app · Go backend · Valhalla · PostgreSQL/PostGIS · RabbitMQ. Sve kontejnerizovano (`docker compose up`).

**Reći:** pratiti jedan tok prstom po dijagramu — `POST /routes` → Valhalla (3 alternative) → scoring → explain → odgovor klijentu. Naglasiti da backend za svaki zahtev sam postaje klijent eksternog servisa; to je razlika u odnosu na klasičnu troslojnu veb aplikaciju.

---

## Slajd 5 — Priprema podataka (0:45)

- Pipeline u 4 koraka kao strelice: Geofabrik ekstrakt Srbije → `osmium tags-filter` → Valhalla graph build → servis.
- Ključni brojevi: nacionalni graf **898 337 čvorova**.
- Napomena o tagovima: ograničenja su u OSM-u na **dva nivoa** — na way-u (`maxheight` na deonici) i na node-u (`barrier` sa `maxheight`), pa filter mora hvatati oba.

**Reći:** kratko. Poenta je da su ograničenja u OSM-u nekonzistentno zavedena i da je to uticalo na dizajn oba sloja rutiranja.

---

## Slajd 6 — Dvoslojni pristup rutiranju (0:45)

- Centralni slajd-koncept, dve kolone:
  - **Produkcioni sloj:** Valhalla `truck` costing → alternative → sopstveni `scoring` → sopstveni `explain`.
  - **Algoritamski sloj:** sopstveni Dijkstra/A* nad ograničenim podgrafom, bez Valhalle, samo kroz testove.
- Ispod: *„Reimplementacija nacionalnog routing engine-a nije realan cilj diplomskog rada — demonstracija principa nad manjim, ali stvarnim grafom jeste."*

**Reći:** ovo je odbrana metodološke odluke i najverovatnije pitanje komisije — reći to sam, pre nego što te pitaju.

---

## Slajd 7 — Valhalla truck costing (0:45)

- Skraćen Listing 6.1: JSON zahtev sa `costing: "truck"` i `costing_options` (height 4.0, width 2.55, weight 40, axle_load 11.5, hazmat).
- Ključno: profil vozila ide **po zahtevu**, graf se ne gradi iznova za svako vozilo.
- Ograničenje: `/route` odgovor ne kaže **koja** ivica je isključena ni njenu tačnu vrednost → motivacija za slajdove 8 i 10.

---

## Slajd 8 — Sopstvena funkcija rizika (1:15) ★

- Formula u čitljivom obliku (ne ceo Go kod — samo članovi):
  `score = w_t·timeTerm + w_h·highwayTerm + w_f·fuelTerm + w_c·cargoTerm + manevri + ferry/toll penali − popust za omiljenu stanicu`
- Preference vozača kao skalar `priority/3` (1–5, 3 = neutralno).
- Naglasiti: Valhalla vraća do 3 alternative ali ih **ne rangira** — ovaj sloj je ono što nedostaje.
- Honest disclosure: koeficijenti su nekalibrisana heuristička procena (i tako je označeno u kodu i u radu).

**Reći:** povezati sa slajdom 14 (slučaj Radalj) — „vremenski član nije dodat unapred, dodat je zato što je konkretan test pokazao pogrešan ishod."

---

## Slajd 9 — Objašnjenje rute: slučaj Novi Banovci (1:30) ★★

Najjači slajd u prezentaciji — konkretan, stvaran, vizuelan.

- Mapa sa dve rute: referentna (opušteni limiti) i stvarna (vozilo 4,7 m) sa detourom kod Novih Banovaca.
- Algoritam u 4 koraka:
  1. Traži se referentna ruta sa nerealno opuštenim dimenzijama (visina 100 m, masa 900 t).
  2. Ako je razlika < 1 km — nema šta da se objasni.
  3. Dimenzije se oslobađaju **jedna po jedna**; ona čije oslobađanje vrati direktnu rutu je vezujuće ograničenje.
  4. Tačka odstupanja se nalazi **geometrijski** — prva tačka udaljena > 200 m od svake tačke referentne rute.
- Izlaz vozaču: *„Ruta skreće kod [ulica] jer visina vozila (4,7 m) ne zadovoljava ograničenje na toj deonici."*

**Reći:** dve stvari koje treba izgovoriti eksplicitno:
- Ograničenje kod Novih Banovaca (A1, izlaz 21/22, između 4,5 i 4,6 m) je **prvo ručno potvrđeno** binarnom pretragom, pa tek onda automatizovano.
- Poređenje je geometrijsko, **ne** po nazivima ulica/manevrima — jer dve rute do istog odredišta mogu imati potpuno različitu strukturu manevara, pa poređenje po poziciji u nizu nije pouzdano.

---

## Slajd 10 — Sopstveni Dijkstra/A* (1:00)

- Skraćen kod: funkcija `allowed()` (tvrdo isključivanje po maxheight/maxweight/hazmat) + `cost()` (dužina × podloga × klasa puta).
- Dijkstra i A* dele **istu** funkciju `search`; razlika je samo u heuristici (nula vs. haversine do cilja).
- Dodatno: penal za skretanje, ograničenja i na ivici i na čvoru (dva nivoa iz slajda 5).
- Pošteno navesti pojednostavljenje: penal za skretanje se računa nad jednim fiksiranim prethodnikom, ne nad parom (čvor, dolazna ivica).

**Reći:** naglasiti da ovaj modul nije uvezan u `main.go` — čista biblioteka koja postoji radi demonstracije i merenja.

---

## Slajd 11 — Backend i model podataka (0:45)

- REST API grupisan po celinama: `/auth`, `/vehicles`, `/routes`, `/trips`, `/dispatcher`, `/chats`, `/ws`.
- Model podataka: Driver (role: driver/dispatcher) · Vehicle (pripada **ili** vozaču **ili** dispečeru, nikad oboma) · Trip · TripEvent · ChatMessage.
- Autentifikacija: JWT (HS256) + `token_version` — odjava sa svih uređaja inkrementiranjem broja, bez blocklist tabele.

---

## Slajd 12 — Asinhrona obrada i real-time (0:45)

- RabbitMQ tok: `trip.started` → worker → predlog pauze → upis u `trips` + `trip_events`.
- Modul pauze: prag 270 min (4,5 h) → tačka na ruti u tom trenutku → najbliže odmaralište u koridoru trase (hazmat preferira pumpe, favorizuje omiljenu stanicu).
- WebSocket gateway: relej pozicije svim pretplatnicima ture; novi pretplatnik odmah dobija poslednju poznatu poziciju.
- Napomena: 270 min je svesno pojednostavljenje AETR regulative — cilj je mehanizam, ne zakonska logika.

---

## Slajd 13 — Mobilna aplikacija (0:30)

- 3–4 screenshot-a u nizu: planiranje rute sa alternativama · aktivna tura sa ETA i alertom za pauzu · dispečerska live mapa · dnevnik ture.
- Jedna Flutter kodna baza, grananje po ulozi odmah nakon prijave (`entry_router`).
- Dva detalja koja vredi pomenuti (pokazuju promišljenost, ne samo obim):
  - `start_proximity_status`: tura se ne može pokrenuti ako vozač nije unutar 500 m od polazišta — garantuje da je prva pozicija stvaran GPS fix.
  - Automatski reroute ako vozač odstupi od rute za više od 300 m.
  - Dispečerska live mapa drži **N paralelnih WebSocket konekcija**, po jednu za svaku aktivnu turu.

---

## Slajd 14 — Evaluacija (1:15) ★

Tabela sa stvarnim brojevima iz poglavlja 9 — komisija ceni merene rezultate:

| Šta | Rezultat |
|---|---|
| Automatizovani testovi | **57** test funkcija u 8 paketa, svi prolaze |
| Podgraf Beograd–Novi Sad | 38 265 čvorova, 50 705 ivica |
| Dijkstra ruta BG→NS | 81,0 km (Valhalla: 84,8–92,7 km) → plauzibilno |
| Dijkstra / A* | identična cena; A* ~17,1 ms vs. 19,3 ms |
| Učitavanje OSM XML-a | ~405 ms — dominira nad samom pretragom |
| Isključivanje po stvarnom tagu | `maxheight=4.3` u Beogradu: vozilo 4,0 m prolazi, 4,5 m nema put |

**Reći:** dva zaključka — (1) A* daje isti put uz manje pregledanih čvorova, kako teorija i predviđa za dopustivu heuristiku; (2) pretraga nije usko grlo, izgradnja grafa jeste — što je upravo razlog zašto je u produkciji graf prepušten Valhalli.

---

## Slajd 15 — Studija slučaja: Radalj (0:30)

- Rana verzija formule (bez `timeTerm`) birala rutu **47 % dužu i 30 % sporiju** samo zbog povoljnijeg odnosa magistrale i lokalnog puta.
- Dodavanjem vremenskog člana formula je ispravno počela da bira bržu rutu, bez uklanjanja ostalih članova.
- Lekcija: novi član u heurističkoj funkciji cene treba motivisati **uočenim lošim ishodom**, a ne pretpostavljenom kompletnošću.

**Reći:** ovo je konkretan dokaz da sopstveni sloj rangiranja stvarno menja ishod — bez njega bi „sirov" izbor prve Valhalla alternative bio pogrešan.

---

## Slajd 16 — Ograničenja i budući rad (0:30)

Dve kolone — ograničenja levo, pravci desno:

| Svesno izostavljeno | Budući rad |
|---|---|
| Puna AETR regulativa | Kompletna AETR logika |
| Elevacioni (nagibni) rizik | SRTM elevacioni podaci u funkciji rizika |
| Offline režim aplikacije | Lokalni keš + odloženo slanje pozicija |
| Samo Srbija | Proširenje na region/Evropu (+ PostGIS upiti) |
| Nekalibrisani koeficijenti | Kalibracija regresijom nad stvarnim podacima |
| Propustljiv WS `CheckOrigin`, minimalna MQ topologija | Bezbednosni hardening, retry/DLQ |

**Reći:** pomenuti i pošteno prijavljen negativan nalaz — sopstveni modul **ne reprodukuje** slučaj Novi Banovci, jer u tom ograničenom ekstraktu ograničenje nije zavedeno kao way tag. To je u radu navedeno, ne prikriveno.

---

## Slajd 17 — Zaključak (0:15)

- Tri rečenice, velikim fontom:
  1. Objašnjiv sistem rutiranja teretnih vozila je izvodljiv nad potpuno otvorenim podacima.
  2. Sopstveni doprinos je u dva sloja koja Valhalla nema — rangiranje po riziku i objašnjenje odstupanja rute.
  3. Oba su potvrđena na stvarnim, ne hipotetičkim slučajevima (Novi Banovci, Radalj).
- Zahvalnica mentoru i komisiji. „Hvala na pažnji."

---

## Fotografije u prezentaciji

Sve slike su **ugrađene u HTML** kao base64 — prezentacija je i dalje jedan samostalan fajl koji radi bez interneta. Originali stoje u [prezentacija/slike/](prezentacija/slike/).

| Slajd | Fajl | Autor i izvor | Uloga |
|---|---|---|---|
| 1 | `avi-richards-…jpg` | Avi Richards / Unsplash | Vertikalna traka desno — kamion na vijaduktu u sumrak |
| 2 | `pexels-cihat-dede-…jpg` | Cihat Dede / Pexels | Glavna slika problema — nadvožnjak sa znakom 3,30 m |
| 2 | `ragnar-rebase-…jpg` | Ragnar Rebase / Unsplash | „Danger low bridge" — prepreka je fizička |
| 17 | `jametlene-reskp-…jpg` | Jametlene Reskp / Unsplash | Zatamnjena pozadina zaključka — kamioni na koridoru |

Krediti su odštampani sitno u uglu svake fotografije. Unsplash i Pexels ne zahtevaju atribuciju, ali je na odbrani bolje da stoji.

**Neiskorišćene rezerve:** `pexels-jakubzerdzicki-…jpg` (znak 2 m) i `pexels-jan-van-der-wolf-…jpg` (stub sa 2,7 m i 30). Obe su dobre, ali su treća i četvrta varijacija istog motiva — znaka ograničenja visine. Stoje kao zamena ako neka od korišćenih ne odgovara.

**Još nedostaje:** screenshot-ovi aplikacije za slajd 13 (četiri okvira su i dalje placeholderi) i, opciono, stvarna mapa sa dve rute za slajd 9 umesto shematskog crteža.

## Backup slajdovi (ne prikazuju se, drže se za pitanja)

- **B1** — Cela tabela testova po paketima (Tabela 9.1).
- **B2** — Tabela faktora klase puta `roadClassMultiplier` (Tabela 6.1).
- **B3** — Dijagram toka `explain.Explain` (`diagrams/explain.pdf`).
- **B4** — Puna ER šema baze / migracije.
- **B5** — Kompletan `docker-compose.yml` sa tabelom servisa i portova.
- **B6** — Detalji autentifikacije: bcrypt, Google JWKS bez Firebase SDK-a, jednokratni reset tokeni.

## Očekivana pitanja komisije i pripremljeni odgovori

1. **„Zašto niste sami napisali routing engine?"** → Slajd 6: graf od 898 337 čvorova; sopstvena implementacija postoji i evaluirana je nad 38 265 čvorova, ali kao demonstracija principa, ne kao zamena.
2. **„Kako znate da su vam koeficijenti dobri?"** → Ne tvrdim da jesu; eksplicitno su označeni kao nekalibrisani. Slučaj Radalj pokazuje kako je jedan član dodat na osnovu uočene greške. Kalibracija regresijom je naveden pravac budućeg rada.
3. **„Šta ako OSM podaci nisu tačni?"** → Sistem je onoliko tačan koliko i podaci; zato mehanizam objašnjenja prikazuje vozaču razlog, da može sam da proceni razumnost predloga — to je prednost u odnosu na zatvorena rešenja.
4. **„Zašto poređenje geometrijom, a ne po nazivima ulica?"** → Slajd 9: dve rute mogu imati potpuno različitu strukturu manevara; poređenje po poziciji u nizu otkazuje čim se rute globalno raziđu.
5. **„Da li je sistem testiran sa stvarnim vozilom?"** → Nije; GPS pozicije dolaze sa stvarnog uređaja, ali ne iz kamiona u vožnji. Provera blizine polazišta od 500 m je uvedena upravo da bi prva pozicija bila stvaran GPS fix, a ne proizvoljna vrednost.
6. **„Koja je razlika Dijkstra i A* u vašim merenjima?"** → Ista cena puta (dopustiva heuristika), A* pregleda manje čvorova, 17,1 vs. 19,3 ms — razlika je mala jer je graf mali i dominira parsiranje.

## Praktični saveti za izlaganje

- Ukupno 13:30 ostavlja rezervu do 15 min; ako kasniš, skrati slajdove 5, 12 i 13 na po jednu rečenicu. Štoperica u prezentaciji (taster `T`) boji vreme crveno kad zaostaješ više od 30 s za planom.
- Na slajdovima 8, 9, 10 i 14 ne čitaj sa slajda — tu je najviše sadržaja i tu se najviše boduje razumevanje.
- Kod na slajdovima drži na maksimum 10–12 linija, font ne manji od 16 pt; sve što je duže ide u backup.
- Uvežbaj naglas bar dvaput sa štopericom — pisani plan uvek zvuči kraće nego što se izgovori.
