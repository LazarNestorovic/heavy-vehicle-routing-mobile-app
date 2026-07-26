# Kako ručno testirati rundu 2 (Blokovi D-M) na uređaju

Ovo je vodič za tebe da na svom telefonu/aplikaciji prođeš kroz sve što je implementirano u `documentations/features/2026-07-26-live-gps-tracking.md`, `2026-07-21-bounded-astar-dijkstra.md` (dopuna) i `2026-07-26-scope-cut-closures.md`. Sve što je bilo moguće već je provereno preko curl-a protiv Docker stack-a (vidi te dokumente) — ovo su koraci za ono što curl ne može da pokrije: vizuelni izgled, GPS, gestovi.

## Priprema (jednom)

1. Backend mora da radi: `docker compose up -d` u root folderu projekta.
2. `mobile/lib/config.dart` već ima tvoju LAN IP adresu i Google client ID popunjene — telefon mora biti na ISTOJ Wi-Fi mreži kao računar.
3. Pokreni app: `cd mobile && flutter run` (vidi `documentations/guides/run-flutter-app.md` za detalje).
4. Za Blokove F/G (email) — SMTP je već podešen u `.env`, mejlovi stvarno stižu na `lazarnestorovic003@gmail.com`.
5. Za Blok K treba ti DVA naloga — jedan `dispatcher`, jedan `driver` koji ga je odobrio (registruj oba u app-u ako ih još nemaš, isprati "Zahtevi dispečera" tok da ih povežeš).

Redosled ispod prati blokove iz plana — možeš ih raditi bilo kojim redom, nezavisni su.

---

## Blok D — Pravo GPS praćenje

1. Uloguj se kao samostalan vozač (bez dispečera), napravi vozilo ako nemaš, isplaniraj rutu (tap na mapu ili pretraga adrese) i pritisni **"Kreni na put"**.
2. Na `ActiveTripScreen`-u telefon će tražiti dozvolu za lokaciju — **odobri je**.
3. Posmatraj plavi/ljubičasti marker na mapi: umesto skriptovane 60-sekundne animacije, sad treba da prati TVOJU stvarnu poziciju (pomeri se sa telefonom, ili samo sačekaj GPS da se stabilizuje).
4. U AppBar-u se pojavljuje ikonica zastavice — to je dugme **"Stigao sam"**, vidljivo samo dok je GPS aktivan. Pritisni ga na kraju — treba da iskoči dijalog "Stigli ste".
5. **Provera auto-reconnect-a**: dok je tura aktivna, uključi pa isključi avionski režim na 5-10 sekundi. Mapa treba sama da se oporavi (nova pozicija stigne) bez potrebe da zatvoriš/otvoriš ekran.
6. **Bonus (dispečerova mapa)**: ako imaš dispečer nalog koji upravlja ovim vozačem, otvori na drugom telefonu/emulatoru dispečerovu "Vozila uživo" mapu dok je tura aktivna — treba da prikazuje ISTU stvarnu poziciju, ne simulaciju.

## Blok E — Bounded A*/Dijkstra (algoritamski modul za rad, nema UI)

Ovo je eksperimentalni modul za tezu, nije deo `/routes` API-ja koji app koristi — nema šta da se klikne u aplikaciji. Provera je isključivo preko testova:

```bash
cd backend
go test ./internal/algorithm/... -v
```

Trebalo bi da vidiš 14 testova, svi `PASS`, uključujući `TestRealCorridor_ParsesNodeBarrierHeightTag` i `TestDijkstra_NodeBarrierExcludesTallVehicle` (novi u ovoj rundi).

## Blok F — Reset lozinke + odjava sa svih uređaja

**Reset lozinke:**
1. Na ekranu za prijavu, pritisni **"Zaboravljena lozinka?"**.
2. Unesi email nekog naloga koji ima email postavljen (registruj novi nalog sa email-om ako nemaš pogodan).
3. Proveri inbox — stiže mejl "Resetovanje lozinke". Otvori link (otvara se u browseru na telefonu/računaru).
4. Unesi novu lozinku na toj web stranici, pošalji.
5. Vrati se u app, probaj login sa STAROM lozinkom (treba da odbije), pa sa NOVOM (treba da uspe).

**Odjava sa svih uređaja:**
1. Uloguj se, idi na **Profil**.
2. Pritisni **"Odjavi sve uređaje"** — treba da te vrati na ekran za prijavu (isto kao obična odjava, lokalno).
3. Da bi VIDEO da su i DRUGI uređaji stvarno odjavljeni, treba ti drugi uređaj (ili drugi emulator) ulogovan na isti nalog ISTOVREMENO — nakon što na prvom pritisneš "Odjavi sve uređaje", drugi uređaj gubi pristup na sledećem pozivu (npr. otvori bilo koji ekran koji zove API — dobiće grešku i trebalo bi da ga app izbaci na login).

