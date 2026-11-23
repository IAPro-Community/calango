# Site de apresentação do Calango.
#
#   docker build -t calango-site .
#   docker run --rm -p 8080:8080 calango-site
#   abra http://localhost:8080
#
# Imagem final: nginx não-root, ~20 MB, sem ferramentas de build.

FROM nginxinc/nginx-unprivileged:1.27-alpine

LABEL org.opencontainers.image.title="Calango — site" \
      org.opencontainers.image.description="Página de apresentação do Calango" \
      org.opencontainers.image.source="https://github.com/IAPro-Community/calango" \
      org.opencontainers.image.licenses="AGPL-3.0-only"

COPY deploy/nginx.site.conf /etc/nginx/conf.d/default.conf
COPY site/ /usr/share/nginx/html/

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
