IMAGE_NAME=fake-k8s-device-plugin
IMAGE_TAG=0.1.0


build:
	docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .

load:
	kind load docker-image ${IMAGE_NAME}:${IMAGE_TAG}
