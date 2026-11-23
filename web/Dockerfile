# ---- build ----------------------------------------------------------------
# Vite 7 needs Node >= 20.19 (the old node:18 base could not build this app).
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json ./
RUN npm install --no-audit --no-fund
COPY . .
RUN npm run build

# ---- serve ----------------------------------------------------------------
FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
# The nginx image renders /etc/nginx/templates/*.template with envsubst.
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
ENV GATEWAY_UPSTREAM=gateway.lastmile.svc.cluster.local:80
EXPOSE 80