## Blok G — Rest-stop poboljšanja (uglavnom backend, suptilno vidljivo)

1. Napravi vozilo sa **hazmat: uključeno**.
2. Isplaniraj DUGU rutu (preko 4.5h vožnje) — npr. Subotica → Vranje preko pretrage adresa.
3. Pritisni "Kreni na put".
4. Za par sekundi (worker obrađuje u pozadini) treba da iskoči snackbar "Predlog pauze: [ime stanice]".
5. Ovo je suptilna promena — glavna razlika (da predložena stanica stvarno leži NA ruti, ne samo vazdušno blizu jedne tačke) se ne vidi golim okom bez poređenja sa starim ponašanjem. Ako želiš da vizuelno uporediš, napravi istu rutu sa hazmat isključeno i pogledaj da li se predložena stanica razlikuje.

## Blok H — Geometrijski presek putanja (objašnjenje rute)

1. Napravi vozilo sa **visinom 4.7m** (ostalo standardno).
2. Isplaniraj rutu Beograd → Novi Sad (centri gradova, preko pretrage adresa).
3. Pritisni **"Pregled rute"**.
4. Ispod distance/trajanja treba da se pojavi kurzivna napomena tipa *"Ruta skreće kod [ulica] jer visina vozila (4.7m) ne zadovoljava ograničenje na toj deonici."*
5. Promeni visinu na 4.0m, ponovi pregled — napomena treba da NESTANE (nema stvarnog ograničenja za tu visinu na ovoj ruti).

## Blok I — "Zašto" poruka za omiljenu pumpu

1. Idi na **Preference** → **Omiljene lokacije** → dugme za dodavanje.
2. Pretraži ili tapni tačku NA putu kojim ćeš praviti rutu (npr. blizu Beograda ako ćeš praviti rutu koja prolazi kroz Beograd), sačuvaj.
3. Isplaniraj rutu koja prolazi blizu te tačke, pritisni "Pregled rute".
4. Treba da vidiš napomenu: *"Ruta prolazi blizu vaše omiljene pumpe "[ime]"."* (ista kartica kao za Blok H, samo drugi razlog).

## Blok J — Alternativne rute na mapi

1. Isplaniraj bilo koju rutu, pritisni **"Pregled rute"**.
2. Pored obojene izabrane rute, na mapi treba da vidiš 1-2 dodatne TANJE SIVE linije — to su alternative koje je Valhalla razmatrala, a nisu izabrane.

## Blok K — Dispečerov pristup vozačevim ličnim vozilima

1. Uloguj se kao vozač, dodaj LIČNO vozilo (ako nemaš).
2. Uloguj se kao dispečer koji upravlja tim vozačem (ako veza ne postoji, uspostavi je preko "Zahtevi dispečera" toka).
3. Kao dispečer, otvori tog vozača i napravi novu turu.
4. Otvori padajući meni za vozilo — treba da vidiš I flotna vozila (označena "Flota") I vozačevo lično vozilo (označeno njegovim korisničkim imenom), zajedno u istoj listi.

## Blok L — Pretraga adresa

1. Na bilo kom ekranu za planiranje rute (ili biraču omiljene lokacije), ukucaj naziv mesta (npr. "Novi Sad") u novo polje za pretragu na vrhu, pritisni ikonicu lupe.
2. Treba da se pojavi lista stvarnih rezultata ispod polja.
3. Tapni jedan — mapa se centrira na njega i postavlja marker (polazak/odredište/omiljena lokacija, zavisno od ekrana), isto kao da si tapnuo direktno na mapu.

## Blok M — Izmena/brisanje vozila

1. Idi na **"Moja vozila"**.
2. Na bilo kom vozilu pritisni meni (tri tačke) → **"Izmeni"** — forma treba da bude POPUNJENA postojećim vrednostima. Promeni nešto (npr. težinu), sačuvaj — lista treba da se osveži sa novom vrednošću.
3. Probaj **"Obriši"** na vozilu koje NEMA nijednu turu — treba da nestane iz liste odmah nakon potvrde.
4. Probaj **"Obriši"** na vozilu koje IMA bar jednu turu (napravljenu u Bloku D/G/H/itd. testiranju) — treba da dobiješ crvenu poruku o grešci (409, vozilo se ne briše) umesto da nestane.

---

## Ako nešto ne radi

Za svaki blok postoji odgovarajuća sekcija u dokumentaciji (`documentations/features/2026-07-26-*.md`) sa tačnim opisom šta je implementirano i kako je već provereno preko curl-a — korisno da uporediš da li je problem u UI-ju ili je nešto stvarno pokvareno u pozadini. Ako naiđeš na grešku, javi mi tačno na kom koraku i šta se desilo (poruka o grešci, screenshot) pa nastavljamo odatle.
