# Google Maps (prikaz), Google prijava, i potvrda email adrese

**Datum:** 2026-07-25
**Plan:** `/home/lazar/.claude/plans/lovely-cooking-puppy.md` (plan mode, odobren pre implementacije, uz 4 pitanja korisniku)
**Fajlovi:** `mobile/lib/screens/*.dart`, `mobile/lib/services/{google_auth,polyline,api_client,auth_storage}.dart`, `backend/internal/{auth/google,mailer,store/{driver,email_verification_token}}.go`, `backend/internal/httpapi/auth.go`, `backend/internal/db/migrations/00003_google_and_email_auth.sql`

## Zašto

Tri odvojene integracije iz istog razgovora:

1. **Google Maps — samo prikaz.** `flutter_map`/OSM tajlovi zamenjeni `google_maps_flutter`-om, ali **Valhalla ostaje jedini koji računa rutu** (visina/težina/osovinsko opterećenje/hazmat costing). Google-ove rute nemaju istu podršku za teretna vozila i naplaćuju se po pozivu — zamena bi oslabila baš onaj deo projekta koji ga čini "heavy vehicle routing" tezom. Google Maps SDK samo iscrtava već izračunatu Valhalla rutu na lepšoj podlozi. Ova podela je bila eksplicitna odluka korisnika (jedna od 4 odgovora pre plana).
2. **Google prijava — dodatna opcija**, ne zamena. Korisničko ime/lozinka forma ostaje nepromenjena, svi postojeći test nalozi rade i dalje.
3. **Email + potvrda — potpuno ručna implementacija**, ne Firebase (korisnik je eksplicitno izabrao ovo zbog manje zavisnosti i pune kontrole, dosledno projektnom stilu bez ORM-a/frameworka).

## Blok A — Google Maps SDK

- `pubspec.yaml`: `flutter_map`/`latlong2` uklonjeni, `google_maps_flutter: ^2.12.3` dodat.
- API ključ NIKAD u git-u: čita se iz gitignore-ovanog `android/local.properties` (`MAPS_API_KEY=...`), prosleđuje se kao Gradle manifest placeholder (`build.gradle.kts` → `AndroidManifest.xml` → `<meta-data android:name="com.google.android.geo.API_KEY">`).
- `services/polyline.dart` — `LatLng` sad iz `google_maps_flutter` umesto `latlong2` (oblikom identičan, praktično bezbolna izmena).
- 6 ekrana migrirano na isti obrazac: `route_request_screen.dart`, `dispatcher_create_trip_screen.dart`, `active_trip_screen.dart`, `dispatcher_live_map_screen.dart`, `trip_detail_screen.dart`, `preferences_screen.dart`. `MarkerLayer`/`PolylineLayer` → `Set<Marker>`/`Set<Polyline>`, `MapController` → `GoogleMapController` (dobijen iz `onMapCreated`) + `animateCamera`, `BitmapDescriptor.defaultMarkerWithHue(...)` umesto proizvoljnih widget markera (stvarno smanjenje mogućnosti flutter_map-a — boje očuvane preko hue konstanti: zeleno=polazak, crveno=odredište, ljubičasto=kamion, narandžasto=pumpa/odmorište, žuto=omiljena lokacija).
- `trip_detail_screen.dart` — `LatLngBounds.fromPoints` ne postoji u google_maps_flutter, pa je dodat ručni min/max lat/lon getter + `CameraUpdate.newLatLngBounds` pozvan iz `_onMapCreated`.
- Novi vodič `documentations/guides/google-maps-setup.md` — korak-po-korak GCP konzola (projekat, billing, "Maps SDK for Android", SHA-1 fingerprint preko `./gradlew signingReport`, API ključ + restrikcija).

**Verifikacija**: `flutter analyze`/`flutter test` čisti. Vizuelni izgled mape/tajlova/markera nije proveren — zahteva korisnikov uređaj i pravi API ključ (placeholder trenutno u `local.properties`).

## Blok B — Google prijava

- `backend/internal/auth/google.go` — `GoogleVerifier` oko `github.com/MicahParks/keyfunc/v3` (preuzima/kešira Google-ov JWKS), integrisan sa postojećim `golang-jwt/jwt/v5`. Namerno mala zavisnost umesto ručnog RSA/JWKS parsiranja (osetljiv kriptografski kod) ili celog `google.golang.org/api` paketa. `VerifyIDToken` proverava potpis/`iss`/`aud`(=`GOOGLE_CLIENT_ID`)/`exp`.
- `store/driver.go` — `PasswordHash` promenjen u `*string` (Google-only nalog nema lozinku, migracija diže NOT NULL ograničenje). Novi `GetByGoogleSub`, `GetByEmail`, `LinkGoogleSub`, `CreateGoogle`.
- `POST /api/v1/auth/google` (`httpapi/auth.go`, `handleGoogleAuth`) — tri-putanjska rezolucija naloga:
  1. `google_sub` se poklapa → login.
  2. Ne poklapa se, ali `email` se poklapa sa postojećim korisničko ime/lozinka nalogom → `LinkGoogleSub` pa login (spaja naloge).
  3. Nijedno → `CreateGoogle` (nov nalog, `email_verified=true` direktno iz Google claim-a, jer Google to već garantuje).
  Vraća 503 ako `GOOGLE_CLIENT_ID` nije podešen na serveru (graceful degradation, ne pada backend).
