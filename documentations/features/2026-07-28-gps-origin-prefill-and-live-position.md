# Polazna tačka: "Trenutna pozicija" po podrazumevanom + dva odvojena unosa adrese + vozačeva pozicija stalno na mapi + reverse geocoding

**Datum:** 2026-07-28
**Fajlovi:** `backend/internal/geocode/geocode.go`, `backend/internal/httpapi/{geocode,server}.go`, `mobile/lib/services/{location_service,api_client}.dart`, `mobile/lib/widgets/address_search_field.dart`, `mobile/lib/screens/{route_request,trip_detail}_screen.dart`

Tri stavke iz `BELESKE.txt` (druga verzija — prva iteracija je auto-popunjavala i dozvoljavala prevlačenje polaznog markera; korisnik je posle testiranja tražio drugačiji mehanizam, opisan ispod):
- Polazna tačka ne treba da se automatski POSTAVLJA na trenutnu poziciju (kao fiksna vrednost) — treba da bude prikazana kao "Trenutna pozicija" po podrazumevanom, dinamički (prati GPS), dok je vozač eksplicitno ne promeni.
- Dva odvojena tekstualna polja za unos adrese — jedno za polaznu, jedno za odredišnu tačku.
- Kad je polazna tačka "trenutna pozicija", klik na to polje otvara izbor: unos nove adrese ILI nova opcija — biranje tačke dodirom na mapu.
- Odredišna tačka ostaje IDENTIČNA postojećem ponašanju (pretraga adrese + dodir na mapu) — eksplicitno zatraženo da se ne menja.

## Mehanizam

**Polazna tačka** (`RouteRequestScreen`):
- `_origin == null` znači "trenutna pozicija" — MOD, ne zamrznuta tačka. `_effectiveOrigin` getter (`_origin ?? _myPosition`) je tačka koja se stvarno šalje backend-u pri pregledu/pokretanju, pa dok je mod aktivan, polazište prati vozačevo kretanje uživo (svaki GPS tik ažurira zeleni marker).
- Polje "Polazna tačka" na vrhu ekrana (`InputDecorator` + `InkWell`, izgleda kao tekstualni unos ali je zapravo dugme) prikazuje "Trenutna pozicija" ili adresu/izabranu tačku. Klik otvara `showModalBottomSheet` sa tri opcije:
  1. Pretraga adrese (reused `AddressSearchField`) — bira konkretnu tačku, izlazi iz "trenutna pozicija" moda.
  2. "Trenutna pozicija" — vraća na dinamički mod.
  3. "Izaberi tačku na mapi" — naoružava `_pickingOriginOnMap`; sledeći dodir na mapu postavlja polazište (umesto odredišta) i automatski razoružava mod.

**Odredišna tačka**: nepromenjena — zaseban `AddressSearchField` (sad sa jasnom oznakom "Odredište (adresa)" umesto generičkog naslova, pošto sad postoje dva polja) i dodir na mapu (kad `_pickingOriginOnMap` NIJE aktivan, dodir uvek ide na odredište, tačno kao pre).

**Vozačeva pozicija na mapi**: plavi "Vaša pozicija" marker (uveden u prvoj iteraciji ovog fičera) ostaje nepromenjen — kontinuirano se ažurira, odvojen od zelenog polaznog markera. Isti marker na `TripDetailScreen` takođe nepromenjen.

## Dopuna: reverse geocoding za tačke izabrane dodirom na mapu

Kad se odredište (ili polazište, preko "Izaberi tačku na mapi") postavi dodirom na mapu, tekstualno polje je do sada ostajalo prazno/pokazivalo koordinate — korisnik je tražio da se umesto toga automatski upiše naziv adrese te tačke.

- Nov `Client.Reverse(ctx, lat, lon)` u `internal/geocode` — mirror postojećeg `Search`, poziva Nominatim `/reverse?lat=&lon=&format=jsonv2`, vraća `display_name`. Isto grlo za throttling (1 req/s) kao i pretraga, pošto oba prolaze kroz isti `Client`.
- Nov `GET /api/v1/geocode/reverse?lat=&lon=` (`handleReverseGeocode`) — vraća isti `{lat, lon, display_name}` oblik kao pretraga, pa Flutter strana ponovo koristi postojeći `GeocodeResult.fromJson`.
- Flutter `ApiClient.reverseGeocode(lat, lon)` — tanka obertka oko novog endpoint-a.
- `AddressSearchField` dobija opcioni `controller` parametar (spolja dodeljen `TextEditingController`) — bez ovoga roditeljski ekran nije imao način da programski upiše tekst u polje (widget je do sada uvek upravljao sopstvenim internim kontrolerom). `RouteRequestScreen` sad prosleđuje `_destinationCtrl` polju za odredište.
- Dodir na mapu (za odredište ILI za polazište preko "Izaberi tačku na mapi") odmah postavlja tačku, čisti tekstualno polje, i u pozadini poziva reverse geocoding; kad rezultat stigne, upisuje adresu u polje (`_destinationCtrl.text = ...` za odredište, `_originLabel` za polazište). Best-effort — ako reverse geocoding ne uspe (bez interneta, nema adrese za tu tačku), sama tačka ostaje potpuno upotrebljiva, polje samo ostaje prazno/pokazuje koordinate.
- Zaštita od zastarelog odgovora: ako korisnik ponovo dodirne mapu pre nego što prethodni reverse geocoding odgovori, rezultat prve pretrage se odbacuje (`_destination != point` / `_origin != point` provera) umesto da prepiše noviju tačku pogrešnim tekstom.

## Namerno uklonjeno iz prve iteracije

Prevlačenje (`draggable`) zelenog markera je uklonjeno — zamenjeno bottom-sheet mehanizmom (pretraga adrese / dodir na mapu / trenutna pozicija) po eksplicitnom zahtevu. Auto-popunjavanje `_origin` GPS pozicijom pri otvaranju ekrana je takođe uklonjeno — polazište ostaje u "trenutna pozicija" modu (dinamički, ne zamrznuto) dok se eksplicitno ne promeni.

## Verifikacija

`go build`/`vet`/`test` i `flutter analyze`/`flutter test` čisti. Uživo (curl protiv Docker stack-a): `GET /api/v1/geocode/reverse?lat=44.8125&lon=20.4612` vraća čitljivu adresu (Trg Nikole Pašića, Beograd); poziv bez `lat`/`lon` vraća `400`. Ostatak (bottom sheet, dodir na mapu, kontinuirano ažuriranje pozicije) zahteva fizički uređaj — isto ograničenje kao za sav raniji GPS-zavisan rad.
