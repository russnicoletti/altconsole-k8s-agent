#  Instructions for running the altconsole Kubernetes Agent

## Install local kubernetes (minikube)

To install minikube on MacOS:

`curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-darwin-amd64`  
`sudo install minikube-darwin-amd64 /usr/local/bin/minikube`

For other operating systems, see [minikube start](https://minikube.sigs.k8s.io/docs/start/) page

### Start minikube cluster with CNI
`minikube start --vm-driver=hyperkit --cni calico`

## Deploy local server
`git clone https://github.com/altconsole/k8s-agent`  
`cd node-server`  
`./make.sh`  
`kubectl apply -f server.yaml`  
`cd -`

To view the logs:  
``kubectl logs -f `eval kubectl get pods | grep nodeserver | cut -d " " -f 1,2` ``  
  
`"nodeserver listening on port 3000"`

## Install and configure Helm
On MacOS:  
`brew install helm`

### Specify altconsole k8s-agent helm repo:  
`helm repo add altc-helm https://russnicoletti.github.io/altc-helm`

### Sync with local:  
`helm repo update`

## Build and deploy k8s-agent
`./make.sh`  
`./deploy.sh`

There should now be two pods in the cluster:  
`kubectl get pods` 
   
`NAME                          READY   STATUS    RESTARTS   AGE`  
`altc-agent-84fd87c46b-b445t   1/1     Running   0          4s`  
`nodeserver-78d7b5cc6c-5z9wb   1/1     Running   0          28s`

To view agent logs:  
``kubectl logs -f `eval kubectl get pods | grep altc | cut -d " " -f 1,2` ``

## k8s-agent Behavior
### Authentication
The altconsole k8s-agent authenticates using the `altconsole registration (Test Application)` auth0 application.  
The authentication process consists of:  
- Authenticate with auth0 using the `client credentials` (machine-to-machine) flow, in which auth0 returns an access token in the form of a `JWT`
- Validate the `JWT` and retrieve the `TokenId` from the `https://altconsole.register.com/clientTokenId` custom claim (inserted into the `JWT` by the `tokenIdHandler` Action)  
- Note: currently the `TokenId` is not used. In the future, it will be exchanged for an altconsole `JWT` that will be used to make requests to the altconsole backend

### Collect kubernetes resources
- Wait 20 seconds (currently not configurable) for the kubernetes `informers` to populate their cache
- **TODO** refer to Jira ticket explaining how this arbitrary wait can be avoided
- Start the informers. They will begin sending the kubernetes resources to the k8s-agent informer handler
- The handler will add the resources to a queue
- The k8s-agent controller will continually process the queue, creating a batch of resources.
- The batch of resources is sent to the local server via the server's `/kubernetes/resource` endpoint

## Known Issues
[address continual update of endpoints](https://altconsole.atlassian.net/browse/ALTC-315)  
[avoid fixed wait time at startup](https://altconsole.atlassian.net/browse/ALTC-316)