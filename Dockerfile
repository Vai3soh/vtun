FROM golang:alpine

WORKDIR /app
COPY . /app
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn
RUN go mod tidy
RUN go build -o ./bin/vtun ./main.go

ENTRYPOINT ["./bin/vtun"]

