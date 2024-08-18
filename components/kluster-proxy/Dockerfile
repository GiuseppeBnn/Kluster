FROM node:latest

COPY . /kluster-proxy

WORKDIR /kluster-proxy

RUN npm install

EXPOSE 3001

ENTRYPOINT ["/bin/bash", "-c", "node kluster-proxy.js"]