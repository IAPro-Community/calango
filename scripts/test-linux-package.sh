#!/usr/bin/env bash
# Valida o .deb do desktop em contêineres Docker limpos (issue #40):
#   1. instala uma versão anterior e sobe o daemon instalado (cria dados);
#   2. atualiza para o .deb atual por cima e confere versão, arquivos, licenças e dependências;
#   3. sobe o daemon de novo sobre os mesmos dados, confere /healthz, print-token e que o log
#      não contém o token;
#   4. abre o app instalado em Xvfb e confere que ele sobe o daemon de resources/bin e grava
#      o log em ~/.config/Calango Desktop/logs/daemon.log sem o token;
#   5. remove (purge) e confere que não sobrou nada do pacote.
# O pacote é instalado sem Recommends, para que as dependências declaradas bastem.
# A chave do cofre é aleatória e descartável; o keyring do sistema fica fora deste teste.
#
# Uso: bash scripts/test-linux-package.sh [arquivo.deb] [deb-anterior]
#   IMAGES="debian:12 ubuntu:24.04" muda as imagens testadas.
#   Sem deb-anterior, o script reempacota o atual com a versão "<versão>~anterior" para testar a
#   atualização pelo dpkg (mesmo conteúdo, versão menor).
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
release_dir="$repo_root/apps/desktop/release"
deb="${1:-}"
if [[ -z "$deb" ]]; then
  deb="$(ls -t "$release_dir"/Calango-*-linux-amd64.deb 2>/dev/null | head -n 1 || true)"
fi
if [[ -z "$deb" || ! -f "$deb" ]]; then
  echo "pacote .deb não encontrado; rode antes: npm run desktop:package:linux" >&2
  exit 2
fi
deb="$(cd "$(dirname "$deb")" && pwd)/$(basename "$deb")"
previous="${2:-}"
if [[ -n "$previous" ]]; then
  previous="$(cd "$(dirname "$previous")" && pwd)/$(basename "$previous")"
fi
images="${IMAGES:-debian:12 ubuntu:24.04}"

echo "pacote: $deb"
sha256sum "$deb"

for image in $images; do
  echo "::group::$image"
  docker_args=(--rm -i
    -v "$deb:/pkg/current.deb:ro"
    -v "$repo_root/LICENSE:/src/LICENSE:ro"
    -v "$repo_root/NOTICE:/src/NOTICE:ro"
    -v "$repo_root/third_party:/src/third_party:ro")
  if [[ -n "$previous" ]]; then
    docker_args+=(-v "$previous:/pkg/previous.deb:ro")
  fi
  docker run "${docker_args[@]}" "$image" bash -s <<'IN_CONTAINER'
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
fail() { echo "FALHOU: $*" >&2; exit 1; }

apt-get update -qq
apt-get install -y -qq --no-install-recommends curl ca-certificates >/dev/null

version="$(dpkg-deb -f /pkg/current.deb Version)"
[[ "$(dpkg-deb -f /pkg/current.deb Package)" == calango-desktop ]] || fail "nome do pacote"
if [[ ! -f /pkg/previous.deb ]]; then
  work="$(mktemp -d)"
  dpkg-deb -R /pkg/current.deb "$work/pkg"
  sed -i "s/^Version: .*/Version: ${version}~anterior/" "$work/pkg/DEBIAN/control"
  dpkg-deb -b "$work/pkg" /tmp/previous.deb >/dev/null
  rm -rf "$work"
  previous=/tmp/previous.deb
else
  previous=/pkg/previous.deb
fi
previous_version="$(dpkg-deb -f "$previous" Version)"
dpkg --compare-versions "$previous_version" lt "$version" || fail "versão anterior ($previous_version) não é menor que $version"

daemon=/opt/Calango/resources/bin/calango
useradd -m tester
data_dir="$(su tester -c 'mktemp -d')"
key="$(head -c 32 /dev/urandom | base64)"
run_daemon() {
  local log="$1"
  su tester -c "CALANGO_VAULT_KEY='$key' CALANGO_DATA_DIR='$data_dir' CALANGO_ADDR=127.0.0.1:7317 $daemon >'$log' 2>&1 & echo \$!" >/tmp/daemon.pid
  for _ in $(seq 1 30); do
    if curl -fsS http://127.0.0.1:7317/healthz >/tmp/healthz 2>/dev/null; then return 0; fi
    sleep 1
  done
  cat "$log" >&2
  fail "daemon instalado não respondeu /healthz"
}
stop_daemon() {
  kill "$(cat /tmp/daemon.pid)"
  for _ in $(seq 1 20); do kill -0 "$(cat /tmp/daemon.pid)" 2>/dev/null || return 0; sleep 0.5; done
  fail "daemon não encerrou com SIGTERM"
}

echo "== instalar $previous_version"
apt-get install -y -qq --no-install-recommends "$previous" >/dev/null
[[ "$(dpkg-query -W -f '${Version}' calango-desktop)" == "$previous_version" ]] || fail "versão instalada"
run_daemon /tmp/daemon-previous.log
stop_daemon
[[ -f "$data_dir/calango.db" ]] || fail "daemon anterior não criou calango.db"

