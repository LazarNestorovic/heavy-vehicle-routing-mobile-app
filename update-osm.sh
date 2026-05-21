#!/bin/bash
set -e

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
if md5sum --status -c serbia-new.osm.pbf.md5; then
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

    echo "[$(date)] Sve operacije su uspešno završene." >> "$LOG"
else
    echo "[$(date)] GREŠKA: Checksum se ne poklapa! Fajl je oštećen ili nepotpun." >> "$LOG"
    # Brišemo oštećene fajlove da ne troše prostor
    rm -f serbia-new.osm.pbf serbia-new.osm.pbf.md5
    exit 1
fi