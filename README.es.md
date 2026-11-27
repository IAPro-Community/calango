<p align="center">
  <img src="site/assets/logo-192.png" width="112" alt="Logo de Calango" />
</p>

<h1 align="center">Calango</h1>

<p align="center">
  <strong>Tu asistente personal. Funciona en tu computadora, sigue trabajando en tu VPS y responde desde tu celular.</strong>
</p>

<p align="center">
  Rutinas · aprobaciones · navegador · ADE para programar proyectos · canales con personas y agentes
</p>

<p align="center">
  <a href="LICENSE"><img alt="Licencia: AGPL-3.0" src="https://img.shields.io/badge/licencia-AGPL--3.0-E8A600" /></a>
  <img alt="Estado: en construcción" src="https://img.shields.io/badge/estado-en%20construcci%C3%B3n-2D7A22" />
  <img alt="Go 1.23+" src="https://img.shields.io/badge/go-1.23%2B-00ADD8" />
</p>

<p align="center">
  <a href="README.md">English</a> · <b>Español</b> · <a href="README.pt-BR.md">Português (BR)</a>
</p>

---

> [!NOTE]
> **En construcción.** Estamos creando la V1 de Calango en público. Mira el [TODO de la V1](DEV/TODO_V1.md) y [cómo contribuir](CONTRIBUTING.md).

## Por qué Calango

- **Recuerda, ejecuta y guarda lo que le pides.** Trabaja en la computadora que ya usas.
- **Nada importante ocurre sin tu OK.** Enviar, borrar, publicar, pagar o hacer deploy siempre espera una aprobación.
- **Tus datos se quedan contigo.** Cada nodo guarda su propia base de datos. Los secretos nunca llegan a los clientes.
- **Contrólalo desde cualquier lugar.** Usa la app de escritorio, el celular o ambos. Tu VPS sigue trabajando cuando la computadora está apagada.

## Inicio rápido

### Ver la página del proyecto (Docker)

```bash
docker build -t calango-site .
docker run --rm -p 8080:8080 calango-site
```

Luego abre <http://localhost:8080>.

### Ejecutar el núcleo (Go 1.23+)

```bash
go vet ./...
go test ./...
go run ./cmd/calango          # escucha en 127.0.0.1:7317
curl http://127.0.0.1:7317/healthz
```

### Construir la imagen del nodo (para VPS)

```bash
docker build -f deploy/Dockerfile.node -t calango/calango .
```

Ejemplo de compose con TLS automático: [deploy/docker-compose.yml](deploy/docker-compose.yml).

## Estructura del repositorio

| Carpeta | Contenido |
|---|---|
| `cmd/calango` | punto de entrada del servidor |
| `internal/` | núcleo: agentes, planificador, API |
| `api/` | contrato OpenAPI (fuente de los SDK) |
| `packages/tokens` | design tokens (fuente de CSS y del tema Flutter) |
| `packages/i18n` | catálogo en tres idiomas (TypeScript y ARB para Flutter) |
| `site/` | página del proyecto |
| `deploy/` | Dockerfile del nodo, compose, Caddy |
| `DEV/` | documentación de producto e ingeniería |

## Generar y verificar tokens y traducciones

Los archivos en `packages/tokens/generated` y `packages/i18n/generated` se derivan de las fuentes JSON. Vuelve a generarlos después de editar la fuente y verifica el drift con:

```bash
node scripts/generate-tokens.mjs
node scripts/generate-i18n.mjs
node scripts/generate-tokens.mjs --check
node scripts/generate-i18n.mjs --check
```

## Documentación

- [docs/guia](docs/guia/README.md): guía de uso en portugués de Brasil (primeros pasos, emparejamiento, VPS, Maestro y privacidad).
- [AGENTS.md](AGENTS.md): punto de entrada para quien va a programar (persona o agente).
- [DEV/INDEX.md](DEV/INDEX.md): mapa de toda la documentación.

## Idioma del proyecto

El repositorio está en **inglés**. Issues, pull requests, mensajes de commit, comentarios de código y documentación deben escribirse en inglés. Consulta [CONTRIBUTING.md](CONTRIBUTING.md#language).

## Licencia

Código abierto bajo [GNU AGPL-3.0](LICENSE). Licencia comercial para quien no pueda cumplir la AGPL: [COMMERCIAL.md](COMMERCIAL.md). Las contribuciones siguen el [CLA](CLA.md). Nombre y logo: [TRADEMARKS.md](TRADEMARKS.md).

Calango usa el [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro) (Apache-2.0) como proceso de trabajo de sus agentes. Créditos y avisos en [NOTICE](NOTICE).

## Seguridad

Consulta [SECURITY.md](SECURITY.md). No abras un issue público para una vulnerabilidad.