echo "== atualizar para $version"
apt-get install -y -qq --no-install-recommends /pkg/current.deb >/dev/null
[[ "$(dpkg-query -W -f '${Version}' calango-desktop)" == "$version" ]] || fail "versão após atualização"

echo "== arquivos"
for f in /opt/Calango/calango-desktop /opt/Calango/resources/app.asar "$daemon" \
  /usr/share/applications/calango-desktop.desktop /usr/share/icons/hicolor/512x512/apps/calango-desktop.png; do
  [[ -f "$f" ]] || fail "faltando: $f"
done
[[ -x "$daemon" ]] || fail "daemon sem permissão de execução"
[[ "$(readlink -f /usr/bin/calango-desktop)" == /opt/Calango/calango-desktop ]] || fail "/usr/bin/calango-desktop"
grep -q '^Exec=/opt/Calango/calango-desktop' /usr/share/applications/calango-desktop.desktop || fail "Exec do .desktop"
if find /opt/Calango -perm -o+w ! -type l | grep -q .; then fail "arquivo gravável por qualquer usuário em /opt/Calango"; fi

echo "== licenças (cópia integral)"
( cd /src && find LICENSE NOTICE third_party -type f ) | while read -r f; do
  cmp -s "/src/$f" "/opt/Calango/resources/legal/$f" || fail "legal/$f ausente ou diferente"
done
for f in third_party/iapro-sdk/LICENSE third_party/iapro-sdk/NOTICE; do
  [[ -s "/opt/Calango/resources/legal/$f" ]] || fail "legal/$f"
done

echo "== dependências das bibliotecas"
missing="$(ldd /opt/Calango/calango-desktop | grep 'not found' || true)"
[[ -z "$missing" ]] || fail "bibliotecas ausentes: $missing"

echo "== daemon instalado sobre os dados da versão anterior"
run_daemon /tmp/daemon.log
grep -q '"status":"ok"' /tmp/healthz || fail "healthz: $(cat /tmp/healthz)"
token="$(su tester -c "CALANGO_VAULT_KEY='$key' CALANGO_DATA_DIR='$data_dir' $daemon print-token")"
[[ ${#token} -ge 20 ]] || fail "print-token vazio ou curto"
stop_daemon
if grep -qF "$token" /tmp/daemon.log /tmp/daemon-previous.log; then fail "log do daemon contém o token"; fi
echo "healthz: $(cat /tmp/healthz); token com ${#token} caracteres (não exibido)"

echo "== app instalado (Electron em Xvfb) sobe o daemon embutido"
apt-get install -y -qq --no-install-recommends xvfb xauth procps >/dev/null
app_key="$(head -c 32 /dev/urandom | base64)"
su tester -c "CALANGO_VAULT_KEY='$app_key' CALANGO_ADDR=127.0.0.1:7318 xvfb-run -a calango-desktop --no-sandbox >/tmp/app.log 2>&1 &"
app_ok=""
for _ in $(seq 1 60); do
  if curl -fsS http://127.0.0.1:7318/healthz >/dev/null 2>&1; then app_ok=1; break; fi
  sleep 1
done
[[ -n "$app_ok" ]] || { tail -n 30 /tmp/app.log >&2; fail "app instalado não subiu o daemon"; }
pgrep -u tester -f "^$daemon\$" >/dev/null || fail "daemon filho não roda de $daemon"
app_log="/home/tester/.config/Calango Desktop/logs/daemon.log"
[[ -s "$app_log" ]] || fail "app não gravou $app_log"
app_token="$(su tester -c "CALANGO_VAULT_KEY='$app_key' $daemon print-token")"
if grep -qF "$app_token" "$app_log" /tmp/app.log; then fail "log do app contém o token"; fi
pkill -u tester -f calango-desktop || true
for _ in $(seq 1 20); do pgrep -u tester -f "^$daemon\$" >/dev/null || break; sleep 0.5; done
pkill -u tester -f "^$daemon\$" || true
echo "app: daemon de $daemon respondeu /healthz; log em $app_log sem o token"

echo "== remover"
apt-get purge -y -qq calango-desktop >/dev/null
dpkg-query -W calango-desktop >/dev/null 2>&1 && fail "pacote ainda registrado"
for p in /opt/Calango /usr/bin/calango-desktop /usr/share/applications/calango-desktop.desktop \
  /usr/share/icons/hicolor/512x512/apps/calango-desktop.png /etc/alternatives/calango-desktop; do
  [[ -e "$p" || -L "$p" ]] && fail "sobrou após remoção: $p"
done
update-alternatives --query calango-desktop >/dev/null 2>&1 && fail "alternativa calango-desktop ainda registrada"
[[ -f "$data_dir/calango.db" ]] || fail "remoção apagou os dados do usuário"
echo "OK: $(. /etc/os-release && echo "$PRETTY_NAME"): instalar $previous_version, atualizar para $version, daemon, licenças e remoção"
IN_CONTAINER
  echo "::endgroup::"
done
