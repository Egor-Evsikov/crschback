 FROM golang:1.25.3-alpine AS builder

WORKDIR /crschback

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o main.exe .

EXPOSE 8080

CMD [ "./main.exe" ]