# Implantação — Calango na sua VPS

Status: desenho. Dockerfiles e compose existem como ponto de partida, **não construídos nem testados** (daemon Docker indisponível neste ambiente).

## Modelo

```
VPS da pessoa                          clientes
┌──────────────────────────┐          ┌─────────────┐
│ docker: calango         │◄──TLS────│ desktop     │
│  - núcleo, rotinas       │          │ iOS         │
│  - navegador (chromium)  │          │ Android     │
│  - dados (SQLite em vol) │          │ web         │
└──────────────────────────┘          └─────────────┘
```

O servidor é o único lugar onde o agente roda, guarda dados e executa rotinas. Os clientes são controles remotos: não rodam agente, só mostram e enviam comandos. Por isso o agente segue vivo quando o notebook desliga, e o celular funciona de qualquer lugar.

Também funciona com o servidor no próprio computador (modo local). Mesmo binário, só muda o endereço.

## Pareamento

1. A pessoa sobe o container na VPS.
2. O servidor mostra um código de pareamento de uso único (QR ou texto), válido por poucos minutos.
3. O app lê o código, troca por um token de dispositivo e guarda o token no keychain do sistema.
4. Cada dispositivo pode ser revogado separadamente.

## Acesso seguro (obrigatório)

- **Nunca expor a porta em texto puro.** Tudo passa por TLS, via proxy reverso (Caddy ou Traefik com certificado automático) ou pela rede privada.
- **Alternativa sem porta pública**: rede privada (Tailscale ou WireGuard). Recomendar como padrão para quem não quer expor a VPS.
- Senha de administrador forte no primeiro acesso; sem credenciais padrão.
- Tokens de dispositivo com escopo e validade; revogação imediata.
- Segredos (chaves de modelo, tokens de conector) criptografados em repouso, com chave fora do volume de dados.

## Notificações push (mobile)

Um servidor próprio não fala direto com APNs (iOS) nem FCM (Android) sem credenciais da Apple e do Google. Proposta: um **relay de push** pequeno, hospedado por nós, que recebe um aviso do servidor da pessoa e encaminha para o celular dela. O relay não vê conteúdo: só manda "tem novidade", e o app busca o conteúdo no servidor da pessoa. Isso também é um ponto de monetização (ver [MONETIZATION.md](MONETIZATION.md)).

## Atualizações

- Imagem versionada (`calango/calango:<versão>`). Atualizar é trocar a tag e reiniciar.
- Migrações de banco rodam na subida e são irreversíveis sem backup: o compose documenta o volume a salvar.
- Backup do volume de dados: responsabilidade da pessoa, com comando documentado.

## Hospedagem gerenciada (opção paga)

Para quem não quer administrar VPS, oferecer o mesmo servidor hospedado por nós, com as mesmas APIs. Corresponde ao plano Pro/Team em [MONETIZATION.md](MONETIZATION.md). A versão self-hosted segue disponível, então o produto não depende de nós para existir.

## Arquivos

- `deploy/Dockerfile.node`: build do nó em duas etapas, imagem final mínima, usuário não-root. Site: `Dockerfile` na raiz, nginx não-root.
- `deploy/docker-compose.yml`: exemplo com volume de dados e proxy TLS.

## Pendências

- Pareamento e tokens de dispositivo: ainda não implementados.
- Relay de push: a definir.
- Navegador no container (chromium) exige imagem maior; avaliar uma imagem separada.
