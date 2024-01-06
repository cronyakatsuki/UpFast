# UpFast

The new and improwed version of upfast, now writen in GO!

This one is gonna be a lot easier to deploy since all you need to do is download the upfast binary and run it in a directory.

To change the port or the ip adress to listen for you can use `-p` and `-a` options respectivelly.

There is alse the `-d` option that will set what domain name upfast will return when returning the file path when uploaded.

> example

```bash
./upfast -p 8080 -a 0.0.0.0 -d https://upfast.cronyakatsuki.xyz
```

By default upfast listen's on 127.0.0.1 adress so if you wan't to access it outside your network you will either need to listen to 0.0.0.0 or use a reverse proxy. I recommend the usage of a reverse proxy.

## Note for the users of the python version

You will need to rename the `upload` folder to `files`, since I changed the naming a bit.

## Installation

This installation step's will also harden our installation and limit the size of how much can be uploaded at the same time so your system can't be flooded.

### Creating user
```bash
# Create upfast user with no login shell, no home directory and as a system user
useradd --shell /usr/sbin/nologin --system -M upfast
# Resctrict login to upfast user
usermod -L upfast
```

### Creating virtual filesystem

This is done in order to limit the amount of storage the upfast installation, so people can't storage dos you.


Creating virtual filesystem to limit amount people can upload like chad's, using dd to create an empty file of size you choose.

Change the size output file to whatever you wan't and size to whatever you wan't.
``` bash
# make sure to first change into a directory where you wan't to save the virtual filesystem
dd if=/dev/zero of=20gb bs=1M count=20480
# make that file into a filesystem
mkfs.ext4 20gb
# create directory to mount the filesystem, I recommend /usr/local/upfast because that's where the included systemd service looks for upfast binary
mkdir /usr/local/upfast
# mounting the filesystem
mount -o loop,rw /home/ivek/20gb /usr/local/upfast
```
Now to make it mount on system reboot we need to add this line to fstab.
``` fstab
/home/amir/mydatafile /usr/local/upfast ext4 loop,rw,usrquota,grpquota 0 0
```
### Downloading the binary
```bash
# go to upfast's user's home and curl the upfast binary, make sure to run the next command's as root
cd /usr/local/upfast
curl -O https://code.cronyakatsuki.xyz/crony/UpFast/releases/download/v1.1/upfast
chmod +x upfast
```
```bash
# Own the directory with upfast user and group
chown upfast:upfast -R /usr/local/upfast
chmod 700 /usr/local/upfast
```

## Updating

When new update of upfast come's out all you will need to change to upfast user and curl the new binary in it's home directory overwritting the old one like this.

```bash
su upfast
cd /usr/local/upfast
curl -O https://code.cronyakatsuki.xyz/crony/UpFast/releases/download/v1.1/upfast
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
ExecStart=/usr/local/upfast/upfast -p 8000 -a 127.0.0.1 -d https://upfast.cronyakatsuki.xyz
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Remember to change the port if you need to, the listening adress and the domain name.

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
