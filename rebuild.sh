#!/bin/bash
git pull
go build .
systemctl stop scanner.service
cp scanner /usr/bin/
systemctl start scanner.service
systemctl status scanner.service
