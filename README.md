# UpFast

The new and improwed version of upfast, now writen in GO!

This one is gonna be a lot easier to deploy since all you need to do is download the upfast binary and run it in a directory.

To change the port or the ip adress to listen for you can use `-p` and `-a` options respectivelly.

> example

```bash
./upfast -p 8080 -a 0.0.0.0
```

By default upfast listen's on 127.0.0.1 adress so if you wan't to access it outside your network you will either need to listen to 0.0.0.0 or use a reverse proxy. I recommend the usage of a reverse proxy.

## Note for the users of the python version

You will need to rename the `upload` folder to `files`, since I changed the naming a bit.

## Installation

```bash
# Create upfast user and home directory
useradd --shell /bin/sh --system --home-dir /usr/local/upfast upfast
mkdir -p /usr/local/upfast # dodge copying of skeletons
chown upfast:upfast /usr/local/upfast
chmod 700 /usr/local/upfast

# change to upfast user
su upfast
# go to upfast's user's home and curl the upfast binary
cd ~
curl -O https://code.cronyakatsuki.xyz/crony/UpFast/releases/download/v1.0/upfast
chmod +x upfast
```

## Updating

When new update of upfast come's out all you will need to change to upfast user and curl the new binary in it's home directory overwritting the old one like this.

```bash
su upfast
cd ~
curl -O https://code.cronyakatsuki.xyz/crony/UpFast/releases/download/v1.0/upfast
```

## Startup automaticallly on system restart

If you use the old version of upfast you just need to change values in the systemd service located `/etc/systemd/system/upfast.service`, otherwise create a file in that path with this content.

> /etc/systemd/system/upfast.service

```systemd-service
[Unit]
Description=UpFast service
Documentation=https://code.cronyakatsuki.xyz/crony/upfast

[Service]
User=upfast
Group=upfast
WorkingDirectory=/usr/local/upfast/
ExecStart=/usr/local/upfast/upfast -p 8000 -a 127.0.0.1
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Remember to change the port if you need to or the listening adress.

After that run `systemctl daemon-reload` as root or with sudo.

## Nginx proxy setup

To setup nginx as the reverse proxy you will need to add this server block to either the main nginx config file or in a separate file depending on your version of nginx.

```nginx
server {
   listen 80;
   listen [::]:80;

   server_name upfast.example.xyz;

   location / {
        proxy_pass https://127.0.0.1:8000;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```
