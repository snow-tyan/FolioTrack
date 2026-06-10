#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage:
  sh deploy/scripts/set-domain.sh <new-domain> [old-domain]

Examples:
  sh deploy/scripts/set-domain.sh track.your-domain.com
  sh deploy/scripts/set-domain.sh track.your-domain.com track.example.com

Notes:
  - Default old-domain is track.example.com.
  - Run this from the repository root.
  - The script updates deploy/README.server.md and deploy/nginx/*.conf.
USAGE
}

NEW_DOMAIN="${1:-}"
OLD_DOMAIN="${2:-track.example.com}"

if [ -z "$NEW_DOMAIN" ]; then
  usage
  exit 1
fi

case "$NEW_DOMAIN" in
  *://*|*/*|*:*|*' '*|.*|*.)
    echo "Invalid domain: $NEW_DOMAIN" >&2
    exit 1
    ;;
esac

if [ ! -d deploy ] || [ ! -d deploy/nginx ]; then
  echo "Please run this script from the repository root." >&2
  exit 1
fi

replace_in_file() {
  file="$1"
  if [ -f "$file" ]; then
    tmp_file="${file}.tmp"
    awk -v old="$OLD_DOMAIN" -v new="$NEW_DOMAIN" '
      {
        line = $0
        output = ""
        while ((pos = index(line, old)) > 0) {
          output = output substr(line, 1, pos - 1) new
          line = substr(line, pos + length(old))
        }
        print output line
      }
    ' "$file" > "$tmp_file"
    mv "$tmp_file" "$file"
  fi
}

replace_in_file deploy/README.server.md

for file in deploy/nginx/*.conf; do
  [ -e "$file" ] || continue
  replace_in_file "$file"
done

rename_if_exists() {
  from="$1"
  to="$2"
  if [ -f "$from" ] && [ "$from" != "$to" ]; then
    mv "$from" "$to"
  fi
}

rename_if_exists "deploy/nginx/${OLD_DOMAIN}.conf" "deploy/nginx/${NEW_DOMAIN}.conf"
rename_if_exists "deploy/nginx/${OLD_DOMAIN}.http.conf" "deploy/nginx/${NEW_DOMAIN}.http.conf"

cat <<EOF2
Domain replacement complete.

Old domain: $OLD_DOMAIN
New domain: $NEW_DOMAIN

Updated files:
  deploy/README.server.md
  deploy/nginx/*.conf
EOF2
