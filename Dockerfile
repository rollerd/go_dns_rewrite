FROM golang:latest
RUN apt-get update
COPY . /dnsproxy
WORKDIR /dnsproxy
RUN go mod init dnsproxy/v2
RUN go mod tidy
RUN go build -o dnsproxy
EXPOSE 53/udp
ENTRYPOINT ["./dnsproxy"]
