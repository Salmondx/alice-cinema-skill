FROM golang:latest
WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./...
RUN GOARCH=amd64 GOOS=linux go build -o app .
EXPOSE 5000
CMD ["./app"]

# zeit now doesnt work with multistaged builds
# FROM scratch  
# WORKDIR /root/
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# COPY --from=builder /go/src/app .
# EXPOSE 5000
# CMD ["./app"] 