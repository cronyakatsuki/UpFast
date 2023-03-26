# UpFast

simple tool to for uploading and sharing files that is selfhostable.

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
uvicorn main:app --host 127.0.0.1 --port 8000 --proxy-headers
```

### Docker

In the repo there is an included dockerfile to generate an image from the latest version of everything, to generate an image just run `docker build . -t upfast` (You need root privileges or to be in the docker group).

To run the docker container check the container id with `docker images` command.

> sample docker run command
```bash
sudo docker run -p 8000:8000 -v ./upload:/usr/src/app/upload CONTAINER_ID
```

The sample command will need a upload directory to where you run it from so you can replace ./upload with a different path to save uploaded stuff.
