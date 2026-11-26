#!/bin/sh
# Instalador do Calango para VPS (Linux com Docker Engine e Docker Compose v2).
#
#   curl -fsSL https://raw.githubusercontent.com/IAPro-Community/calango/vX.Y.Z/deploy/install.sh -o install.sh
#   less install.sh    # leia antes de rodar como root
#   sudo sh install.sh --domain calango.exemplo.com --version vX.Y.Z
#
# O que faz: confere Docker e Compose, cria /opt/calango, gera a chave do cofre (uma vez só,
# arquivo 0400 do uid 65532, nunca impressa), escreve .env, docker-compose.yml e Caddyfile,
# baixa a imagem fixando o digest, sobe o nó atrás do Caddy (TLS automático) e espera o
# /healthz. A variante padrão (maestro) traz Node.js e npm para o Orquestrador Maestro se
# instalar e se atualizar sozinho; --variant minimal usa a imagem sem Node.js. Rodar de novo
# atualiza a configuração e a imagem sem trocar a chave nem apagar dados. Não instala Docker nem baixa scripts de terceiros. Detalhes em DEV/DEPLOYMENT.md.
set -eu

DEFAULT_DIR=/opt/calango
DEFAULT_REPO=ghcr.io/iapro-community/calango
DEFAULT_SUBNET=172.30.117.0/24
# Usuário não root das duas variantes da imagem (nonroot), dono do volume de dados e da chave.
NODE_UID=65532
# maestro: imagem X.Y.Z-maestro, com Node.js LTS e npm (o Maestro instala e atualiza sozinho).
# minimal: imagem X.Y.Z, distroless sem Node.js (o Maestro não instala).
DEFAULT_VARIANT=maestro
PROJECT=calango

say() { printf '%s\n' "$*"; }
warn() { printf 'aviso: %s\n' "$*" >&2; }
die() {
	printf 'erro: %s\n' "$*" >&2
	exit 1
}

usage() {
	cat <<'EOF'
Uso: install.sh --domain DOMINIO --version vX.Y.Z [opções]
     install.sh --uninstall [--purge [--yes]] [--dir DIR]

Instalação (rodar de novo é seguro: mantém chave, dados e token):
  --domain DOMINIO   domínio público que aponta para este VPS (obrigatório na 1ª vez)
  --version VERSAO   versão publicada da imagem (ex.: v0.4.0); sem "latest"
  --variant VARIANTE maestro (padrão: com Node.js e npm, o Maestro instala e atualiza
                     sozinho) ou minimal (sem Node.js, sem Maestro); vale com --version
  --image REF        referência completa da imagem (substitui --version; espelhos e testes)
  --dir DIR          diretório de instalação (padrão: /opt/calango)
  --subnet CIDR      sub-rede interna do Compose (padrão: 172.30.117.0/24)
  --tls-internal     certificado da CA interna do Caddy (só testes/rede local)
  --no-pull          não baixa a imagem (usa a que já está no Docker)

Remoção:
  --uninstall        para e remove os contêineres; preserva dados, chave e configuração
  --purge            com --uninstall: apaga também volumes, chave e DIR (irreversível)
  --yes              confirma --purge sem perguntar

Outros:
  --print-compose    imprime o docker-compose.yml gerado e sai
  --print-caddyfile  imprime o Caddyfile gerado e sai
  -h, --help         esta ajuda
EOF
}

