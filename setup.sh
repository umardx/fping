#!/bin/bash
# Print work dir
pwd="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Go install
go build -o infping infping.go

# Create infping.service

cat <<'EOF' > infping.service
[Unit]
Description=infPing
Requires=network-online.target
After=network-online.target consul.service

[Service]
Type=idle
User=root
Group=root
WorkingDirectory=$pwd
PIDFile=/var/run/infping.pid
ExecStart=$pwd/infping
ExecReload=/bin/kill -HUP $MAINPID
KillSignal=SIGINT

[Install]
WantedBy=multi-user.target
EOF

sed -i "s|WorkingDirectory.*|WorkingDirectory=$pwd|g" $pwd/infping.service
sed -i "s|ExecStart.*|ExecStart=$pwd/infping|g" $pwd/infping.service

# Create systemd infping.service
sudo mv $pwd/infping.service /etc/systemd/system/infping.service
sudo systemctl daemon-reload
sudo systemctl enable infping.service
sudo systemctl start infping.service
