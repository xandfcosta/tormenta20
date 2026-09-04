package table

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/aovivo"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/events"
	"t20engine/tabuleiro"
	"t20engine/web/bookui"
	"t20engine/web/sheetui"
	"t20engine/web/ui"
)

// A PORTA da MESA (ALE-278), a última cena a sair do `api`.
//
// Ela é a mais larga da série, e o motivo não é indisciplina: esta cena é a
// única que MOVIMENTA estado ao vivo. As outras leem o banco e desenham; esta
// abre e encerra cena, move peça, pinta terreno, mede distância, vira turno e
// empurra tudo isso para quem está olhando — por dois stores em memória, um
// hub de SSE e um barramento de eventos.
//
// O contraste que ensina é com o trilho do mestre, que tem DOIS métodos: ele
// desenha o livro embutido e não toca banco. **A porta é fina quando a cena não
// precisa do servidor, não quando alguém foi disciplinado.**
//
// # Os STORES atravessam, e os campos do `Server` não
//
// `Boards`, `Sessions`, `Presence` e `Bus` são tipos de OUTROS pacotes
// (`tabuleiro`, `aovivo`, `events`) — a cena os recebe inteiros pela mesma
// razão que a forja recebe o `Queries`: eles são o vocabulário do domínio ao
// vivo, não o hospedeiro com outro nome. Uma porta que os embrulhasse método a
// método teria oitenta entradas e nenhuma fronteira a mais.
//
// O que NÃO atravessa é o `*sql.DB`. Duas gravações compunham `setBuilder` +
// `"UPDATE sessions"` aqui dentro, e cena que compõe SQL é cena com o banco
// dentro — viraram `SaveNotes` e `SaveSessionTitle`.
type Deps interface {
	// Queries é o banco, pelas consultas geradas.
	Queries() *sqlcgen.Queries
	// Catalogs é o motor primado, para computar a ficha de quem senta à mesa.
	Catalogs() *engine.Catalogs
	// Boards são os tabuleiros vivos por sessão; Sessions é a fila e a cena.
	Boards() *tabuleiro.BoardStore
	Sessions() *aovivo.SessionStore
	// Presence é quem está online na sala; SSE são os leitores por sessão e
	// papel; Bus é o que aconteceu na mesa.
	Presence() *aovivo.PresenceRegistry
	SSE() *aovivo.SSEHub
	Bus() *events.Bus
	// CurrentUserID é quem está pedindo, pelo ID e não pelo usuário inteiro —
	// o tipo do usuário é do hospedeiro, e uma porta que o devolvesse não é
	// porta.
	CurrentUserID(r *http.Request) int64
	// IsAdmin diz se quem pede administra, para a cena que o rodapé oferece.
	IsAdminRequester(ctx context.Context, userID int64) bool

	// SessionForCaller é a trava de acesso à mesa: existe, e quem pede pertence?
	//
	// Ela devolve a LINHA da sessão, o papel e o STATUS: a cena desenha os dois
	// primeiros — o número e o título no cabeçalho, o rodapé só para quem é
	// mestre — e responde o terceiro, porque quem está do outro lado é um
	// navegador esperando página.
	SessionForCaller(ctx context.Context, userID, campaignID, sessionID int64) (sqlcgen.Session, string, int, error)

	// As perguntas do estado AO VIVO, que o hospedeiro sabe responder porque a
	// rota JSON faz as mesmas.
	// Os três primeiros levam sufixo `ForTable` porque o `*Server` JÁ tem
	// `StartSession`, `EndSession` e `RestartCombat` — com outra forma e outra
	// pergunta: aqueles recebem a LINHA da sessão e devolvem a linha gravada,
	// estes recebem o id e devolvem o estado AO VIVO, que é o que a cena
	// redesenha. É a colisão que o `IsAdmin` das campanhas registrou, e a regra
	// dela: um contrato que já existe ganha quando é a MESMA pergunta; quando só
	// a cara é a mesma, forçar um nome só junta duas coisas diferentes. O sufixo
	// não foi inventado agora — o `endSceneForTable` já usava.
	StartSessionForTable(ctx context.Context, sessionID int64) (*aovivo.SessionRuntimeState, error)
	EndSessionForTable(ctx context.Context, sessionID int64) (*aovivo.SessionRuntimeState, error)
	RestartCombatForTable(ctx context.Context, sessionID int64) (*aovivo.SessionRuntimeState, error)
	EndSceneForTable(userID, campaignID, sessionID int64) (*aovivo.SessionRuntimeState, error)
	RestParty(userID, campaignID, sessionID int64, escopo, condicao string) (int, int, error)
	// SelfInitiativeEntry monta a linha de quem entra na fila com o próprio d20.
	SelfInitiativeEntry(userID, campaignID, characterID, d20 int64) (aovivo.InitiativeEntry, error)
	// MaterializeEntry transforma o pedido de linha nova (ficha, NPC, verbete)
	// na linha de fila que o store aceita.
	MaterializeEntry(ctx context.Context, userID, campaignID int64, pedido map[string]any) (aovivo.InitiativeEntry, error)
	// PlayerCombatants são os personagens dos jogadores da campanha, e
	// PopulateParty põe os que faltam no mapa.
	PlayerCombatants(ctx context.Context, campaignID int64) ([]Combatant, error)
	PopulateParty(sessionID int64, quem []Combatant) (*aovivo.SessionRuntimeState, error)
	// InitiativeBonus e ComputedSheet são a ficha computada que a fila e o
	// elenco mostram.
	InitiativeBonus(ctx context.Context, characterID int64) (int64, error)
	ComputedSheet(ctx context.Context, row sqlcgen.Character) (engine.ComputedSheetV2, error)
	// SpeedsForBoard é o deslocamento de cada peça, que a prévia do movimento lê.
	SpeedsForBoard(board *tabuleiro.BoardState) map[string]int

	// PUBLICAR é do hospedeiro: ele conhece o hub e o barramento, e a cena só
	// sabe QUANDO alguma coisa mudou.
	PublishSessionState(sessionID int64, estado *aovivo.SessionRuntimeState)
	PublishBoardState(sessionID int64, board *tabuleiro.BoardState)
	PublishWhatIsLeft(ctx context.Context, sessionID int64)
	CharacterChanged(characterID int64)

	// As DUAS escritas que a cena montava em SQL.
	SaveNotes(ctx context.Context, sessionID int64, texto string) error
	SaveSessionTitle(ctx context.Context, sessionID int64, titulo string) error

	// PlayerSheet é a ficha EMBUTIDA de quem senta à mesa (ALE-275).
	//
	// A cena não monta a cena da ficha: ela pede o painel pronto. Montá-la aqui
	// obrigaria a Mesa a cumprir a `sheetui.Deps` inteira — dezoito métodos que
	// ela não usa — só para desenhar um painel. Nulo é caminho normal: quem é
	// mestre não tem ficha na mesa, e uma ficha que não carrega tira a aba da
	// tela sem derrubar a sessão.
	PlayerSheet(r *http.Request, characterID int64) *sheetui.View
	// BookAddress é o endereço do livro, ou o zero quando não há `LIVRO_PDF`.
	// A cena não pergunta "há livro?": o valor já responde, como o leitor.
	BookAddress() bookui.BookAddress
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// Combatant é quem entra na fila, na forma que a CENA declara.
//
// O hospedeiro tem um `combatant` de campos minúsculos, e ele é dele: um tipo
// não exportado não atravessa fronteira. A cena declara o que ela desenha e o
// hospedeiro mapeia — é o `ListRow` das campanhas outra vez.
type Combatant struct {
	CharacterID int64
	Name        string
	HpCurrent   int64
	HpMax       int64
	MpCurrent   int64
	MpMax       int64
}

// Scene é a cena montada, com as dependências dela e o estado que é DELA.
//
// As lentes e as abas escolhidas eram CAMPOS do `*Server`, e não deviam ser: a
// pergunta "quem está vendo como a mesa vê" e "que tabuleiro cada um está
// olhando" só existem nesta tela. Elas moram no servidor e não num sinal do
// navegador porque o stream não pergunta nada a ninguém — mas o dono é a cena.
//
// Por isso o `New` é chamado UMA vez, no registro das rotas: duas chamadas
// dariam dois estados, e metade da mesa não veria a lente da outra metade.
type Scene struct {
	deps       Deps
	lenses     *lenses
	chosenTabs *chosenTabs
}

func New(d Deps) Scene {
	return Scene{deps: d, lenses: newLenses(), chosenTabs: newTabs()}
}
