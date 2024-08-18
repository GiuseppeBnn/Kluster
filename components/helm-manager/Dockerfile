FROM golang:latest

COPY /bin/helm-manager /helm-storage/helm-manager

EXPOSE 9000

WORKDIR /helm-storage/
ENTRYPOINT ["/bin/bash", "-c", "./helm-manager"]
