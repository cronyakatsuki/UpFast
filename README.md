# UpFast

simple tool for uploading and sharing files that is selfhostable.

## How to host

### Regular system

To run on a regular system I recommend to use a virtual environment to install the dependencies and run the project from there.


```bash
# Create the virtual environment
python -m venv venv

# Activate the virtual environment
source venv/bin/activate

# Install all the requirements
pip install -r requirements.txt

# Run the project
uvicorn main:app --host 127.0.0.1 --port 8000 --proxy-headers --forwarded-allow-ips='*'
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
