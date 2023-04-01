set -v
GOOS=linux go build -C src -o ../altc-agent-linux .
docker build -t russnicoletti/altc-agent-linux .
docker tag russnicoletti/altc-agent-linux:latest russnicolettidocker/altc-agent-linux:latest
docker push russnicolettidocker/altc-agent-linux:latest

