#!/bin/bash
systemctl daemon-reload
systemctl enable bbx-server

echo "=================================================="
echo "BBX (Vigil) has been installed to /opt/bbx"
echo ""
echo "Start server:   systemctl start bbx-server"
echo "Stop server:    systemctl stop bbx-server"
echo "View status:    systemctl status bbx-server"
echo "CLI usage:      bbx-cli --help"
echo "=================================================="
