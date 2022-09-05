#!/bin/bash

BIN_DIR=/usr/bin

if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
	deb-systemd-invoke stop telegraf-nd.service
else
	# Assuming sysv
	invoke-rc.d telegraf-nd stop
fi
