#!/bin/bash

for i in `seq 1 $1`; do
	curl -H "Content-Type: application/json" localhost:8000/process --data "{\"id\": $i}" &
done
