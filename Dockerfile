FROM golang:1.22 AS builder

RUN apt-get update && apt-get upgrade -y
RUN apt-get install -y sqlite3 npm openjdk-17-jre
RUN go install github.com/volatiletech/sqlboiler/v4@latest
RUN go install github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-sqlite3@latest

WORKDIR /src

COPY dbschema.sqlite.sql dbschema.sqlite.sql 
RUN mkdir -p /usr/lib/actdata
RUN cp /src/dbschema.sqlite.sql /usr/lib/actdata/dbschema.sqlite.sql 

COPY go.mod go.mod
COPY go.sum go.sum

# ENV GO111MODULE on

RUN go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build make

RUN cp /src/bin/actdata /usr/bin/actdata

EXPOSE 8000 8000

CMD ["actdata"]
