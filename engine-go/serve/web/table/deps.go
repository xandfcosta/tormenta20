package table

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/app/rest"
	"t20engine/app/session"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
	"t20engine/serve/web/bookui"
	"t20engine/serve/web/sheetui"
	"t20engine/serve/web/ui"
)

// A PORTA da MESA, a mais larga da série: esta é a única cena que MOVIMENTA
// estado ao vivo. As outras leem o banco e desenham; esta abre e encerra cena,
// move peça, vira turno e empurra tudo para quem olha. O trilho do mestre tem
// DOIS métodos — a porta é fina quando a cena não precisa do servidor.
//
// Os STORES atravessam inteiros (`Boards`, `Sessions`, `Presence`, `Bus`): eles
// são o vocabulário do domínio ao vivo, e uma porta que os embrulhasse método a
// método teria oitenta entradas e nenhuma fronteira a mais.
//
// O `*sql.DB` NÃO atravessa: cena que compõe SQL é cena com o banco dentro.
type Deps interface {
	// Queries é o banco, pelas consultas geradas.
	Queries() *sqlcgen.Queries
	// Catalogs é o motor primado, para computar a ficha de quem senta à mesa.
	Catalogs() *engine.Catalogs
	// Boards são os tabuleiros vivos por sessão; Sessions é a fila e a cena.
	Boards() *board.BoardStore
	Sessions() *live.SessionStore
	// Presence é quem está online na sala; SSE são os leitores por sessão e
	// papel; Bus é o que aconteceu na mesa.
	Presence() *live.PresenceRegistry
	SSE() *live.SSEHub
	Bus() *events.Bus
	// CurrentUserID é quem está pedindo, pelo ID e não pelo usuário inteiro —
	// o tipo do usuário é do hospedeiro, e uma porta que o devolvesse não é
	// porta.
	CurrentUserID(r *http.Request) int64

	// PlaceDraftCampaign é a trava do RASCUNHO DE LUGAR, e não o
	// `SessionForCaller`: o rascunho acontece quando NÃO há sessão, e usá-lo
	// aqui exigiria inventar uma para autorizar preparação. É `gm` e não
	// "membro" — um jogador que abrisse esta tela veria a emboscada de sábado.
	//
	// Devolve a CAMPANHA e não um booleano porque a cena escreve o nome dela no
	// "voltar".
	//
	// A trava do lugar que já está numa MESA é do domínio (`EditPlace`), e não
	// desta porta: perguntar "qual é a sessão ativa" não vê o tabuleiro aberto
	// numa sessão encerrada, que reabre com ele.
	PlaceDraftCampaign(ctx context.Context, userID, campaignID int64) (campanha sqlcgen.Campaign, status int, err error)

	// SessionForCaller é a trava de acesso à mesa: existe, e quem pede pertence?
	//
	// Ela devolve a LINHA da sessão, o papel e o STATUS: a cena desenha os dois
	// primeiros — o número e o título no cabeçalho, o rodapé só para quem é
	// mestre — e responde o terceiro, porque quem está do outro lado é um
	// navegador esperando página.
	SessionForCaller(ctx context.Context, userID, campaignID, sessionID int64) (sqlcgen.Session, string, int, error)

	// SelfInitiativeEntry monta a linha de quem entra na fila com o próprio d20.
	SelfInitiativeEntry(userID, campaignID, characterID, d20 int64) (live.InitiativeEntry, error)
	// CloneCreatureBlock é o "chefe que ganha nome": o bloco é um MOLDE que duas
	// linhas dividem, e clonar só importa quando o mestre vai EDITAR uma delas —
	// sem a cópia, dar 30 PV ao chefe daria aos outros três zumbis também.
	//
	// É o BLOCO e não a ficha: clonar personagem exigiria matricular a cópia na
	// campanha, e todo membro aparece no painel do Grupo.
	CloneCreatureBlock(ctx context.Context, creatureID, campaignID int64, nome string) (int64, error)
	// MaterializeEntry transforma o pedido de linha nova (ficha, NPC, verbete)
	// na linha de fila que o store aceita.
	MaterializeEntry(ctx context.Context, userID, campaignID int64, pedido map[string]any) (live.InitiativeEntry, error)
	// PopulateParty põe na fila os combatentes que ainda não estão lá.
	PopulateParty(sessionID int64, quem []Combatant) (*live.SessionRuntimeState, error)
	// InitiativeBonus é o bônus de Iniciativa que a fila mostra, computado pelo
	// motor — a conta é regra do livro, e ela tem um dono só.
	InitiativeBonus(ctx context.Context, characterID int64) (int64, error)
	// SpeedsForBoard é o deslocamento de cada peça, que a prévia do movimento lê.
	SpeedsForBoard(board *board.BoardState) map[string]int

	// PUBLICAR é do hospedeiro: ele conhece o hub e o barramento, e a cena só
	// sabe QUANDO alguma coisa mudou.
	PublishSessionState(sessionID int64, estado *live.SessionRuntimeState)
	PublishBoardState(sessionID int64, board *board.BoardState)
	PublishWhatIsLeft(ctx context.Context, sessionID int64)
	CharacterChanged(characterID int64)

	// SaveNotes é a escrita que a cena montava em SQL. O título saiu daqui na
	// ALE-344 e mora no `app/session`; as notas seguem, e pelo mesmo caminho.
	SaveNotes(ctx context.Context, sessionID int64, texto string) error

	// PlayerSheet é a ficha EMBUTIDA, pedida PRONTA: montá-la aqui obrigaria a
	// Mesa a cumprir a `sheetui.Deps` inteira para desenhar um painel. Nulo é
	// caminho normal — mestre não tem ficha na mesa.
	PlayerSheet(r *http.Request, characterID int64) *sheetui.View
	// BookAddress é o endereço do livro, ou o zero quando não há `LIVRO_PDF`.
	// A cena não pergunta "há livro?": o valor já responde, como o leitor.
	BookAddress() bookui.BookAddress
	// Asset é o endereço VERSIONADO de um estático: os arquivos são `go:embed`
	// do hospedeiro.
	Asset(arquivo string) string
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// Combatant é quem entra na fila, na forma que a CENA declara: o `combatant` do
// hospedeiro não é exportado, e tipo não exportado não atravessa fronteira.
type Combatant struct {
	CharacterID int64
	Name        string
	HpCurrent   int64
	HpMax       int64
	MpCurrent   int64
	MpMax       int64
}

// Scene é a cena montada, com as dependências dela e o estado que é DELA: as
// lentes e as abas escolhidas. Elas vivem no servidor e não num sinal do
// navegador porque o stream não pergunta nada a ninguém.
//
// O `New` é chamado UMA vez, no registro das rotas: duas chamadas dariam dois
// estados, e metade da mesa não veria a lente da outra metade.
type Scene struct {
	deps Deps
	// lifecycle é o CASO DE USO do ciclo da sessão, e ele chega por parâmetro e
	// não pela `Deps` (ALE-344).
	//
	// A diferença é a razão inteira da camada: o `app/session` fica ABAIXO desta
	// cena e do `serve/api`, então a cena o importa DIRETO. Não há ciclo para
	// desviar, e por isso não há interface — cinco entradas que existiam só para
	// contornar a falta de um lugar saíram com ele.
	lifecycle session.Lifecycle
	// party é o CASO DE USO do descanso do grupo, e chega igual: por parâmetro,
	// porque o `app/` está abaixo desta cena.
	party      rest.Party
	lenses     *lenses
	chosenTabs *chosenTabs
}

func New(d Deps, ciclo session.Lifecycle, grupo rest.Party) Scene {
	return Scene{deps: d, lifecycle: ciclo, party: grupo, lenses: newLenses(), chosenTabs: newTabs()}
}
