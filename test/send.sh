#!/bin/bash
for i in {1..100000}; do
	for i in {1..30}; do
		../mailpit sendmail --smtp-addr="localhost:1025" < email;
	done
	#sleep 0.2m
done
