#!/usr/bin/env bash

docker run --rm \
	-v $PWD/api.json:/app/swagger.json \
	-p 8080:8080 \
	swaggerapi/swagger-ui
