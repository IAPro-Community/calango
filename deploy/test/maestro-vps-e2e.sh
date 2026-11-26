#!/bin/sh
# Teste de ponta a ponta da atualização automática do Orquestrador Maestro no VPS (issue #79).
#
#   sh deploy/test/maestro-vps-e2e.sh IMAGEM      # ex.: calango/calango:maestro-e2e
#
# Usa o instalador real (deploy/install.sh) num contêiner com o Docker CLI e o socket do host,
# com --dir numa pasta temporária, domínio calango.test e CA interna do Caddy. Precisa de Docker,
# curl, jq e openssl no host, das portas 80 e 443 livres e de acesso ao registry.npmjs.org.
#
# Fase A (registro npm real): pasta de dados nova -> Maestro instalado sozinho; Atualizar agora
# sem versão nova; reinício e reinstalação preservam a instalação; nada fora de /data.
# Fase B (registro npm falso, deploy/test/fake-npm-registry.mjs, atrás do nome registry.npmjs.org
# só na rede do Compose e com CA de teste): uma 1.1.2 boa (a 1.1.1 reempacotada) e uma 1.1.3 com
# verify quebrado. Opt-out, atualização manual, verify, rollback e versão que falha no verify.
#
# Remove tudo no fim (instalador --uninstall --purge). KEEP=1 mantém para inspeção.
set -eu

IMAGE=${1:?uso: maestro-vps-e2e.sh IMAGEM}
CLI_IMAGE=${CLI_IMAGE:-docker:29-dind}
REPO=$(cd "$(dirname "$0")/../.." && pwd)
WORK=$(mktemp -d "${TMPDIR:-/tmp}/calango-maestro-e2e.XXXXXX")
DIR=$WORK/calango
FIX=$WORK/fixtures
DOMAIN=calango.test
NODE=calango-calango-1
PKG=@iapro/orquestrador-maestro-cli
BASE_VERSION=1.1.1
TOKEN=""
STEP=0

say() { printf '%s\n' "$*"; }
ok() {
	STEP=$((STEP + 1))
	printf 'ok %d - %s\n' "$STEP" "$*"
}
fail() {
	printf 'FALHOU: %s\n' "$*" >&2
	docker logs --tail 80 "$NODE" >&2 2>&1 || true
	exit 1
}

# cli PROGRAMA ARGS...: roda no contêiner com o Docker CLI (root, rede do host, socket do host).
# As pastas aparecem no mesmo caminho do host porque o Compose resolve os binds no daemon.
cli() {
	prog=$1
	shift
	docker run --rm --network host -v /var/run/docker.sock:/var/run/docker.sock \
		-v "$WORK:$WORK" -v "$REPO/deploy:$REPO/deploy:ro" --entrypoint "$prog" "$CLI_IMAGE" "$@"
}

installer() {
	cli sh "$REPO/deploy/install.sh" --dir "$DIR" --domain "$DOMAIN" --image "$IMAGE" --no-pull --tls-internal
}

compose() {
	cli docker compose --project-name calango --project-directory "$DIR" -f "$DIR/docker-compose.yml" "$@"
}

compose_fake() {
	compose -f "$FIX/override.yml" "$@"
}

cleanup() {
	status=$?
	if [ "${KEEP:-0}" = 1 ]; then
		say "KEEP=1: instalação mantida em $DIR"
		return
	fi
	if [ -f "$FIX/override.yml" ]; then
		compose_fake down --remove-orphans >/dev/null 2>&1 || true
	fi
	cli sh "$REPO/deploy/install.sh" --dir "$DIR" --uninstall --purge --yes >/dev/null 2>&1 || true
	rm -rf "$WORK"
	exit "$status"
}
trap cleanup EXIT INT TERM

# api MÉTODO CAMINHO [JSON]: chama a API pelo Caddy (TLS), como um cliente faria. Imprime o corpo;
# o status fica em http_code. O token nunca vai para a saída.
api() {
	[ -n "$TOKEN" ] || TOKEN=$(docker exec "$NODE" calango print-token)
	out=$WORK/api.out
	curl -sS -k --max-time 900 -o "$out" -w '%{http_code}' --resolve "$DOMAIN:443:127.0.0.1" \
		-X "$1" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
		${3:+--data "$3"} "https://$DOMAIN$2" >"$WORK/api.code"
	cat "$out"
}

