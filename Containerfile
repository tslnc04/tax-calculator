# Builder for taxcalc and taxcalcd.
FROM docker.io/golang AS builder

COPY . /src
WORKDIR /src

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /bin/taxcalc cmd/taxcalc/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /bin/taxcalcd cmd/taxcalcd/main.go

# Container for the taxcalc command line tool.
FROM gcr.io/distroless/static AS taxcalc

COPY --from=builder /bin/taxcalc /bin/taxcalc

ENTRYPOINT ["/bin/taxcalc"]
CMD ["-h"]

# Container for the taxcalcd web server.
FROM gcr.io/distroless/static AS taxcalcd

COPY --from=builder /bin/taxcalcd /bin/taxcalcd

EXPOSE 8080

VOLUME /log

ENTRYPOINT ["/bin/taxcalcd"]
CMD ["-log_dir", "/log", "-v", "10"]