print_compose() {
	cat <<'EOF'
# Calango no VPS: nó + proxy Caddy com TLS automático.
# Gerado por deploy/install.sh (sh deploy/install.sh --print-compose); a CI exige os dois iguais.
# Variáveis no .env ao lado: CALANGO_IMAGE, CALANGO_DOMAIN, CALANGO_SUBNET e CALANGO_TLS.
# A chave do cofre fica em ./secrets/vault.key (base64 de 32 bytes, dono 65532, modo 0400).
name: calango

services:
  # Ajusta o dono do volume de dados para o usuário não root do nó (uid 65532) e termina.
  data-init:
    image: busybox:1.37
    user: "0:0"
    command: ["chown", "65532:65532", "/data"]
    network_mode: none
    volumes:
      - calango-data:/data
    restart: "no"

  calango:
    image: ${CALANGO_IMAGE:?defina CALANGO_IMAGE no .env}
    restart: unless-stopped
    depends_on:
      data-init:
        condition: service_completed_successfully
    environment:
      CALANGO_ADDR: 0.0.0.0:7317
      CALANGO_DATA_DIR: /data
      CALANGO_VAULT_KEY_FILE: /run/secrets/calango_vault_key
      # Só o proxy e o próprio nó estão nesta rede; o nó não publica porta no host.
      CALANGO_TRUSTED_PROXIES: ${CALANGO_SUBNET:-172.30.117.0/24}
    secrets:
      - calango_vault_key
    volumes:
      - calango-data:/data
    networks:
      - calango
    expose:
      - "7317"
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    healthcheck:
      test: ["CMD", "/usr/local/bin/calango", "healthcheck"]
      interval: 10s
      timeout: 5s
      start_period: 30s
      retries: 3

  proxy:
    image: caddy:2.10
    restart: unless-stopped
    depends_on:
      calango:
        condition: service_healthy
    environment:
      CALANGO_DOMAIN: ${CALANGO_DOMAIN:?defina CALANGO_DOMAIN no .env}
      CALANGO_TLS: ${CALANGO_TLS:-acme}
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config
    networks:
      - calango
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
    security_opt:
      - no-new-privileges:true
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "/dev/null", "http://127.0.0.1:2019/config/"]
      interval: 30s
      timeout: 5s
      start_period: 10s
      retries: 3

networks:
  calango:
    ipam:
      config:
        - subnet: ${CALANGO_SUBNET:-172.30.117.0/24}

secrets:
  calango_vault_key:
    file: ./secrets/vault.key

volumes:
  calango-data:
  caddy-data:
  caddy-config:
EOF
}

print_caddyfile() {
	cat <<'EOF'
# Calango atrás do Caddy. Gerado por deploy/install.sh (sh deploy/install.sh --print-caddyfile).
# CALANGO_DOMAIN vem do .env. CALANGO_TLS=acme (padrão) pede o certificado público
# automaticamente (portas 80 e 443 abertas, DNS apontando para o VPS); CALANGO_TLS=internal
# usa a CA interna do Caddy, só para testes ou rede local.
(calango_tls_acme) {
}

(calango_tls_internal) {
	tls internal
}

{$CALANGO_DOMAIN} {
	import calango_tls_{$CALANGO_TLS:acme}
	reverse_proxy calango:7317
}
EOF
}

# env_get FILE KEY: valor de KEY num .env escrito por este script (sem aspas nem espaços).
env_get() {
	[ -f "$1" ] || return 0
	sed -n "s/^$2=//p" "$1" | tail -n 1
}

valid_domain() {
	printf '%s\n' "$1" | grep -Eq '^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$'
}

valid_image() {
	printf '%s\n' "$1" | grep -Eq '^[a-z0-9][a-z0-9._/-]*(:[A-Za-z0-9._-]+)?(@sha256:[a-f0-9]{64})?$'
}

# image_for VERSAO VARIANTE: referência oficial da imagem dessa versão e variante.
image_for() {
	case "$2" in
	maestro) printf '%s:%s-maestro\n' "$DEFAULT_REPO" "${1#v}" ;;
	minimal) printf '%s:%s\n' "$DEFAULT_REPO" "${1#v}" ;;
	esac
}

valid_subnet() {
	printf '%s\n' "$1" | grep -Eq '^[0-9]{1,3}(\.[0-9]{1,3}){3}/[0-9]{1,2}$'
}

compose() {
	docker compose --project-name "$PROJECT" --project-directory "$DIR" -f "$DIR/docker-compose.yml" "$@" </dev/null
}

# write_file DEST MODE: grava a entrada padrão em DEST de forma atômica.
write_file() {
	tmp="$1.tmp.$$"
	cat >"$tmp"
	chmod "$2" "$tmp"
	mv -f "$tmp" "$1"
}

require_docker() {
	command -v docker >/dev/null 2>&1 || die "Docker não encontrado. Instale o Docker Engine pelo guia oficial (https://docs.docker.com/engine/install/) e rode de novo."
	docker info >/dev/null 2>&1 || die "o daemon do Docker não respondeu. Confira se ele está rodando (systemctl status docker)."
	docker compose version >/dev/null 2>&1 || die "Docker Compose v2 (plugin 'docker compose') não encontrado. Instale o pacote docker-compose-plugin."
}