- `mobile/lib/services/google_auth.dart` — omotava `google_sign_in: ^7.2.0`, novi singleton API (`GoogleSignIn.instance.initialize(serverClientId:...)` pa `.authenticate()`); otkazivanje od strane korisnika (`GoogleSignInExceptionCode.canceled`) tretirano kao `null`, ne greška.
- `LoginScreen`/`RegisterScreen` — dugme "Nastavi sa Google nalogom" pored postojeće forme (`RegisterScreen` šalje trenutno izabranu ulogu; `LoginScreen` je šalje samo kao podrazumevanu za slučaj da je nalog nov).
- Deljeni `entry_router.dart` → `applySession(api, result)` helper da se izbegne trostruko ponavljanje ~7-poljnog assignment-a preko login/register/Google puteva. `authResponse` na backendu dobio `Username` polje baš zbog ovoga (Flutter ne mora posebno da prati "šta sam ukucao" vs "šta je Google/backend generisao").

**Verifikacija**: `go build`/`vet`/`test` čisti. `POST /auth/google` sa lažnim tokenom → ispravno `503` (server nije konfigurisan). Puna provera (pravi Google ID token, spajanje naloga) **ostaje na korisniku** — zahteva provizioniran GCP OAuth klijent (`googleServerClientId` u `mobile/lib/config.dart` i `GOOGLE_CLIENT_ID` env var su trenutno prazni placeholder-i) i fizički uređaj; nije proverljivo preko curl-a.

## Blok C — Email + potvrda

- `email_verification_tokens` tabela (ista `00003` migracija) — 32-bajtni hex token (`crypto/rand`), 24h TTL.
- `internal/mailer/mailer.go` — stdlib `net/smtp`, bez MIME biblioteke. `Enabled()` vraća `false` ako `SMTP_HOST` nije podešen — `Send()` postaje tihi no-op umesto greške (isti obrazac graceful degradation-a kao Google prijava).
- **SMTP header injection zaštita** (samostalno uočeno, ne prijavljeno od korisnika): `to`/`subject` su korisnički unos (vozačev sopstveni email), a poruka se ručno sastavlja preko `fmt.Sprintf` — dodat `stripCRLF` da spreči ubacivanje dodatnih zaglavlja.
- `POST /auth/register` prima opcioni `email` → `sendVerificationEmailIfNeeded` šalje link `GET {PUBLIC_BACKEND_URL}/api/v1/auth/verify-email?token=...`.
- `GET /api/v1/auth/verify-email` (javan, bez JWT-a — token je sam po sebi dokaz) — renderuje **prostu HTML stranicu direktno sa backend-a** (ne redirect nazad u Flutter app — namerni obim-cut, izbegava Android App Links/iOS Universal Links komplikovanost). Razlikuje istekao/iskorišćen/nepostojeći token različitim porukama.
- `POST /api/v1/auth/resend-verification` (autentifikovano) — nov token, 409 ako je već potvrđen.
- `mobile/lib/widgets/email_verification_banner.dart` — traka na sve tri home-ekrana uloga (`vehicle_list_screen.dart` za samostalnog vozača, `offered_trips_screen.dart` za upravljanog vozača, `dispatcher_home_screen.dart` za dispečera), prikazuje se samo kad `api.email != null && !api.emailVerified`, dugme "Pošalji ponovo" zove resend endpoint.
- Flutter saznaje da je email potvrđen tek na SLEDEĆEM loginu (nema polling/refresh mehanizam) — eksplicitno naznačeno u UI tekstu ("Poslato! Klikni link u email-u, pa se ponovo uloguj da se status osveži.").

**Verifikacija uživo (curl + ručno ubačen token iz baze)**: migracija primenjena čisto; registracija sa email-om vraća `email_verified:false`; nevalidan email (`"not-an-email"`) → 400; token pulled iz `docker exec hvr-postgres psql`, `GET /auth/verify-email?token=...` → HTML uspeh, `email_verified` postaje `true` na sledećem loginu; ponovna upotreba istog tokena → 400 "već iskorišćen"; resend na nepotvrđenom nalogu → nov token kreiran, na već potvrđenom → 409. Stvarna isporuka email-a (pravi SMTP) **ostaje na korisniku** — `SMTP_HOST` je trenutno prazan placeholder u `docker-compose.yml`.

## Namerni obim-cut-ovi

- Nema deep-linking-a iz email linka nazad u app.
- Nepotvrđen email ništa ne blokira u aplikaciji, samo prikazuje upozorenje.
- iOS namerno van dometa (projekat se testira isključivo na fizičkom Android uređaju, `mobile/ios/` nikad nije build-ovan u ovoj sesiji).

## Šta ostaje korisniku

1. GCP provizioning (`documentations/guides/google-maps-setup.md`): Maps API ključ + OAuth Web/Android client ID-jevi, popuniti `mobile/lib/config.dart` (`googleServerClientId`, `apiBaseUrl` ako treba) i `GOOGLE_CLIENT_ID` env var na backendu.
2. SMTP kredencijali (Gmail App Password ili Mailtrap sandbox) u `docker-compose.yml`.
3. Vizuelna provera Google Maps prikaza i pun end-to-end test Google prijave/email potvrde na uređaju.
