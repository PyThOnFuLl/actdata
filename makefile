all: bin/actdata

models: .database.db
	sqlboiler sqlite3

.database.db: dbschema.sqlite.sql
	rm $@
	sqlite3 -init dbschema.sqlite.sql $@ .exit

.(PHONY): bin/actdata
bin/%: *.go | models
	go build -o $@

