all: bin/actdata

models: .database.db
	rm -rf $@
	sqlboiler --no-hooks sqlite3

.database.db: dbschema.sqlite.sql
	rm $@
	sqlite3 -init dbschema.sqlite.sql $@ .exit

.(PHONY): bin/actdata
bin/%: *.go models apis
	go build -o $@

apis: api.json
	rm -rf $@
	npx -y @openapitools/openapi-generator-cli generate -i api.json -g go -o apis
	rm -rf ./apis/test ./apis/go.*

api.json: api.ts node_modules
	npx @airtasker/spot generate --generator openapi3 --out . -l json -c api.ts

node_modules: package.json
	npm i

actdata.zip: *.go models apis
	git archive -o $@ HEAD
	zip -ru $@ apis models
