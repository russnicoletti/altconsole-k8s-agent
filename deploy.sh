#!/bin/sh
ALTC_CONFIG_MAP_NAME="altc-agent"

kubectl get configmap $ALTC_CONFIG_MAP_NAME > /dev/null 2>&1
if [ "$?" -eq "0" ]; then
  echo "ConfigMap '${ALTC_CONFIG_MAP_NAME}' already exists"
else
  echo "ConfigMap '${ALTC_CONFIG_MAP_NAME}' does not exist. Creating ..."
  kubectl create configmap $ALTC_CONFIG_MAP_NAME --from-literal=CLUSTER_NAME=`kubectl config view --minify -o jsonpath='{.clusters[].name}'`
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
