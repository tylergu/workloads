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
	minikube start --driver=kvm2 --cpus=4 --memory=8192 --disk-size=20000mb --nodes 4
	kubectl delete storageclass standard
	kubectl apply -f data/kubevirt-hostpath-provisioner.yaml

minikube-stop:
	minikube stop

minikube-delete:
	minikube delete

tidb-writer:
	docker build -f tidb/writer/Dockerfile -t ghcr.io/xlab-uiuc/tidb-writer:v1 .
	docker push ghcr.io/xlab-uiuc/tidb-writer:v1

mariadb-writer:
	docker build -f mariadb/writer/Dockerfile -t ghcr.io/xlab-uiuc/mariadb-writer:v1 .
	docker push ghcr.io/xlab-uiuc/mariadb-writer:v1

mongodb-writer:
	docker build -f mongodb/writer/Dockerfile -t ghcr.io/xlab-uiuc/mongodb-writer:v1 .
	docker push ghcr.io/xlab-uiuc/mongodb-writer:v1

cassandra-writer:
	docker build -f cassandra/writer/Dockerfile -t ghcr.io/xlab-uiuc/cassandra-writer:v1 .
	docker push ghcr.io/xlab-uiuc/cassandra-writer:v1

all-writers: tidb-writer mariadb-writer mongodb-writer cassandra-writer
