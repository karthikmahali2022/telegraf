#!/bin/bash

if ! grep "^telegraf-nd:" /etc/group &>/dev/null; then
    groupadd -r telegraf-nd
fi

if ! id telegraf &>/dev/null; then
    useradd -r -M telegraf-nd -s /bin/false -d /etc/telegraf-nd -g telegraf-nd
fi

if [[ -d /etc/opt/telegraf-nd ]]; then
    # Legacy configuration found
    if [[ ! -d /etc/telegraf-nd ]]; then
        # New configuration does not exist, move legacy configuration to new location
        echo -e "Please note, Telegraf-nd's configuration is now located at '/etc/telegraf-nd' (previously '/etc/opt/telegraf-nd')."
        mv -vn /etc/opt/telegraf-nd /etc/telegraf-nd

        if [[ -f /etc/telegraf-nd/telegraf-nd.conf ]]; then
            backup_name="telegraf-nd.conf.$(date +%s).backup"
            echo "A backup of your current configuration can be found at: /etc/telegraf/${backup_name}"
            cp -a "/etc/telegraf-nd/telegraf-nd.conf" "/etc/telegraf-nd/${backup_name}"
        fi
    fi
fi
