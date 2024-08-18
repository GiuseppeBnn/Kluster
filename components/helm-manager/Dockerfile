FROM golang:latest

COPY ./helm-manager /helm-storage/helm-manager/helm-manager

EXPOSE 9000

WORKDIR /helm-storage/helm-manager
ENTRYPOINT ["/bin/bash", "-c", "./helm-manager"]
