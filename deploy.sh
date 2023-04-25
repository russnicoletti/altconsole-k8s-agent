#!/bin/sh
ALTC_ENV_CONFIG_MAP_NAME="altc-env-config-map"
ALTC_ENV_SECRET_NAME="altc-env-secret"

echo "num arguments $#"

if [[ $# -ne 2 ]];
then
  echo "please specify auth0 client-id and secret"
  exit 0
fi

AUTH0_CLIENT_ID=$1
AUTH0_SECRET=$2

kubectl get configmap $ALTC_ENV_CONFIG_MAP_NAME > /dev/null 2>&1
if [ "$?" -eq "0" ]; then
  echo "ConfigMap '${ALTC_ENV_CONFIG_MAP_NAME}' already exists"
else
  echo "ConfigMap '${ALTC_ENV_CONFIG_MAP_NAME}' does not exist. Creating ..."
  kubectl create configmap $ALTC_ENV_CONFIG_MAP_NAME --from-literal=CLUSTER_NAME=`kubectl config view --minify -o jsonpath='{.clusters[].name}'`
fi

kubectl get secret $ALTC_ENV_SECRET_NAME > /dev/null 2>&1
if [ "$?" -eq "0" ]; then
  echo "secret '${ALTC_ENV_SECRET_NAME}' already exists"
else
  echo "secret'${ALTC_ENV_SECRET_NAME}' does not exist. Creating ..."
  kubectl create secret generic $ALTC_ENV_SECRET_NAME --from-literal=AUTH0_CLIENT_ID=$AUTH0_CLIENT_ID --from-literal=AUTH0_SECRET=$AUTH0_SECRET
fi

echo "${ALTC_CONFIG_MAP_NAME} ConfigMap contents:"
echo ""
minikube kubectl -- describe configmap $ALTC_CONFIG_MAP_NAME | grep -A 2 "CLUSTER_NAME"

echo ""
echo "installing agent..."
helm list | grep altc-chart > /dev/null 2>&1
if [ "$?" -eq "0" ]; then
  helm upgrade altc-chart altc-helm/altc-agent
else
  helm install altc-chart altc-helm/altc-agent
fi

