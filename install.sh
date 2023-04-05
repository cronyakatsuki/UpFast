#!/bin/sh

# quick install (and run) script for upfast using systemd
if [ $EUID -ne 0]
then
	echo "Please run as root"
	exit 1
fi

useradd --shell /bin/sh --system --home-dir /usr/local/upfast upfast
mkdir -p /usr/local/upfast # dodge copying of skeletons

chown upfast:upfast /usr/local/upfast
chmod 700 /usr/local/upfast

su upfast -c'
cd
git clone https://code.cronyakatsuki.xyz/crony/upfast .

python3 -m venv env
. ./env/bin/activate
pip install -r requirements.txt
mkdir upload'

cp ./upfast.service /etc/systemd/system/upfast.service
systemctl daemon reload
