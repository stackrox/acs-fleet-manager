#!/bin/sh
watch "psql -h localhost -d $POSTGRES_DB -U $POSTGRES_USER -c 'select * from dinosaur_requests;'"
