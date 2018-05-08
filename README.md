# aws-proxy
- Aws-proxy is a http proxy written in go to transparently sign requests to AWS endpoints.
- Proxy server build on fasthttp
- Get service and region info by parsing http header arg "Authorization", then it's easy to resolve the destination.

# Usage
- go build main.go
- ./main
- Located endpoint_url like this:
```Python
# Python3
import boto3
SERVICE = "dynamodb "  # sqs、Dynamodb、athena, etc.
dydb = boto3.Session().client("SERVICE", endpoint_url="http://localhost:8082")
# then do anything you want
```