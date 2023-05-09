kubectl create ns chaos-mesh

# Default to /var/run/docker.sock
# minikube
helm install chaos-mesh chaos-mesh/chaos-mesh -n=chaos-mesh --version 2.5.2