# http_code: status HTTP da última chamada de api (que costuma rodar num subshell).
http_code() { cat "$WORK/api.code"; }

status() { api GET /v1/maestro; }

# wait_for JQ SEGUNDOS DESCRIÇÃO: espera GET /v1/maestro satisfazer o filtro jq.
wait_for() {
	i=0
	while [ "$i" -lt "$2" ]; do
		if status 2>/dev/null | jq -e "$1" >/dev/null 2>&1; then
			return 0
		fi
		i=$((i + 3))
		sleep 3
	done
	say "último status: $(status 2>/dev/null || true)" >&2
	fail "$3"
}

wait_healthy() {
	i=0
	until docker exec "$NODE" calango healthcheck >/dev/null 2>&1; do
		i=$((i + 1))
		[ "$i" -lt 60 ] || fail "o nó não ficou saudável"
		sleep 2
	done
}

restart_node() {
	compose_args=${1:-}
	if [ -n "$compose_args" ]; then
		compose_fake restart calango >/dev/null 2>&1
	else
		compose restart calango >/dev/null 2>&1
	fi
	wait_healthy
}

# verify_cli VERSÃO: roda o verify do CLI dessa versão sobre a home estável, com o ambiente do manager.
verify_cli() {
	docker exec -e ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1 -e ORQUESTRADOR_MAESTRO_TELEMETRY=0 \
		-e HOME=/data/maestro -e XDG_CONFIG_HOME=/data/maestro/config -e CODEX_HOME=/data/maestro/.codex \
		"$NODE" "/data/maestro/versions/$1/node_modules/.bin/orquestrador-maestro" verify --home-path /data/maestro >/dev/null
}

switches() {
	api GET /v1/audit/export | jq -c 'select(.action == "maestro.version_switch") | .facts'
}

isolation_check() {
	# /run/secrets é o ponto de montagem da chave; /tmp/node-compile-cache é o cache de compilação
	# do Node (npm 11 liga sozinho), descartável e sem dado do usuário; /fixtures é a CA da fase B.
	changed=$(docker diff "$NODE" | grep -Ev '^[AC] (/run|/run/secrets(/.*)?|/tmp|/tmp/node-compile-cache(/.*)?|/fixtures(/ca\.pem)?)$' || true)
	[ -z "$changed" ] || fail "o contêiner escreveu fora de /data: $changed"
	mounts=$(docker inspect --format '{{range .Mounts}}{{.Destination}} {{end}}' "$NODE")
	case "$mounts" in
	*codex* | *home*) fail "montagem inesperada: $mounts" ;;
	esac
	docker exec "$NODE" sh -c 'test -z "$(ls -A "$HOME")" && test ! -e /data/.codex' || fail "algo foi escrito em \$HOME ou em /data/.codex"
	docker exec "$NODE" test -d /data/maestro/.codex || fail "home estável do Codex ausente"
}

# ---------------------------------------------------------------- fase A: registro real
say "# Fase A: instalador, registro npm real ($IMAGE)"
mkdir -p "$FIX"
installer >"$WORK/install.log" 2>&1 || {
	cat "$WORK/install.log" >&2
	fail "instalador"
}
grep -q '^Imagem: .* (maestro)$' "$WORK/install.log" || fail "o instalador não reconheceu a variante maestro da imagem"
ok "instalador subiu o nó e o Caddy (pasta de dados nova, variante maestro pelo LABEL)"
[ "$(docker exec "$NODE" id -u)" = 65532 ] || fail "o nó não roda como uid 65532"
ok "nó roda como não root (uid 65532), sem ORQUESTRADOR_ALLOW_ROOT_INSTALL"

wait_for '.installed == true and .version == "'"$BASE_VERSION"'" and .auto_update == true' 600 \
	"o Maestro não se instalou sozinho"
ok "Maestro $BASE_VERSION instalado automaticamente; auto_update ligado por padrão ($(status | jq -c .))"
verify_cli "$BASE_VERSION" || fail "verify da $BASE_VERSION"
ok "verify da $BASE_VERSION passa na home estável /data/maestro"
switches | jq -e 'select(.from == "" and .to == "'"$BASE_VERSION"'" and .auto == true and .result == "success")' >/dev/null ||
	fail "auditoria da instalação automática ausente"
ok "auditoria registra a troca automática \"\" -> $BASE_VERSION"
if docker logs "$NODE" 2>&1 | grep -q 'maestro automatic update failed'; then fail "log de falha na atualização"; fi
ok "sem 'maestro automatic update failed' no log"

