# Command

## curl
curl -X GET http://localhost:8080/user/68
curl -X GET http://localhost:8080/user-batch/68

## pprof
go tool pprof -http=:8082 "http://localhost:6060/debug/pprof/profile?seconds=10"

## vegeta
vegeta attack -duration=30s -rate=1000 -targets=targets.txt | vegeta report
vegeta attack -duration=30s -rate=1000 -targets=targets-batch.txt | vegeta report

## combine
(vegeta attack -duration=30s -rate=1000 -targets=targets.txt | vegeta report >> vegeta_report.txt) & go tool pprof "http://localhost:6060/debug/pprof/profile?seconds=30"
(vegeta attack -duration=30s -rate=1000 -targets=targets-batch.txt | vegeta report >> vegeta_report.txt) & go tool pprof "http://localhost:6060/debug/pprof/profile?seconds=30"

## jobs
jobs
fg %1
ctrl + C