require_root() {
	[ "$(id -u)" -eq 0 ] || die "rode como root (sudo): a chave do cofre precisa pertencer ao uid $NODE_UID."
}

data_volume_exists() {
	docker volume inspect "${PROJECT}_calango-data" >/dev/null 2>&1
}

# ensure_vault_key: cria a chave só se ainda não existir. Nunca imprime o conteúdo.
ensure_vault_key() {
	key="$DIR/secrets/vault.key"
	mkdir -p "$DIR/secrets"
	chmod 700 "$DIR/secrets"
	if [ -s "$key" ]; then
		chown "$NODE_UID:$NODE_UID" "$key"
		chmod 400 "$key"
		say "Chave do cofre existente mantida: $key"
		return 0
	fi
	if data_volume_exists; then
		die "o volume ${PROJECT}_calango-data já existe, mas $key sumiu. Restaure a chave do backup; uma chave nova não abre os segredos gravados."
	fi
	tmp="$key.tmp.$$"
	(
		umask 077
		if [ -r /dev/urandom ] && command -v base64 >/dev/null 2>&1; then
			head -c 32 /dev/urandom | base64 >"$tmp"
		elif command -v openssl >/dev/null 2>&1; then
			openssl rand -base64 32 >"$tmp"
		else
			exit 1
		fi
	) || {
		rm -f "$tmp"
		die "não foi possível gerar a chave: falta base64 (coreutils) ou openssl."
	}
	[ "$(tr -d '\n' <"$tmp" | wc -c)" -eq 44 ] || {
		rm -f "$tmp"
		die "a chave gerada não tem o tamanho esperado."
	}
	chown "$NODE_UID:$NODE_UID" "$tmp"
	chmod 400 "$tmp"
	mv -f "$tmp" "$key"
	say "Chave do cofre criada: $key (guarde uma cópia cifrada fora do VPS; sem ela os segredos não voltam)."
}

# resolve_image REF: baixa REF (salvo --no-pull) e devolve repositório@digest quando houver.
resolve_image() {
	ref="$1"
	if [ "$PULL" = 1 ]; then
		say "Baixando $ref ..." >&2
		docker pull --quiet "$ref" >/dev/null || die "não foi possível baixar $ref. Confira a versão em https://github.com/IAPro-Community/calango/pkgs/container/calango."
	else
		docker image inspect "$ref" >/dev/null 2>&1 || die "a imagem $ref não está no Docker local (use sem --no-pull)."
	fi
	case "$ref" in
	*@sha256:*)
		printf '%s\n' "$ref"
		return 0
		;;
	esac
	# Tira a tag só do último componente (um registro pode ter porta, ex.: host:5000/calango).
	case "${ref##*/}" in
	*:*) repo="${ref%:*}" ;;
	*) repo="$ref" ;;
	esac
	digest=$(docker image inspect --format '{{range .RepoDigests}}{{println .}}{{end}}' "$ref" 2>/dev/null | grep "^$repo@sha256:" | head -n 1 || true)
	if [ -n "$digest" ]; then
		printf '%s\n' "$digest"
	else
		warn "$ref não tem digest de registro (imagem local); usando a tag."
		printf '%s\n' "$ref"
	fi
}

wait_node_healthy() {
	i=0
	while [ "$i" -lt 60 ]; do
		if compose exec -T calango /usr/local/bin/calango healthcheck >/dev/null 2>&1; then
			return 0
		fi
		i=$((i + 1))
		sleep 2
	done
	compose ps >&2 || true
	die "o nó não ficou saudável. Veja: docker compose -f $DIR/docker-compose.yml logs calango"
}

check_proxy() {
	if ! command -v curl >/dev/null 2>&1; then
		warn "curl não encontrado; pulei a conferência de https://$DOMAIN/healthz pelo proxy."
		return 0
	fi
	insecure=""
	[ "$TLS" = internal ] && insecure="--insecure"
	i=0
	while [ "$i" -lt 30 ]; do
		# --resolve fala com o Caddy local sem depender do DNS já ter propagado.
		# shellcheck disable=SC2086
		if curl -fsS --max-time 10 $insecure --resolve "$DOMAIN:443:127.0.0.1" "https://$DOMAIN/healthz" >/dev/null 2>&1; then
			say "Proxy OK: https://$DOMAIN/healthz respondeu."
			return 0
		fi
		i=$((i + 1))
		sleep 3
	done
	warn "https://$DOMAIN/healthz ainda não respondeu pelo proxy. Confira se o DNS de $DOMAIN aponta para este VPS e se as portas 80 e 443 estão abertas; o Caddy tenta de novo sozinho (docker compose -f $DIR/docker-compose.yml logs proxy)."
}

