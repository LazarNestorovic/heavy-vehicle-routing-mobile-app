# RabbitMQ minimalni tok: trip.started → worker → trip.eta_updated

**Datum:** 2026-07-21
**Fajlovi:** [`backend/internal/queue/`](../../backend/internal/queue/), [`backend/internal/worker/`](../../backend/internal/worker/), [`backend/internal/httpapi/trips.go`](../../backend/internal/httpapi/trips.go), [`docker-compose.yml`](../../docker-compose.yml)

## Šta je dodato

Poslednji deo nedelje 2 iz `SPECIFIKACIJA.md` (sekcija 3.6). `docker-compose.yml` dobija `rabbitmq` servis (`rabbitmq:3-management-alpine`, AMQP na `5672`, management UI na `15672`, kredencijali `hvr`/`hvr_dev_password`).

```
backend/internal/queue/
  queue.go   — tanak wrapper: Connect, Publish, Consume nad JEDNIM topic exchange-om "trip.events"
  events.go  — TripStartedEvent, TripETAUpdatedEvent, routing key konstante
backend/internal/worker/
  trip_worker.go — konzumira trip.started, računa pojednostavljeni rest-stop predlog, upisuje u DB, publikuje trip.eta_updated
```

`POST /api/v1/trips` (`trips.go`) posle uspešnog upisa u bazu publikuje `trip.started` sa `{trip_id}` — namerno **ne** propagira grešku publish-a nazad klijentu (trip je već perzistiran, to je izvor istine; RabbitMQ hiccup se samo loguje).

Worker radi kao goroutine unutar **istog** binarnog fajla kao HTTP server (`main.go`), ne kao poseban servis/kontejner — nema još operativne potrebe da se skaliraju nezavisno za MVP demo. Koriste se **dve odvojene AMQP konekcije** (jedna za publisher u HTTP handleru, jedna za worker-ov consumer), pošto `amqp091-go` kanal nije bezbedan za konkurentnu upotrebu iz više gorutina.

## Pravilo za predlog pauze (namerno pojednostavljeno, vidi `SPECIFIKACIJA.md` 3.8)

```go
const restThresholdMin = 270 // 4.5h
```

Ako je `duration_min` rute veće od 270 minuta, worker upisuje `next_rest_suggestion_min = 270`; inače `NULL`. **Ovo nije prava AETR/EU regulativa** (nema kumulativnog praćenja radnih sati kroz više putovanja, nema lokacije stvarnog odmarališta iz OSM-a) — čisto dokazuje da async tok radi od kraja do kraja. Puna logika je future work.

## Šema

```sql
ALTER TABLE trips ADD COLUMN IF NOT EXISTS next_rest_suggestion_min DOUBLE PRECISION;
```
(`ALTER ... IF NOT EXISTS` umesto ubacivanja u `CREATE TABLE`, da važi i za baze koje već imaju `trips` tabelu bez ove kolone — isti princip idempotentne "migracije" kao ranije.)

`trips.status` sada ima dve vrednosti u praksi: `created` (odmah posle `POST /trips`, pre nego što worker stigne do poruke) → `in_progress` (posle worker obrade).

## Verifikacija (2026-07-21)

Testirane obe grane pravila, oba puta end-to-end (HTTP → DB → RabbitMQ → worker → DB):
- Beograd → Novi Sad (68.5 min): `worker: processed trip 3 (rest_suggestion_min=<nil>)`, status `in_progress`, `next_rest_suggestion_min` ostaje `NULL`.
- Subotica → Vranje (320.3 min, 532.7 km — dijagonala cele Srbije): `worker: processed trip 4 (rest_suggestion_min=0x...)`, status `in_progress`, `next_rest_suggestion_min = 270`.

## Šta namerno NIJE urađeno

- Nema retry/DLQ topologije — jedan topic exchange, dve routing key (`trip.started`, `trip.eta_updated`), dovoljno za demo (`SPECIFIKACIJA.md` 3.6 eksplicitno kaže da ovo ne treba graditi sada).
- `trip.eta_updated` se publikuje ali ga trenutno niko ne konzumira — čeka WebSocket gateway iz nedelje 3, koji će ga gurati klijentu.
- Nema stvarne lokacije odmarališta (OSM `amenity=fuel/parking` node podaci, filtrirani još u [osmium fix-u](../fixes/2026-07-21-osmium-filter-node-tags.md), još se ne čitaju nigde u kodu) — `next_rest_suggestion_min` je samo broj minuta, ne mesto.
- Worker nije poseban kontejner/deployable — goroutine u `main.go`. Ako ikad zatreba nezavisno skaliranje, razdvajanje je mali refaktor (isti `worker.TripWorker` tip, samo drugi `main.go` ulaz).
