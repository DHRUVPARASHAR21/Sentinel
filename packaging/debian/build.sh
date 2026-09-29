#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"; stage="$(mktemp -d /tmp/sentinel-deb.XXXXXX)"; out="$root/packaging/debian/out"; version="0.1.0"; arch="$(dpkg --print-architecture)"
trap 'rm -rf "$stage"' EXIT
rm -rf "$out"; mkdir -p "$stage/DEBIAN" "$stage/usr/bin" "$stage/etc/sentinel" "$stage/usr/share/doc/sentinel" "$stage/lib/systemd/system" "$stage/var/lib/sentinel" "$stage/var/log/sentinel" "$out"
go build -trimpath -o "$stage/usr/bin/sentineld" "$root/cmd/sentineld"; go build -trimpath -o "$stage/usr/bin/sentinel" "$root/cmd/sentinel"
install -m 0644 "$root/packaging/debian/control" "$stage/DEBIAN/control"; sed -i "s/@VERSION@/$version/;s/@ARCH@/$arch/" "$stage/DEBIAN/control"
install -m 0755 "$root/packaging/debian/postinst" "$stage/DEBIAN/postinst"
install -m 0644 "$root/packaging/systemd/sentinel.service" "$stage/lib/systemd/system/sentinel.service"; install -m 0644 "$root/packaging/debian/sentinel.conf" "$stage/etc/sentinel/sentinel.conf"; install -m 0644 "$root/README.md" "$stage/usr/share/doc/sentinel/README.md"
dpkg-deb --root-owner-group --build "$stage" "$out/sentinel_${version}_${arch}.deb"
