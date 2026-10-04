# FeatureTrace dashboard: the log viewer, served by nginx.
#
#   docker build -t featuretrace/dashboard .
#   docker run -p 3009:80 -e QUERY_API_URL=http://query-api:3008 featuretrace/dashboard

FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build

FROM nginx:1.29-alpine
ENV QUERY_API_URL=http://localhost:3008 \
    NGINX_ENTRYPOINT_LOCAL_RESOLVERS=1 \
    NGINX_ENTRYPOINT_QUIET_LOGS=1
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
