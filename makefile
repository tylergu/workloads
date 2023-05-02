rabbitmq-sender:
	docker build -f rabbitmq/sender/Dockerfile -t tylergu1998/rabbitmq-sender:v1 .
	docker push tylergu1998/rabbitmq-sender:v1

rabbitmq-receiver:
	docker build -f rabbitmq/receiver/Dockerfile -t tylergu1998/rabbitmq-receiver:v1 .
	docker push tylergu1998/rabbitmq-receiver:v1

rabbitmq: rabbitmq-sender rabbitmq-receiver