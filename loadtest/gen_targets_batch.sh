#!/bin/bash

# Number of requests to simulate
NUM_REQUESTS=30000  # 1000 RPS for 30 seconds
# Create/overwrite the targets.txt file
echo "" > targets.txt

for ((i=1; i<=NUM_REQUESTS; i++))
do
    # Generate a random ID between 1 and 100
    ID=$i
    echo "GET http://localhost:8080/user-batch/$ID" >> targets-batch.txt
done
