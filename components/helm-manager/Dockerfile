FROM golang:latest

COPY . /helm-manager/
WORKDIR /helm-manager/
RUN go build -o /bin/helm-manager

EXPOSE 9000
WORKDIR /helm-manager/bin
ENTRYPOINT ["/bin/bash", "-c", "./helm-manager"]
