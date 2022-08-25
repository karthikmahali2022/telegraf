#!/bin/bash

BIN_DIR=/opt/bin
LOG_DIR=/var/log/telegraf-nd
SCRIPT_DIR=/opt/lib/telegraf-nd/scripts
LOGROTATE_DIR=/etc/logrotate.d

function install_init {
    cp -f $SCRIPT_DIR/init.sh /etc/init.d/telegraf-nd
    chmod +x /etc/init.d/telegraf-nd
}

function install_systemd {
    cp -f $SCRIPT_DIR/telegraf-nd.service $1
    systemctl enable telegraf-nd || true
    systemctl daemon-reload || true
}

function install_update_rcd {
    update-rc.d telegraf-nd defaults
}

function install_chkconfig {
    chkconfig --add telegraf-nd
}

# Remove legacy symlink, if it exists
if [[ -L /etc/init.d/telegraf-nd ]]; then
    rm -f /etc/init.d/telegraf-nd
fi
# Remove legacy symlink, if it exists
if [[ -L /etc/systemd/system/telegraf-nd.service ]]; then
    rm -f /etc/systemd/system/telegraf-nd.service
fi

# Add defaults file, if it doesn't exist
if [[ ! -f /etc/default/telegraf-nd ]]; then
    touch /etc/default/telegraf-nd
fi

# Add .d configuration directory
if [[ ! -d /etc/telegraf-nd/telegraf-nd.d ]]; then
    mkdir -p /etc/telegraf-nd/telegraf-nd.d
fi

# If 'telegraf-nd.conf' is not present use package's sample (fresh install)
if [[ ! -f /etc/telegraf-nd/telegraf-nd.conf ]] && [[ -f /etc/telegraf-nd/telegraf-nd.conf.sample ]]; then
   cp /etc/telegraf-nd/telegraf-nd.conf.sample /etc/telegraf-nd/telegraf-nd.conf
fi

test -d $LOG_DIR || mkdir -p $LOG_DIR
chown -R -L telegraf-nd:telegraf-nd $LOG_DIR
chmod 755 $LOG_DIR

if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
	install_systemd /lib/systemd/system/telegraf-nd.service
	deb-systemd-invoke restart telegraf-nd.service || echo "WARNING: systemd not running."
else
	# Assuming SysVinit
	install_init
	# Run update-rc.d or fallback to chkconfig if not available
	if which update-rc.d &>/dev/null; then
		install_update_rcd
	else
		install_chkconfig
	fi
	invoke-rc.d telegraf-nd restart
fi
