#!/bin/bash

tests=("TestBasicAgree4B" "TestFailAgree4B" "TestFailNoAgree4B" "TestConcurrentStarts4B" "TestRejoin4B" "TestBackup4B" "TestCount4B" "TestPersist14B" "TestPersist24B" "TestPersist34B" "TestFigure84B" "TestUnreliableAgree4B" "TestFigure8Unreliable4B" "TestReliableChurn4B" "TestUnreliableChurn4B")
outputs=("agree.txt" "failagree.txt" "failnoagree.txt" "concurrentstart.txt" "rejoin.txt" "backup.txt" "count.txt" "persist1.txt" "persist2.txt" "persist3.txt" "figure8.txt" "unreliableagree.txt" "figure8unreliable.txt" "reliablechurn.txt" "unreliablechurn.txt")

for i in "${!tests[@]}"; do
  echo "Running ${tests[$i]}..."
  for j in {1..10}; do
    go test -race -run "${tests[$i]}"
  done > "${outputs[$i]}"
  echo "${tests[$i]} completed and output stored in ${outputs[$i]}"
done

echo "All tests completed."
