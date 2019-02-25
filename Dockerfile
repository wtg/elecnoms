FROM golang:1.11

# Dokku checks http://dokku.viewdocs.io/dokku/deployment/zero-downtime-deploys/
#RUN mkdir /app
#COPY CHECKS /app

RUN mkdir /app
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o elecnoms

EXPOSE 3001
CMD ["/app/elecnoms"]