body=$(api POST /v1/maestro/update)
[ "$(http_code)" = 200 ] || fail "POST /v1/maestro/update: $(http_code) $body"
printf '%s' "$body" | jq -e '.updated == false and .status.version == "'"$BASE_VERSION"'"' >/dev/null || fail "update sem versão nova trocou a versão: $body"
ok "POST /v1/maestro/update sem versão nova: 200, continua $BASE_VERSION"
isolation_check
ok "nada escrito fora de /data/maestro (docker diff vazio, \$HOME do contêiner vazio, sem ~/.codex montado)"

before=$(switches | wc -l)
restart_node
wait_for '.installed == true and .version == "'"$BASE_VERSION"'"' 60 "reinício perdeu a instalação"
installer >"$WORK/install2.log" 2>&1 || fail "reinstalação"
wait_healthy
wait_for '.installed == true and .version == "'"$BASE_VERSION"'"' 60 "reinstalação perdeu a instalação"
sleep 5
[ "$(switches | wc -l)" = "$before" ] || fail "reinício ou reinstalação reinstalou o Maestro"
ok "reinício do contêiner e reinstalação preservam o Maestro sem reinstalar"

# ---------------------------------------------------------------- fase B: registro falso
say "# Fase B: registro npm falso com 1.1.2 (boa) e 1.1.3 (verify quebrado)"
upstream_ip=$(getent ahostsv4 registry.npmjs.org | awk 'NR == 1 { print $1 }')
[ -n "$upstream_ip" ] || fail "não resolvi registry.npmjs.org"
mkdir -p "$FIX/extra" "$FIX/later"
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj "/CN=calango-e2e-ca" \
	-keyout "$FIX/ca.key" -out "$FIX/ca.pem" 2>/dev/null
openssl req -newkey rsa:2048 -nodes -subj "/CN=registry.npmjs.org" -keyout "$FIX/tls.key" -out "$FIX/tls.csr" 2>/dev/null
printf 'subjectAltName=DNS:registry.npmjs.org\n' >"$FIX/san.ext"
openssl x509 -req -in "$FIX/tls.csr" -CA "$FIX/ca.pem" -CAkey "$FIX/ca.key" -CAcreateserial -days 1 \
	-extfile "$FIX/san.ext" -out "$FIX/tls.pem" 2>/dev/null
chmod 644 "$FIX/ca.pem" "$FIX/tls.pem" "$FIX/tls.key"
cp "$REPO/deploy/test/fake-npm-registry.mjs" "$FIX/"
# 1.1.2 = 1.1.1 com outra versão; 1.1.3 = idem, com o verify-install.sh saindo com erro.
docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp -e npm_config_cache=/tmp/npm -v "$FIX:/w" -w /w \
	--entrypoint sh "$IMAGE" -c '
	set -e
	npm pack --silent "'"$PKG@$BASE_VERSION"'" >/dev/null
	repack() {
		rm -rf package && tar xzf iapro-orquestrador-maestro-cli-'"$BASE_VERSION"'.tgz
		sed -i "s/\"version\": \"'"$BASE_VERSION"'\"/\"version\": \"$1\"/" package/package.json
		grep -q "\"version\": \"$1\"" package/package.json
		[ -z "$3" ] || sed -i "2i echo \"verify quebrado de teste\" >\&2; exit 1" package/scripts/verify-install.sh
		tar czf "$2/orquestrador-maestro-cli-$1.tgz" package
	}
	repack 1.1.2 extra ""
	repack 1.1.3 later broken
	rm -rf package'
cat >"$FIX/override.yml" <<EOF
# Só para o teste: registro npm falso atendendo como registry.npmjs.org na rede do Compose.
services:
  fake-registry:
    image: $IMAGE
    user: "0:0"
    entrypoint: ["node", "/fixtures/fake-npm-registry.mjs"]
    environment:
      UPSTREAM_IP: "$upstream_ip"
      EXTRA_DIR: /fixtures/extra
      TLS_KEY: /fixtures/tls.key
      TLS_CERT: /fixtures/tls.pem
    volumes:
      - $FIX:/fixtures:ro
    networks:
      calango:
        aliases: [registry.npmjs.org]
  calango:
    depends_on:
      fake-registry:
        condition: service_started
    environment:
      NODE_EXTRA_CA_CERTS: /fixtures/ca.pem
    volumes:
      - $FIX/ca.pem:/fixtures/ca.pem:ro
