#!/bin/bash
set -euo pipefail

mkdir -p work

YEARMON="2025-05"
ARCHIVE="dbip-asn-lite-${YEARMON}.csv.gz"
URL="https://download.db-ip.com/free/${ARCHIVE}"

if [[ ! -f "work/$ARCHIVE" ]]; then
  echo "Downloading DB-IP ASN Lite database..."
  curl -fSL "$URL" -o "work/$ARCHIVE"
else
  echo "File already exists: $ARCHIVE"
fi

# update symlink to the downloaded file
rm -f work/dbip-asn-lite.csv.gz
(cd work; ln -s "${ARCHIVE}" dbip-asn-lite.csv.gz)
