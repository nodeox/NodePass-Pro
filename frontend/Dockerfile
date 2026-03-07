FROM node:20-alpine AS builder

WORKDIR /app

ARG APP_VERSION=dev
ENV VITE_APP_VERSION=${APP_VERSION}

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:1.27-alpine

ARG APP_VERSION=dev

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
RUN printf '{"version":"%s"}\n' "${APP_VERSION}" > /usr/share/nginx/html/version.json

LABEL org.opencontainers.image.version="${APP_VERSION}"

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
