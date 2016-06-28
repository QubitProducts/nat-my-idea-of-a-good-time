FROM golang:1.6

ADD . /go/src/github.com/QubitProducts/nat-my-idea-of-a-good-time
WORKDIR /go/src/github.com/QubitProducts/nat-my-idea-of-a-good-time

RUN go build .
