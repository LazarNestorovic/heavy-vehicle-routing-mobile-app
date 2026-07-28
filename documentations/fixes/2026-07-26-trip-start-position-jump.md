# Trzaj pozicije pri pokretanju ture (simulacija-pa-stop)

**Datum:** 2026-07-26
**Fajlovi:** `backend/internal/ws/gateway.go`

Korisnik je prijavio da se pri pritisku na "Pokreni" pozicija na mapi vidno pomeri malo ka odredištu, kao da se simulacija na trenutak pokrene pa prestane. Postavljeno je i pitanje da li simulaciju treba potpuno ukloniti — korisnik je izabrao da se simulacija ZADRŽI (i dalje koristan fallback kad GPS dozvola nije data), ali da se trzaj ukloni.

## Uzrok

`HandleTripStream` je odmah pri konekciji, bez čekanja, proveravao da li već postoji stvaran GPS ping za tu turu; ako ne, odmah je ulazio u `simulate()`. Telefonov prvi GPS fix uobičajeno stiže sa kašnjenjem od sekundu-dve i posle odobrene dozvole, pa bi 1-2 simulirana tick-a stigla da pomere marker pre nego što stvarna pozicija preuzme prikaz.

## Prva popravka (uvedena, pa zamenjena posle testa)

Prvi pokušaj je dodao `awaitLiveOrTimeout` — pollovanje na svakih 250ms do 3 sekunde, pre pada na `simulate()`. Ovo je otklonilo trzaj u osnovnom slučaju (nema GPS-a uopšte), ali je uživo test otkrio pravu trku: `awaitLiveOrTimeout` je samo proveravao POSTOJANJE `liveTrip` objekta, bez pretplate na njega. Ako bi `ReportPosition` emitovao (broadcast) baš u razmaku između dva pollovanja, taj broadcast je stizao na NULA pretplatnika i bio trajno izgubljen (`liveTrip.broadcast()` je ne-blokirajući i ne pamti/ponavlja propuštene poruke) — WS konekcija bi zatim, na sledećem pollu, pronašla `liveTrip` i pretplatila se, ali PRVI pravi ping je već bio izgubljen.

## Konačna popravka — pretplata pre odluke

`HandleTripStream` se sad ODMAH pri konekciji pretplaćuje na turin deljeni `liveTrip` objekat (`liveTripFor` sad UVEK kreira/vraća isti objekat, bez zasebne "samo proveri" grane), pre nego što odluči da li da čeka ili da padne na simulaciju:

- Ako `liveTrip` već ima stvarne podatke (`isLive()`) — npr. dispečerova live mapa se priključuje usred ture — odmah prelazi na relay, bez čekanja.
- Ako nema — čeka do 3 sekunde NA VEĆ OTVORENOM kanalu (`select` na kanalu ili timeout-u); pošto je pretplata već aktivna, nijedan broadcast koji stigne u tom prozoru ne može biti propušten.
- Ako ni posle 3 sekunde ništa ne stigne — otkazuje pretplatu i pada na `simulate()`, nepromenjeno.

Dodato je `liveTrip.hasData` polje da razdvoji "objekat postoji" (može postojati čisto zato što WS konekcija čeka) od "objekat je stvarno nosio bar jedan pravi GPS ping" — `simulate()`-ova provera usred simulacije (da li je u međuvremenu stigao pravi ping) sad koristi `liveTripIfLive()` umesto gole provere postojanja, iz istog razloga.

## Namerni obim-cut

Deljeni `liveTrip` objekat za turu koja NIKAD ne dobije pravi GPS ping (npr. dozvola trajno odbijena) ostaje u memoriji do kraja života servera (briše se samo eksplicitnim `CompleteTrip`, tj. "Stigao sam"). Na skali teze/demoa (desetine-stotine tura) ovo je zanemarljivo; nije dodavana dodatna logika čišćenja da se ne komplikuje kod bez stvarne potrebe.

## Verifikacija

`gofmt`, `go build`/`vet`/`test` čisti. Uživo (Node `ws` klijent protiv Docker stack-a), dva scenarija:
- Bez GPS-a uopšte: prva poruka stiže tek u +3022ms, sa `progress_fraction=0` (svež početak, bez trzaja).
- Pravi GPS stigne za vreme čekanja (+1022ms): WS klijent prima tu poziciju već u +1032ms — nijedan gubitak, bez ikakve simulacije u međuvremenu.
