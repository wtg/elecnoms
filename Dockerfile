FROM golang:1.10

# Dokku checks http://dokku.viewdocs.io/dokku/deployment/zero-downtime-deploys/
#RUN mkdir /app
#COPY CHECKS /app

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 && chmod +x /usr/local/bin/dep
RUN mkdir -p /go/src/github.com/wtg/elecnoms
WORKDIR /go/src/github.com/wtg/elecnoms
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure -vendor-only

ADD . /go/src/github.com/wtg/elecnoms
RUN go install github.com/wtg/elecnoms

EXPOSE 3001
CMD ["/go/bin/elecnoms"]
