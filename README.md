# UpFast

simple tool for uploading and sharing files that is selfhostable.

## How to host

### Regular system

To run on a regular system I recommend to use a virtual environment to install the dependencies and run the project from there.

We will also be setting up an specific user to run the app as safe as possible with a systemd service file for startup

```bash
# Create the user (You can specify a different home-dir)
sudo useradd --shell /bin/bash --system \
    --home-dir "/usr/local/upfast" -m upfast

# Change to upfast user and go to upfast dir
su upfast
cd

# Clone the project directly into upfast-src directory
git clone https://code.cronyakatsuki.xyz/crony/upfast .

# Create the virtual environment and load it for user on default
python3 -m venv venv
# Activate the virtual environment
source venv/bin/activate
# Install all the requirements
pip install -r requirements.txt

# Load environment by default
echo "source /usr/local/upfast/venv/bin/activate" > "/usr/local/upfast/.profile"

# create the upload directory
mkdir upload

# Run the project
uvicorn main:app --host 127.0.0.1 --port 8000 --proxy-headers --forwarded-allow-ips='*'
```

#### Autostart with systemd

To autostart with systemd you will need to create a systemd .service file in path `/etc/systemd/system/upfast.service` and add this content to it.

```systemd
[Unit]
Description=UpFast service
Documentation=https://code.cronyakatsuki.xyz/crony/upfast

[Service]
User=upfast
Group=upfast
WorkingDirectory=/usr/local/upfast/
ExecStart=/usr/local/upfast/venv/bin/uvicorn main:app --host 127.0.0.1 --port 8000 --proxy-headers --forwarded-allow-ips='*'
Restart=on-failure
```

Then just run  `sudo systemctl daemon-reload` and enable the service with `sudo systemctl enable upfast.service`

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

```nginx
server {

    listen 80;
    listen [::]:80;

    server_name upfast.example.com ;

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_redirect off;
        proxy_buffering off;
    }
}
```
