FROM golang:1.22 AS builder

RUN apt-get update && apt-get upgrade -y && apt-get install sqlite3

WORKDIR /src

COPY go.mod go.mod
COPY go.sum go.sum

# ENV GO111MODULE on

RUN go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build make

FROM gcr.io/distroless/base-nossl-debian12:nonroot AS runner

RUN apt-get update && apt-get upgrade -y && apt-get install sqlite3

COPY --from=builder --chown=nonroot:nonroot /src/bin/actdata /usr/bin/actdata
COPY --from=builder --chown=nonroot:nonroot /src/dbschema.sqlite.sql /usr/lib/actdata/dbschema.sqlite.sql 

EXPOSE 8000 8000

CMD ["actdata"]
