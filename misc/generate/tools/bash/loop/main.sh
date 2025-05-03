#!/bin/bash
# Script with loop
ITERATIONS={{.iterations}}
echo "Starting loop for $ITERATIONS iterations"
for i in $(seq 1 $ITERATIONS); do
  echo "Iteration $i of $ITERATIONS"
done
echo "Loop completed"