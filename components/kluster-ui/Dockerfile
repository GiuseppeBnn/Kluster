FROM node:latest

COPY . /kluster-ui

WORKDIR /kluster-ui
RUN npm install
RUN npm run build
EXPOSE 3000

ENTRYPOINT ["/bin/bash", "-c", "npm run start"]