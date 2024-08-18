FROM golang:latest

COPY . /helm-man/
WORKDIR /helm-man/
RUN go build -o helm-manager

EXPOSE 9000
ENTRYPOINT ["/bin/bash", "-c", "./helm-manager"]
