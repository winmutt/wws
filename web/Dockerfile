FROM node:20-alpine AS builder

ARG REACT_APP_API_URL=http://192.168.1.96:8080

WORKDIR /build

COPY web/package.json ./
RUN npm install --legacy-peer-deps

COPY web/ ./
RUN echo "REACT_APP_API_URL=$REACT_APP_API_URL" > .env && npm run build

FROM docker.io/library/nginx:alpine

COPY --from=builder /build/build /usr/share/nginx/html

RUN echo "server { listen 80; location / { root /usr/share/nginx/html; index index.html; try_files \$uri \$uri/ /index.html; } }" > /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
