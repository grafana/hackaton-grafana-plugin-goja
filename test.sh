#!/bin/bash

now=$(date +%s000)
ago=$(date -d '1 hour ago' +%s000)
port=$(cat dist/standalone.txt | tr -d ':')

json_data=$(echo '{}' | base64)

request=$(cat <<EOF
{
  "pluginContext": {
    "orgId": 1,
    "pluginId": "test",
    "dataSourceInstanceSettings": {
      "id": 1,
      "name": "test",
      "uid": "test"
    }
  },
  "queries": [{
    "refId": "A",
    "maxDataPoints": 100,
    "intervalMS": 1000,
    "timeRange": {
      "fromEpochMS": $ago,
      "toEpochMS": $now
    },
    "json": "$json_data"
  }]
}
EOF
)

grpcurl -v -proto backend.proto -plaintext -d "$request" localhost:$port pluginv2.Data/QueryData
