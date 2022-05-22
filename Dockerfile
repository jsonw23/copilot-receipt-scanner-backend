FROM golang:1.18 AS go-builder

WORKDIR /usr/src/app

# pre-fetch the dependencies for the go app in a separate layer
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# copy in the go codebase and build
COPY . .
RUN go build -v -o /usr/local/bin/app ./


FROM golang:1.18 AS prod

ENV STAGE=prod

# copy the compiled binary from go-builder
COPY --from=go-builder /usr/local/bin/app /usr/local/bin/app

CMD ["app"]