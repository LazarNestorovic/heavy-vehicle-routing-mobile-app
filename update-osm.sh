#!/bin/bash
set -e

PROJECT_DIR=/home/lazar/Documents/heavy-vehicle-routing
OSM_DIR=/home/lazar/Documents/heavy-vehicle-routing/osm-data
RAW_DIR="$OSM_DIR/raw"
FILTERED_DIR="$OSM_DIR/filtered"
LOG="$OSM_DIR/update.log"

echo "[$(date)] Započinjem proces ažuriranja..." >> "$LOG"

# 1. Preuzimanje novog fajla i njegovog MD5 kontrolnog broja u privremeni folder
# Koristimo .tmp ekstenziju da ne pomešamo sa trenutnim fajlovima
echo "[$(date)] Preuzimanje podataka sa Geofabrik-a..." >> "$LOG"
wget -q https://download.geofabrik.de/europe/serbia-latest.osm.pbf -O "$RAW_DIR/serbia-new.osm.pbf"
wget -q https://download.geofabrik.de/europe/serbia-latest.osm.pbf.md5 -O "$RAW_DIR/serbia-new.osm.pbf.md5"

# 2. Verifikacija (Checksum)
# Moramo ući u direktorijum da bi md5sum mogao da pronađe fajl naveden u .md5 fajlu
cd "$RAW_DIR"
EXPECTED_MD5=$(awk '{print $1}' "$RAW_DIR/serbia-new.osm.pbf.md5")
ACTUAL_MD5=$(md5sum "$RAW_DIR/serbia-new.osm.pbf" | awk '{print $1}')
if [ "$EXPECTED_MD5" = "$ACTUAL_MD5" ]; then
    echo "[$(date)] Checksum uspešan. Podaci su ispravni." >> "$LOG"
    
    # Zamenjujemo stari sirovi fajl novim (atomski move)
    mv serbia-new.osm.pbf serbia-latest.osm.pbf
    rm serbia-new.osm.pbf.md5
    
    # 3. Filtriranje (izvršava se samo ako je preuzimanje uspelo)
    echo "[$(date)] Pokrećem filtriranje podataka..." >> "$LOG"
    osmium tags-filter \
      "$RAW_DIR/serbia-latest.osm.pbf" \
      w/highway w/maxheight w/maxweight w/maxwidth \
      w/hgv w/hazmat w/bridge w/tunnel \
      w/surface w/maxspeed \
      r/type=restriction \
      --output "$FILTERED_DIR/serbia-hvt.osm.pbf" \
      --overwrite

    # --- Valhalla Build ---

    echo "[$(date)] Započinjem Valhalla graph build..." >> "$LOG"
    
    # Ulazimo u direktorijum gde je docker-compose.yml
    cd "$PROJECT_DIR"

    # Pokretanje build profila
    # Koristimo --build da osiguramo da je kontejner ažuran i --abort-on-container-exit
    if docker compose --profile build up valhalla-build --abort-on-container-exit; then
        echo "[$(date)] Valhalla graph uspešno izgrađen." >> "$LOG"
        
        # Opciono: Restartuj glavni server da učita nove mape
        echo "[$(date)] Restartujem Valhalla server..." >> "$LOG"
        docker compose restart valhalla
    else
        echo "[$(date)] GREŠKA: Valhalla build nije uspeo!" >> "$LOG"
        exit 1
    fi

    echo "[$(date)] Sve operacije su uspešno završene." >> "$LOG"
else
    echo "[$(date)] GREŠKA: Checksum se ne poklapa! Fajl je oštećen ili nepotpun." >> "$LOG"
    # Brišemo oštećene fajlove da ne troše prostor
    rm -f serbia-new.osm.pbf serbia-new.osm.pbf.md5
    exit 1
fi