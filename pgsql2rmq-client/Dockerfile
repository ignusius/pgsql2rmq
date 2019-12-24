FROM golang:1.12.7-alpine

WORKDIR /go/src/app
COPY . .

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh
RUN apk add --update tzdata openssl musl-dev ca-certificates && \
rm -rf /var/cache/apk/* && \
cp /usr/share/zoneinfo/Europe/Moscow /etc/localtime && \
echo "Europe/Moscow" > /etc/timezone

RUN go get -d -v ./...
RUN go install -v ./...


CMD ["app"]