install() {
	require_root
	require_docker
	envfile="$DIR/.env"

	[ -n "$DOMAIN" ] || DOMAIN=$(env_get "$envfile" CALANGO_DOMAIN)
	[ -n "$DOMAIN" ] || die "informe --domain (ex.: --domain calango.exemplo.com)."
	valid_domain "$DOMAIN" || die "domínio inválido: $DOMAIN"

	if [ -n "$VARIANT" ]; then
		[ -n "$VERSION" ] || [ -n "$IMAGE" ] || die "--variant precisa de --version (a imagem muda de tag)."
	else
		VARIANT=$(env_get "$envfile" CALANGO_VARIANT)
		[ -n "$VARIANT" ] || VARIANT=$DEFAULT_VARIANT
	fi
	case "$VARIANT" in
	maestro | minimal) ;;
	*) die "variante inválida: $VARIANT (use maestro ou minimal)" ;;
	esac

	if [ -n "$IMAGE" ]; then
		:
	elif [ -n "$VERSION" ]; then
		printf '%s\n' "$VERSION" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$' || die "versão inválida: $VERSION (use vX.Y.Z)"
		IMAGE=$(image_for "$VERSION" "$VARIANT")
	else
		IMAGE=$(env_get "$envfile" CALANGO_SOURCE_IMAGE)
		[ -n "$IMAGE" ] || die "informe --version vX.Y.Z (ou --image REF)."
	fi
	valid_image "$IMAGE" || die "referência de imagem inválida: $IMAGE"

	[ -n "$SUBNET" ] || SUBNET=$(env_get "$envfile" CALANGO_SUBNET)
	[ -n "$SUBNET" ] || SUBNET=$DEFAULT_SUBNET
	valid_subnet "$SUBNET" || die "sub-rede inválida: $SUBNET"

	[ -n "$TLS" ] || TLS=$(env_get "$envfile" CALANGO_TLS)
	[ -n "$TLS" ] || TLS=acme

	mkdir -p "$DIR"
	chmod 750 "$DIR"
	ensure_vault_key

	pinned=$(resolve_image "$IMAGE")
	# A imagem declara a própria variante (LABEL calango.variant no Dockerfile.node); com --image
	# ela vale mais que o padrão, para o .env e o aviso final dizerem a verdade.
	actual=$(docker image inspect --format '{{index .Config.Labels "calango.variant"}}' "$pinned" 2>/dev/null || true)
	case "$actual" in
	maestro | minimal) VARIANT=$actual ;;
	esac

	write_file "$envfile" 600 <<EOF
# Gerado por deploy/install.sh. Rode o instalador de novo para mudar; sem segredos aqui.
CALANGO_VARIANT=$VARIANT
CALANGO_SOURCE_IMAGE=$IMAGE
CALANGO_IMAGE=$pinned
CALANGO_DOMAIN=$DOMAIN
CALANGO_SUBNET=$SUBNET
CALANGO_TLS=$TLS
EOF
	print_compose | write_file "$DIR/docker-compose.yml" 644
	print_caddyfile | write_file "$DIR/Caddyfile" 644

	say "Subindo o Calango em $DIR ..."
	compose up -d --remove-orphans || die "docker compose up falhou. Se a sub-rede $SUBNET já está em uso, rode de novo com --subnet."
	wait_node_healthy
	say "Nó saudável (/healthz)."
	check_proxy

	if [ "$actual" = maestro ]; then
		maestro_note="O Orquestrador Maestro se instala sozinho em segundo plano (npm) e se atualiza uma vez por dia;
