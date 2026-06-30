#!/bin/bash

NC=/data/nextcloud
LINKS=/data/navidrome/nextcloud

[ -d "$NC" ] || exit 0

mkdir -p "$LINKS"

for files in "$NC"/*/files; do
    [ -d "$files" ] || continue
    user=$(basename "$(dirname "$files")")
    link="$LINKS/$user"
    [ "$(readlink "$link" 2>/dev/null)" = "$files" ] || ln -sfn "$files" "$link"
done

for link in "$LINKS"/*; do
    [ -L "$link" ] || continue
    [ -d "$(readlink "$link")" ] || rm -f "$link"
done

chown navidrome:navidrome "$LINKS" 2>/dev/null || true
chown -h navidrome:navidrome "$LINKS"/* 2>/dev/null || true
