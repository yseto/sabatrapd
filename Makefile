SHELL = /bin/bash
DESTBINDIR = /usr/local/bin
DESTETCDIR = /usr/local/etc

sabatrapd:
	go build

clean:
	rm -f sabatrapd

install: sabatrapd
	@install -m 755 sabatrapd $(DESTBINDIR)
	@install -m 644 sabatrapd.yml.sample $(DESTETCDIR)/sabatrapd.yml
	@install -m 644 systemd/sabatrapd.env $(DESTETCDIR)/sabatrapd.env
	@install -m 644 systemd/sabatrapd.service `systemd-path systemd-system-unit`
	@sed -i -e "s|%DESTETCDIR%|$(DESTETCDIR)|" `systemd-path systemd-system-unit`/sabatrapd.service
	@echo "Installation is finished."
	@echo "Modify $(DESTETCDIR)/sabatrapd.yml, then run 'systemctl enable sabatrapd.service && systemctl start sabatrapd.service'"
