FROM python:3.10-alpine

WORKDIR /usr/src/app

COPY requirements.txt ./

RUN apk add --no-cache --update libmagic \
    && pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
