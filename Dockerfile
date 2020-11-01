from golang:1.15-alpine

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download -x all

COPY . .
RUN go install -v ./...

CMD ["server"]