# Dispečerov ekran za kreiranje ture usklađen sa RouteRequestScreen redizajnom

**Datum:** 2026-07-31
**Fajlovi:** `mobile/lib/screens/dispatcher_create_trip_screen.dart`

`DispatcherCreateTripScreen` (dispečerov pandan za `RouteRequestScreen`) je ostao na starom dizajnu dok je `RouteRequestScreen` u međuvremenu prošao kroz niz izmena (dva odvojena polja za adresu, reverse geocoding, kompaktan panel koji se širi preko mape, "lepljiva" dugmad na dnu). Ovaj zapis prenosi te izmene ovde, uz jednu namernu razliku.

## Namerna razlika: polazna tačka je OBIČNO polje za adresu, bez GPS moda i bez birača

`RouteRequestScreen`-ova polazna tačka podrazumevano prati vozačevu živu GPS poziciju i ima poseban bottom-sheet birač (adresa / trenutna pozicija / tačka na mapi) — ima smisla jer vozač i planira i vozi. Za dispečera to ne važi: dispečer nije taj ko vozi, pa GPS **dispečerovog** telefona nema nikakve veze sa tim gde se vozač/kamion stvarno nalazi. Ovo je eksplicitno potvrđeno sa korisnikom pre implementacije, i korisnik je zatim zatražio da polazna tačka bude **potpuno isto polje kao odredišna** — obična `AddressSearchField`, bez ikakvog posebnog birača.

Posledica: `_handleOriginSelect`/`_handleDestinationSelect` su sad simetrični (isti obrazac, samo drugo ciljno polje), a dodir na mapu koristi jednostavan fallback koji je `RouteRequestScreen` imao PRE GPS moda — prvi dodir (dok polazna tačka nije postavljena) postavlja nju, svaki sledeći postavlja odredište.

## Preneto 1:1

- Dva odvojena, simetrična polja: "Polazna tačka (adresa)" i "Odredište (adresa)", oba obična `AddressSearchField` sa spoljnim kontrolerom.
- Reverse geocoding pri dodiru na mapu (za oba polja) — `ApiClient.reverseGeocode` (već postoji, korišćen bez izmena).
- Mapa zauzima skoro ceo ekran; donji panel (greška, pregled rute, tovar, dugmad) lebdi preko nje (`Stack` + `AnimatedContainer`), kompaktan podrazumevano (150px), širi se na 60% visine ekrana kad se otvori "Podaci o tovaru".
- Dugmad "Pregled rute"/"Ponudi turu" prikovana za dno panela (sticky footer, isti `Expanded` + `SingleChildScrollView` obrazac).
- Kompaktniji prikaz greške i sažetka rute (manji padding/font).
- Dodato "Resetuj tačke" dugme u AppBar.

## Namerno NE preneto

- "Vaša pozicija" plavi marker (uvek prikazana živa pozicija) — isti razlog kao GPS mod: dispečerova pozicija nije relevantna za rutu koju vozač vozi.
- Draggable marker za polaznu tačku, bottom-sheet birač za polaznu tačku — `RouteRequestScreen` je i sam prošao kroz tu fazu pa je odbacio u korist običnog polja; ovde je otišlo pravo na finalni, jednostavniji oblik.

## Verifikacija

`flutter analyze`/`flutter test` čisti. Stvarno ponašanje (dodir na mapu, reverse geocoding, širenje panela) zahteva fizički uređaj — isto ograničenje kao za `RouteRequestScreen`-ov originalni redizajn.
