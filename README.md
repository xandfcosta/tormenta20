# tormenta20

pnpm workspace monorepo.

```
engine-go/ Go: API (:3001), motor de regras, e o mesmo motor compilado pra WASM
frontend/  Vite + SolidJS + TanStack Router/Query + Kobalte (Tailwind v4, CSS variables)
e2e/       Playwright, fora do frontend de propósito
```

O frontend era React até o cutover (ALE-76), e o backend NestJS foi removido — o
Go é o backend, e migra o próprio banco (goose embutido) ao subir. O histórico
das duas migrações vive no `git log` e nas issues do Linear.

## Setup

```bash
pnpm install
```

Não há passo de migração: `db.Open` aplica as migrações embutidas na partida.

## Dev

```bash
pnpm dev                 # API Go (:3001) + Vite (:5173) juntos
pnpm dev:frontend        # só o Vite (/api e /socket.io proxiados pra :3001)
```

O `predev` do frontend compila `engine-go` → WASM antes de subir o Vite, porque
o app deriva a ficha pelo MESMO motor do servidor. Precisa do toolchain Go.

A configuração vem de `engine-go/.env.development`, que é **versionado**: nada
ali é segredo, e o servidor recusa subir em produção com aquele `JWT_SECRET`.
Nenhum passo de setup — funciona no clone. O `mestre@t20.local` do seed já é
admin ali, então o menu do Hub tem **Convidar jogador** desde o primeiro boot.

## Testes

```bash
pnpm test                # unit: frontend (NÃO inclui o e2e)
pnpm test:e2e            # Playwright — exige os dois servers de pé (:5173 e :3001)
cd engine-go && go test ./...
```

O e2e ficou FORA do `pnpm test` de propósito (ALE-93): `pnpm -r` roda os pacotes
em paralelo, e disputar CPU com os vitest do front derruba o Playwright por
timeout — e o job `ci`, que roda `pnpm test`, não instala browser nem sobe a API.
No CI o e2e tem job próprio.

## Produção (a mesa na LAN)

Uma vez, no primeiro uso:

```bash
cp engine-go/.env.production.example engine-go/.env.production
openssl rand -hex 32     # cole no JWT_SECRET do arquivo copiado
```

Depois, sempre:

```bash
pnpm build               # WASM + SPA + o binário do servidor
pnpm start               # build + sobe em produção
```

O log termina com o endereço que os jogadores abrem no celular ou no laptop
deles — o servidor escuta em todas as interfaces, então basta estar na mesma
rede:

```
t20 production server listening on :3001 (db=./data/t20-prod.db)
  players can open http://192.168.15.12:3001
```

### Quem entra na mesa

O registro **não é aberto**: quem administra gera um link de uso único (menu do
Hub → **Convidar jogador**) e manda para o jogador, que abre e escolhe a própria
senha. Sem isso, qualquer um que alcance `http://<ip>:3001` criaria conta —
o preço de servir na LAN (ALE-120).

Quem administra vem do `ADMIN_EMAILS`, e **só de lá**: não existe coluna de papel
no banco nem tela para promover alguém, então virar admin exige editar o arquivo
na máquina que hospeda a mesa. Esses e-mails são também a única exceção ao
convite — é assim que você cria a sua própria conta num banco vazio.

O admin abre e edita **qualquer mesa** (as dos outros aparecem nas Crônicas
marcadas com o dono) e tem a cena de **Administração** no menu do Hub: quem está
na mesa e o que cada um tem, os convites em aberto, e o painel de servidor com
backup. Duas ações sobre uma conta: gerar um **link de redefinição de senha** —
você nunca digita nem vê a senha de ninguém — e **apagar**, que leva as fichas
junto e passa as mesas dela para você.

**A stack de produção é um processo só.** Com `STATIC_DIR` apontando para
`frontend/dist`, o `cmd/api` serve o SPA (com fallback pras rotas de cliente), os
assets, `/api/*` e o `/socket.io/` na mesma porta. Não há nginx, não há
docker-compose e não há segundo runtime pra manter — foi por isso que os dois
saíram quando o Nest saiu.

### Backup do banco

```bash
pnpm db:backup           # produção — pode rodar com a mesa no ar
pnpm db:backup dev
```

Sai um `backups/t20-production-AAAAMMDD-HHMMSS.db`, já conferido com
`PRAGMA integrity_check`, e o comando de restauração impresso na tela.

A tela de administração tem o mesmo botão, e escreve no mesmo diretório — lá o
Go usa `VACUUM INTO`, que é a mesma garantia sem depender do `sqlite3` do host.

**Não copie o `.db` na mão.** Com WAL ligado, as transações recentes ainda estão
no arquivo `-wal`: um `cp` do `.db` sozinho leva um banco velho e não acusa erro
nenhum. Medido com a mesa no ar — o backup trazia a conta recém-criada e o `cp`
do `.db` não tinha nem as tabelas. O script usa o `.backup` do sqlite3, que lê um
snapshot coerente das duas partes.

## Os dois ambientes

`APP_ENV` escolhe o arquivo que o binário lê ao subir, e nada mais muda entre os
dois — é o mesmo binário (ALE-119):

| | dev (`pnpm dev`) | produção (`pnpm start`) |
|---|---|---|
| arquivo | `engine-go/.env.development` (versionado) | `engine-go/.env.production` (seu, não versionado) |
| quem serve o SPA | Vite :5173, proxiando `/api` | o próprio binário, mesma porta da API |
| banco | `engine-go/data/t20-dev.db` | `engine-go/data/t20-prod.db` |
| CORS | libera `http://localhost:5173` | nenhum header: tudo é mesma-origem |
| `JWT_SECRET` | público, no repositório | seu, e o boot **falha** sem ele |
| admin | `mestre@t20.local` (o do seed) | o seu, e o boot **falha** sem nenhum |

**O env do processo vence o arquivo**, então dá pra desviar sem editar nada:
`PORT=4000 pnpm dev`. E `ENV_FILE=/caminho/outro.env` troca o arquivo inteiro.

Variáveis (defaults em `engine-go/api/config.go`):

| var | default | o que faz |
|---|---|---|
| `APP_ENV` | `development` | qual `.env.<APP_ENV>` carregar, e se o boot é validado |
| `PORT` | `3001` | porta do servidor |
| `DATABASE_URL` | `file:./data/t20-dev.db` | arquivo SQLite; migra sozinho ao abrir |
| `JWT_SECRET` | — | assina os JWT de sessão; **obrigatório em produção** |
| `ADMIN_EMAILS` | — | quem administra, separado por vírgula; **obrigatório em produção** |
| `COOKIE_SECURE` | `false` | ligue quando houver TLS na frente — em HTTP na LAN, ligado, o browser descarta o cookie e o login não conclui |
| `CORS_ORIGIN` | `http://localhost:5173` (vazio em produção) | a ÚNICA origem liberada; vazio = sem CORS |
| `STATIC_DIR` | vazio | o `dist` do front; vazio = modo dev (o Vite serve) |
| `BACKUP_DIR` | `../backups` | onde o `pnpm db:backup` e a tela de admin escrevem |
| `CATALOG_PATH` | `parity/_catalogs.json` | catálogos dos validadores de mutação |
