#!/bin/sh
set -e

example_config=/usr/share/doc/gost/examples/gost.yml
target_config=/etc/gost/gost.yml

if [ ! -e "$target_config" ]; then
	mkdir -p /etc/gost
	cp "$example_config" "$target_config"
fi
