# CADDY_VERSION set by build.sh based on .tool-versions file
ARG CADDY_VERSION=0
FROM public.ecr.aws/docker/library/caddy:${CADDY_VERSION}-builder-alpine AS builder

WORKDIR /usr/src/redir

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o /usr/bin/mirage ./cmd/mirage

ARG CADDY_VERSION=0
FROM public.ecr.aws/docker/library/caddy:${CADDY_VERSION}-alpine

COPY --from=builder /usr/bin/mirage /usr/bin/mirage

#LABEL com.datadoghq.ad.check_names='["openmetrics"]'
#LABEL com.datadoghq.ad.init_configs='[{}]'
#LABEL com.datadoghq.ad.instances='[{"openmetrics_endpoint": "http://%%host%%:81/metrics"}]'
LABEL com.datadoghq.ad.logs='[{"source": "caddy"}]'

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -q --tries=1 --spider http://127.0.0.1/.ping || exit 1

# Upgrade alpine packages (useful for security fixes)
RUN apk upgrade --no-cache

# Copy mirage config
COPY Caddyfile /etc/caddy/Caddyfile

CMD ["mirage", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
