# tormenta20

pnpm workspace monorepo.

```
engine-go/ Go: API (:3001), motor de regras, e as CENAS em templ + Datastar —
           com a folha e as ilhas de JS delas em api/piloto/src (Tailwind v4)
e2e/       Playwright, dirigindo o app rodando
```

A linguagem do domínio mora em **[GLOSSARIO.md](GLOSSARIO.md)**: uma palavra por
conceito, os sinônimos proibidos e a regra de quando se escreve em português e
quando em inglês. Consulte antes de nomear algo que o usuário vá ler.

Houve três stacks de tela: React, depois SolidJS (ALE-76), e desde a ALE-272 as
cenas são renderizadas pelo SERVIDOR com templ + Datastar. O backend NestJS
também foi removido — o Go é o app, e migra o próprio banco (goose embutido) ao
subir. O histórico das migrações vive no `git log` e nas issues do Linear.

## Setup

```bash
pnpm install
```

Não há passo de migração: `db.Open` aplica as migrações embutidas na partida.

## Dev

```bash
pnpm dev                 # a API e as cenas, com recarga a quente (:3001)
```

Um servidor só: abra `http://localhost:3001`. Mexeu num `.templ`? `go tool templ
generate`. Mexeu na folha ou nas ilhas de JS? `pnpm -F @tormenta20/engine-go run
prebuild`.

A configuração vem de `engine-go/.env.development`, que é **versionado**: nada
ali é segredo, e o servidor recusa subir em produção com aquele `JWT_SECRET`.
Nenhum passo de setup — funciona no clone. O `mestre@t20.local` do seed já é
admin ali, então o menu do Hub tem **Convidar jogador** desde o primeiro boot.

## Testes

```bash
cd engine-go && go test ./...   # a suíte inteira do app
pnpm test:e2e                   # Playwright — sobe o próprio servidor e o próprio banco
```

O e2e tem comando próprio e job próprio no CI (ALE-93): ele instala browser e
sobe a API, e disputar CPU com o resto o derruba por timeout.

## Produção (a mesa na LAN)

Duas formas, e as duas sobem **o mesmo processo único** — o binário serve as
cenas, a API e o fluxo ao vivo na mesma porta, com tudo comprimido por ele mesmo.
Não há proxy reverso em nenhuma das duas.

### Por contêiner (ALE-273)

É a forma recomendada: o build é reproduzível, o processo volta sozinho depois
de um reboot e o banco fica numa pasta que dá para copiar.

```bash
cp .env.example .env     # escreva o JWT_SECRET e o ADMIN_EMAILS
mkdir -p data backups
docker compose up -d --build
```

O `.env` pede o `T20_UID`/`T20_GID` do dono da pasta `data/` (`id -u`, `id -g`):
o contêiner não roda como root, e sem isso ele não consegue escrever no bind
mount — o sintoma é `unable to open database file`, que parece banco corrompido
e é permissão.

O **backup é copiar a pasta `data/`**, e o backup automático já escreve em
`backups/` com o mesmo `VACUUM INTO` do manual.

Livro em PDF e certificado TLS ficam num `docker-compose.override.yml`, porque
apontam para caminhos que só existem na sua máquina — o modelo está comentado no
fim do `docker-compose.yml`.

### Direto na máquina

