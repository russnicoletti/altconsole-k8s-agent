cd server
docker build -t russnicoletti/nodeserver .
docker tag russnicoletti/nodeserver:latest russnicolettidocker/nodeserver:latest
docker push russnicolettidocker/nodeserver:latest
cd -