desligue em Configurações > Maestro ou com PATCH /v1/maestro/settings {\"auto_update\": false}."
	elif [ "$actual" = minimal ]; then
		maestro_note="Variante minimal: sem Node.js, o Orquestrador Maestro não instala (use --variant maestro)."
	else
		maestro_note="A imagem não declara a variante (LABEL calango.variant); confira se ela tem Node.js para o Maestro."
	fi

	cat <<EOF

Calango instalado em $DIR para https://$DOMAIN
Imagem: $pinned ($VARIANT)
$maestro_note

Token de acesso local (trate como senha; não cole em logs, issues nem chats):
  sudo docker compose -f $DIR/docker-compose.yml exec calango calango print-token

Faça backup da chave do cofre em local cifrado fora do VPS: $DIR/secrets/vault.key
Atualizar: rode este instalador de novo com --version nova (chave e dados são mantidos).
Remover: sh install.sh --uninstall (preserva dados; --purge apaga tudo).
EOF
}

uninstall() {
	require_root
	require_docker
	if [ ! -f "$DIR/docker-compose.yml" ]; then
		say "Nada instalado em $DIR."
		[ "$PURGE" = 1 ] || return 0
	fi
	if [ "$PURGE" = 1 ]; then
		if [ "$YES" != 1 ]; then
			(: </dev/tty) 2>/dev/null || die "--purge apaga dados e a chave sem volta; confirme com --yes."
			printf 'Isto apaga os volumes do Calango, a chave do cofre e %s, sem volta. Digite APAGAR para confirmar: ' "$DIR" >/dev/tty
			read -r answer </dev/tty || answer=""
			[ "$answer" = APAGAR ] || die "cancelado."
		fi
		if [ -f "$DIR/docker-compose.yml" ]; then
			compose down --volumes --remove-orphans || die "docker compose down falhou."
		fi
		for v in calango-data caddy-data caddy-config; do
			docker volume rm "${PROJECT}_$v" >/dev/null 2>&1 || true
		done
		rm -rf "$DIR"
		say "Calango removido, com dados e chave."
		return 0
	fi
	compose down --remove-orphans || die "docker compose down falhou."
	say "Contêineres removidos. Dados (volumes ${PROJECT}_*), chave e configuração em $DIR foram preservados."
	say "Para reinstalar com os mesmos dados: rode o instalador de novo. Para apagar tudo: --uninstall --purge."
}

main() {
	DIR=$DEFAULT_DIR
	DOMAIN=""
	VERSION=""
	VARIANT=""
	IMAGE=""
	SUBNET=""
	TLS=""
	PULL=1
	ACTION=install
	PURGE=0
	YES=0
	while [ $# -gt 0 ]; do
		case "$1" in
		--domain) [ $# -ge 2 ] || die "$1 precisa de valor"; DOMAIN=$2; shift ;;
		--version) [ $# -ge 2 ] || die "$1 precisa de valor"; VERSION=$2; shift ;;
		--variant) [ $# -ge 2 ] || die "$1 precisa de valor"; VARIANT=$2; shift ;;
		--image) [ $# -ge 2 ] || die "$1 precisa de valor"; IMAGE=$2; shift ;;
		--dir) [ $# -ge 2 ] || die "$1 precisa de valor"; DIR=$2; shift ;;
		--subnet) [ $# -ge 2 ] || die "$1 precisa de valor"; SUBNET=$2; shift ;;
		--tls-internal) TLS=internal ;;
		--no-pull) PULL=0 ;;
		--uninstall) ACTION=uninstall ;;
		--purge) PURGE=1 ;;
		--yes) YES=1 ;;
		--print-compose) ACTION=print-compose ;;
		--print-caddyfile) ACTION=print-caddyfile ;;
		-h | --help) ACTION=help ;;
		*) usage >&2; die "opção desconhecida: $1" ;;
		esac
		shift
	done
	case "$DIR" in
	/*) ;;
	*) die "--dir precisa ser um caminho absoluto" ;;
	esac
	[ "$DIR" != / ] || die "--dir não pode ser /"
	if [ "$PURGE" = 1 ] && [ "$ACTION" != uninstall ]; then
		die "--purge só vale com --uninstall"
	fi

	case "$ACTION" in
	help) usage ;;
	print-compose) print_compose ;;
	print-caddyfile) print_caddyfile ;;
	uninstall) uninstall ;;
	install) install ;;
	esac
}

# Tudo roda dentro de main: um download cortado no meio não executa nada pela metade.
main "$@"