```bash
cp engine-go/.env.production.example engine-go/.env.production
openssl rand -hex 32     # cole no JWT_SECRET do arquivo copiado
pnpm build               # a folha, as ilhas e o binário do servidor
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

O admin abre e edita **qualquer mesa** (as dos outros aparecem nas Campanhas
marcadas com o dono) e tem a cena de **Administração** no menu do Hub: quem está
na mesa e o que cada um tem, os convites em aberto, e o painel de servidor com
backup. Duas ações sobre uma conta: gerar um **link de redefinição de senha** —
você nunca digita nem vê a senha de ninguém — e **apagar**, que leva as fichas
junto e passa as mesas dela para você.

**A stack de produção é um processo só.** O `cmd/api` serve as cenas, os
estáticos (que viajam DENTRO do binário) e `/api/*` na mesma porta. Não há nginx
e não há segundo runtime pra manter — foi por isso que os dois saíram quando o
Nest saiu.

### HTTPS na LAN, e o app instalado no telefone

Em `http://` o navegador não considera a página **contexto seguro**, e é daí que
vem o buraco. Medido neste servidor, com o mesmo endereço de LAN nos dois
esquemas:

| pelo `http://192.168.x.x:3001` | pelo `https://…` |
|---|---|
| `window.isSecureContext` **false** | **true** |
| `'serviceWorker' in navigator` **false** | **true** |

Sem contexto seguro nenhum service worker registra, e o Chrome do Android não
oferece instalar: o "Adicionar à Tela de Início" devolve um **atalho**, uma aba
comum com a barra de endereço de volta.

Com HTTPS, o `manifest.webmanifest` que já está no `index.html` passa a valer e
o app abre em janela própria nos **dois** sistemas. A **Tela cheia** do menu do
Hub continua existindo e é outra coisa — é o gesto de quem não instalou, e é a
saída do desktop (glossário, colisão C7).

**Não há service worker, e a ausência é deliberada** (ALE-118, decisão 4 ainda
aberta). Ele traria cache offline e traria junto o problema clássico da versão
velha grudada, sem caminho óbvio para o jogador sair dela — e o app depende de
buscar catálogo por HTTP (`GET /catalog/:nome`), então a estratégia de cache não
é detalhe. Isso se decide antes de escrever, não depois.

O TLS termina **neste processo**, não num proxy na frente: a stack de produção é
um binário só, e é a decisão que o `engine-go/CLAUDE.md` registra.

```bash
# no .env.production
TLS_CERT_FILE=/caminho/cert.pem
TLS_KEY_FILE=/caminho/key.pem
COOKIE_SECURE=true
```

O log passa a imprimir `https://…`, que é o endereço que a mesa digita. Quem
digitar `http://` na mesma porta recebe um 400 do próprio Go — feio, mas
visível.

**Terminar o TLS fora também funciona** (um túnel, um proxy): deixe os dois
caminhos vazios, mantenha `COOKIE_SECURE=true`, e o processo segue falando HTTP
para quem está na frente.

#### De onde vem o certificado — decisão em ABERTO

As duas saídas, com o preço de cada uma. Nenhuma delas está feita neste
repositório, e nenhuma delas se faz de dentro dele:

| | `mkcert` | domínio próprio + Let's Encrypt (DNS-01) |
|---|---|---|
| custo por aparelho | instalar a CA em **cada** telefone que sentar à mesa | nenhum |
| custo fixo | nenhum | ter um domínio e um DNS com API |
| internet | nenhuma | só de **saída**, para renovar |
| exposição | nenhuma | nenhuma — o registro A aponta para o IP **privado**, que só responde de dentro da rede |

`mkcert` é o caminho de uma tarde; o DNS-01 é o que não cobra nada por telefone
novo. Diferença que decide junto com a tabela: o `mkcert` emite para um IP se
você pedir, e o Let's Encrypt só emite para **nome** — o que puxa a outra metade
do problema.

#### Chegar no app sem digitar um IP

Hoje o jogador digita `http://192.168.15.3:3001`, um número que muda quando o
roteador reinicia. Duas coisas baratas, na ordem em que valem a pena, e **as
duas fora do código**:

1. **Reserva de DHCP** no roteador, para o IP parar de mudar. Custa cinco
   minutos e não exige nada deste repositório.
2. **Um nome** (`t20.local` por mDNS, ou o nome da reserva) — e é para esse
   nome que o certificado do passo anterior é emitido.

Um QR code na tela do mestre resolveria o resto, e é trabalho de tela: está
anotado na ALE-118, fora desta parte.

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
| quem serve as cenas | o próprio binário | o próprio binário |
| banco | `engine-go/data/t20-dev.db` | `engine-go/data/t20-prod.db` |
| CORS | libera `http://localhost:5173` (herança da SPA) | nenhum header: tudo é mesma-origem |
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
| `TLS_CERT_FILE` / `TLS_KEY_FILE` | vazios | o par de certificados; os DOIS preenchidos = este processo fala HTTPS, os dois vazios = HTTP. Meio par **derruba o boot** |
| `BACKUP_DIR` | `../backups` | onde o `pnpm db:backup` e a tela de admin escrevem |
| `CATALOG_PATH` | `parity/_catalogs.json` | catálogos dos validadores de mutação |
| `LIVRO_PDF` | vazio | o Tormenta 20 em PDF que o servidor entrega em `/piloto/livro`; vazio = não serve nada e o botão "abrir no livro" não aparece. **Linearize antes** (`qpdf --linearize`): medido, o navegador transfere o arquivo inteiro para abrir uma página nos dois casos, e o qpdf encolhe os 89,5 MB para 78,6 — o boot avisa quando não está |
| `LIVRO_ABERTURA` | `6` | quantas páginas o arquivo tem antes da página impressa 1 — `#page=N` conta páginas do ARQUIVO e o catálogo grava a IMPRESSA |
