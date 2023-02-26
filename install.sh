#!/bin/bash
PATH=/usr/bin:/bin
DESTBINDIR=${DESTBINDIR:-/usr/local/bin}
DESTETCDIR=${DESTETCDIR:-/usr/local/etc}

if [ ! -f sabatrapd.yml ]; then
  echo "Please create sabatrapd.yml first."
  exit 1
fi

install -m 755 sabatrapd $DESTBINDIR || exit 1
install -m 644 sabatrapd.yml $DESTETCDIR/sabatrapd.yml || exit 1
install -m 644 systemd/sabatrapd.env $DESTETCDIR/sabatrapd.env || exit 1
install -m 644 systemd/sabatrapd.service `systemd-path systemd-system-unit` || exit 1
sed -i -e "s|%DESTBINDIR%|$DESTBINDIR|" -e "s|%DESTETCDIR%|$DESTETCDIR|" `systemd-path systemd-system-unit`/sabatrapd.service || exit 1
systemctl enable sabatrapd.service || exit 1
systemctl start sabatrapd.service || exit 1
echo "sabatrapd installation is finished."
echo "$DESTETCDIR/sabatrapd.yml will be used."
echo "To check sabatrapd's status, type 'journalctl -u sabatrapd'"