EOF

body=$(api PATCH /v1/maestro/settings '{"auto_update": false}')
printf '%s' "$body" | jq -e '.auto_update == false' >/dev/null || fail "PATCH auto_update=false: $body"
# O npm guarda o packument da fase A por 5 minutos; sem isto a 1.1.2 só apareceria depois.
docker exec "$NODE" rm -rf /data/maestro/npm-cache
compose_fake up -d --wait calango >/dev/null 2>&1 || compose_fake up -d calango >/dev/null
wait_healthy
TOKEN=""
wait_for '.update_available.version == "1.1.2" and .version == "'"$BASE_VERSION"'" and .auto_update == false' 300 \
	"com auto_update desligado, a 1.1.2 deveria só aparecer como disponível"
ok "opt-out: com auto_update=false a 1.1.2 só aparece em update_available; continua $BASE_VERSION"
restart_node fake
sleep 10
status | jq -e '.version == "'"$BASE_VERSION"'" and .auto_update == false' >/dev/null || fail "opt-out não sobreviveu ao reinício"
ok "opt-out persiste após reinício e o agendador não instala"

body=$(api POST /v1/maestro/update)
[ "$(http_code)" = 200 ] || fail "POST /v1/maestro/update: $(http_code) $body"
printf '%s' "$body" | jq -e '.updated == true and .status.version == "1.1.2" and .status.rollback_available == true' >/dev/null || fail "update para 1.1.2: $body"
verify_cli 1.1.2 || fail "verify da 1.1.2"
ok "Atualizar agora: 1.1.2 instalada, verify passa, rollback disponível"

body=$(api POST /v1/maestro/rollback)
[ "$(http_code)" = 200 ] || fail "POST /v1/maestro/rollback: $(http_code) $body"
printf '%s' "$body" | jq -e '.version == "'"$BASE_VERSION"'"' >/dev/null || fail "rollback: $body"
verify_cli "$BASE_VERSION" || fail "verify após rollback"
switches | jq -e 'select(.from == "1.1.2" and .to == "'"$BASE_VERSION"'" and .result == "rollback")' >/dev/null ||
	fail "auditoria do rollback ausente"
ok "Reverter versão: volta para $BASE_VERSION, verify passa, auditoria registra rollback"

api PATCH /v1/maestro/settings '{"auto_update": true}' >/dev/null
restart_node fake
wait_for '.version == "1.1.2"' 300 "com auto_update ligado o agendador deveria instalar a 1.1.2"
ok "auto_update ligado: o agendador instala a versão mais nova (1.1.2) no início, inclusive a revertida"

cp "$FIX/later/orquestrador-maestro-cli-1.1.3.tgz" "$FIX/extra/"
restart_node fake
i=0
until switches | jq -e 'select(.to == "1.1.3" and .result == "verify_failed")' >/dev/null 2>&1; do
	i=$((i + 3))
	[ "$i" -lt 300 ] || fail "a 1.1.3 quebrada deveria falhar no verify"
	sleep 3
done
status | jq -e '.version == "1.1.2"' >/dev/null || fail "a 1.1.3 quebrada substituiu a ativa"
docker logs "$NODE" 2>&1 | grep -q 'maestro automatic update failed' || fail "falha sem registro no log"
verify_cli 1.1.2 || fail "verify da ativa após a falha"
ok "1.1.3 com verify quebrado: falha registrada (auditoria verify_failed, log), 1.1.2 continua ativa"
body=$(api POST /v1/maestro/update)
code=$(http_code)
[ "$code" != 200 ] || fail "update manual para a 1.1.3 quebrada deveria falhar: $body"
status | jq -e '.version == "1.1.2"' >/dev/null || fail "update manual quebrado trocou a versão"
ok "Atualizar agora com a 1.1.3 quebrada: HTTP $code $(printf '%s' "$body" | jq -c .error.code 2>/dev/null) e a 1.1.2 continua ativa"

isolation_check
ok "nada escrito fora de /data/maestro depois de update, rollback e falha"
restart_node fake
wait_for '.installed == true and .version == "1.1.2"' 60 "reinício final perdeu a instalação"
ok "reinício final preserva a instalação ($(status | jq -c '{installed, version, auto_update, rollback_available}'))"
say "auditoria maestro.version_switch:"
switches
say "PASSOU: $STEP verificações"
