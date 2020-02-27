#!/bin/bash
go build .
cp scanner /usr/bin/
mkdir -p /etc/scanner/
cp config.json /etc/scanner/
cp scanner.service /lib/systemd/system/
systemctl daemon-reload
systemctl start scanner.service
systemctl status scanner.service
systemctl enable scanner.service
