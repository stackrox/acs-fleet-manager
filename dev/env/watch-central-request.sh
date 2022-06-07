#!/bin/sh
watch "psql -h localhost -d $POSTGRES_DB -U $POSTGRES_USER -c 'select name,status from dinosaur_requests;'"

