# Traka "Email nije potvrđena" se nije sklanjala posle verifikacije + email obavezan

**Datum:** 2026-07-26
**Fajlovi:** `backend/internal/httpapi/auth.go`, `backend/internal/httpapi/server.go`, `mobile/lib/services/{api_client,auth_storage}.dart`, `mobile/lib/widgets/email_verification_banner.dart`, `mobile/lib/screens/register_screen.dart`

Korisnik je uživo testirao email verifikaciju (`2026-07-25-google-maps-signin-email-verification.md`) i prijavio da traka upozorenja ne nestaje ni posle klika na link u email-u. Uzrok nije bio bag u smislu pogrešnog koda — to je bilo NAMERNO ponašanje iz prošle runde ("app saznaje da je email potvrđen tek na sledećem loginu") — ali se korisniku ispravno činilo kao bag, jer ništa u UI-ju nije govorilo da je re-login potreban niti je postojao lak način da se status osveži bez potpune odjave.

## Popravka

- Nov `GET /api/v1/auth/me` (autentifikovano) — vraća trenutno stanje naloga iz baze (`username`, `role`, `dispatcher_id`, `email`, `email_verified`), bez izdavanja novog tokena.
- `ApiClient.refreshAccountStatus()` poziva ga i upisuje `email_verified`/`dispatcher_id` u `AuthStorage` preko novog `saveEmailVerified()` (ciljana izmena samo tog polja, isti obrazac kao postojeći `saveDispatcherId()`).
- `EmailVerificationBanner` sad implementira `WidgetsBindingObserver` i osvežava status na `AppLifecycleState.resumed` — tačno u trenutku kad se vozač vrati u app posle klika na link u browseru. Provera se dešava samo dok traka realno može biti prikazana (`email != null && !emailVerified`), da ne pinguje server bez potrebe.
- Tekst poruke posle "Pošalji ponovo" ažuriran — više ne traži re-login, samo povratak u app.

## Dodatan zahtev: email obavezan pri registraciji

Do sada opciono (`credentialsRequest.Email *string`, `omitempty` validacija). `validate()` sad zahteva da `Email` bude postavljen i validan; `register_screen.dart` polje više nema "(opciono)" oznaku, validator vraća "Obavezno polje" na praznom unosu, `ApiClient.register` prima `required String email`.

**Napomena**: ovo utiče samo na NOVE registracije korisničko ime/lozinka putem. Google prijava i dalje sama diktira email (dolazi iz Google claim-a, uvek prisutan). Postojeći nalozi bez email-a ostaju kako jesu, ništa retroaktivno.

## Verifikacija

Uživo protiv Docker stack-a: registracija bez email-a → `400 "email is required and must be valid"`; registracija sa email-om → `201`; `GET /auth/me` odmah posle registracije → `email_verified:false`; token iz baze → `GET /verify-email` → ponovni `GET /auth/me` (ISTIM tokenom, bez re-logina) → `email_verified:true`. `go build`/`vet`/`test` i `flutter analyze`/`test` čisti.
