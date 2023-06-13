FROM golang:alpine3.16
WORKDIR /app
COPY . /app
RUN go get github.com/gorilla/mux
EXPOSE 3000
CMD go run ./main.go