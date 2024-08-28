FROM node:latest

COPY . /packs-ui

WORKDIR /packs-ui
RUN npm install
RUN npm run build
EXPOSE 3000

ENTRYPOINT ["/bin/bash", "-c", "npm run start"]