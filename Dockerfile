FROM golang:1.8.0-alpine
RUN apk update && apk add openssh
WORKDIR /go/src/docker-tunnel
COPY main.go main.go
COPY vendor vendor
RUN go install
EXPOSE 2375
ENTRYPOINT ["docker-tunnel"]

