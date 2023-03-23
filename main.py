import shutil, magic, re
from os import listdir, remove
from os.path import abspath, dirname, exists
from fastapi import FastAPI, Request, UploadFile, File, Header
from fastapi.responses import HTMLResponse, PlainTextResponse, RedirectResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates


# file class
class _File:
    def __init__(self, name: str, fileType: str, content=None):
        self.name = name
        self.fileType = fileType
        self.content = content if content is not None else ''


# File list generator with filetype assignemt and content reading for previews
def file_list_generator(path: str, file_list: list[str]):
    files = []
    for file in file_list:
        file_type = magic.from_file(f"{path}{file}", mime=True)
        if re.search("^image/.*", file_type):
            _file = _File(file, "image")
        elif re.search("^video/.*", file_type):
            _file = _File(file, "video")
        elif re.search("^text/.*", file_type):
            with open(f"{path}{file}") as f:
                content = f.read()
            _file = _File(file, "text", content)
        elif file_type == 'application/json':
            with open(f"{path}{file}") as f:
                content = f.read()
            _file = _File(file, "text", content)
        else:
            _file = _File(file, "else")
        files.append(_file)
    return files


# Load jinja2 for templating
templates = Jinja2Templates(directory="templates")


# Create fastapi template
app = FastAPI()


# Mount all uploaded files at the /files path
app.mount("/files", StaticFiles(directory="upload"), name='upload')


# Mount static files like css
app.mount("/static", StaticFiles(directory="static"), name='static')


# show the homepage when just getting the website
@app.get("/", response_class=HTMLResponse)
async def index(request: Request, user_agent: str | None = Header(default=None)):
    if re.search("^curl/.*", str(user_agent)):
        return PlainTextResponse("It fucking works!\n")
    else:
        context = {'request': request, "user_agent": user_agent}
        return templates.TemplateResponse("index.html", context)


# get the file user want's to upload and save it
@app.post("/")
async def upload( request: Request, file: UploadFile = File(...), user_agent: str | None = Header(default=None)):
    file_location = f"upload/{file.filename}"
    if exists(file_location):
        context = f'''File: "{request.url._url}files/{file.filename}" already exists on the server!
Please rename the file or delete the one on the server\n'''
        return PlainTextResponse(context)
    with open(file_location, "wb+") as file_object:
        shutil.copyfileobj(file.file, file_object)
    if re.search("^curl/.*", str(user_agent)):
        return PlainTextResponse(f"{request.url._url}files/{file.filename}\n")
    else:
        return {"filename": file.filename, "path": f"{request.url._url}files/{file.filename}"}


# show the files page with the list of all files or if run with curl list all files in url format
@app.get("/files")
async def files( request: Request, user_agent: str | None = Header(default=None)):
    path = f"{dirname(abspath(__file__))}/upload/"
    file_list = listdir(path)
    files = file_list_generator(path, file_list)
    context = { 'request': request, "files": files}
    if re.search("^curl/.*", str(user_agent)):
        context = ""
        for file in files:
            context += f"{request.url._url}/{file.name}\n"
        return PlainTextResponse(f"{context}")
    else:
        return templates.TemplateResponse("files.html", context)


# delete specific file when getting this request
@app.get("/delete/{file}")
async def delete(request: Request, file: str, user_agent: str | None = Header(default=None)):
    file_path = f"{dirname(abspath(__file__))}/upload/{file}"
    if exists(file_path):
        remove(file_path)
        if re.search("^curl/.*", str(user_agent)):
            return PlainTextResponse(f"file {file} doesn't exist on the server\n")
        else:
            return RedirectResponse(request.url_for('files'))
    if re.search("^curl/.*", str(user_agent)):
        return PlainTextResponse(f"file {file} doesn't exist on the server\n")
    else:
        return RedirectResponse(request.url_for('files'))
