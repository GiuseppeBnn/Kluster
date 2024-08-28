FROM node:latest

COPY . /packs-proxy

WORKDIR /packs-proxy

RUN npm install

EXPOSE 3001

ENTRYPOINT ["/bin/bash", "-c", "node packs-proxy.js"]