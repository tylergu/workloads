rabbitmq-sender:
	docker build -f rabbitmq/sender/Dockerfile -t tylergu1998/rabbitmq-sender:v1 .
	docker push tylergu1998/rabbitmq-sender:v1

rabbitmq-receiver:
	docker build -f rabbitmq/receiver/Dockerfile -t tylergu1998/rabbitmq-receiver:v1 .
	docker push tylergu1998/rabbitmq-receiver:v1

rabbitmq: rabbitmq-sender rabbitmq-receiver

rabbitmq-operator:
	kubectl apply -f https://github.com/rabbitmq/cluster-operator/releases/latest/download/cluster-operator.yml

minikube-start:
	minikube start --driver=kvm2 --cpus=4 --memory=8192 --nodes 3
	kubectl delete storageclass standard
	kubectl apply -f data/kubevirt-hostpath-provisioner.yaml