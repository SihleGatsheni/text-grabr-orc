FROM golang:latest

RUN mkdir /build
WORKDIR /build

RUN export GO111MODULE=auto
RUN go get github.com/SihleGatsheni/text-grabr-orc/main
RUN go mod init
RUN cd /build && git clone https://github.com/SihleGatsheni/text-grabr-orc.git

RUN cd /build/text-grabr-orc/main && go build

EXPOSE 8080

ENTRYPOINT ["/build/text-grabr-orc/main/main"]