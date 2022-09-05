#!/bin/bash

function disable_systemd {
    systemctl disable telegraf-nd
    rm -f $1
}

function disable_update_rcd {
    update-rc.d -f telegraf-nd remove
    rm -f /etc/init.d/telegraf-nd
}

function disable_chkconfig {
    chkconfig --del telegraf-nd
    rm -f /etc/init.d/telegraf-nd
}

if [ "$1" == "remove" -o "$1" == "purge" ]; then
	# Remove/purge
	rm -f /etc/default/telegraf-nd

	if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
		disable_systemd /lib/systemd/system/telegraf-nd.service
	else
		# Assuming sysv
		# Run update-rc.d or fallback to chkconfig if not available
		if which update-rc.d &>/dev/null; then
			disable_update_rcd
		else
			disable_chkconfig
		fi
	fi
fi
