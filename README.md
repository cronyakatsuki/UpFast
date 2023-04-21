# UpFast

Simple tool for uploading and sharing files that you can self-host.

## How to host

### Regular system

To run on a regular system I recommend to use a virtual environment to install the dependencies and run the project from there.

We will also be setting up an specific user to run the app as safe as possible with a systemd service file for startup

#### Installing
```bash
./install.sh
```

#### Running
```bash
systemctl start upfast.service
```

#### Start on boot
```bash
systemctl enable upfast.service
```
### Docker

In the repo there is an included dockerfile to generate an image from the latest version of everything, to generate an image just run `docker build . -t upfast` (You need root privileges or to be in the docker group).

To run the docker container check the container id with `docker images` command.

> sample docker run command
```bash
sudo docker run -p 8000:8000 -v ./upload:/usr/src/app/upload CONTAINER_ID
```

The sample command will need an upload directory, you can replace `./upload` with a different path to save uploaded stuff.

### Nginx Proxy setup

This is an example nginx proxy config for http
```bash
cp ./upfast-nginx /etc/nginx/sites-available/upfast
ln -sf /etc/nginx/sites-available/upfast /etc/nginx/sites-enabled/
```

> Load config
```bash
systemctl reload nginx
```

# Contributions

Thanks [TLasT](https://craftmenners.men) on his help with testing and documentation.
