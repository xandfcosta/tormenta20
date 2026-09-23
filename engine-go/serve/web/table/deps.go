package table

import (
	"context"
	"net/http"
	"t20engine/app/boards"

	"github.com/a-h/templ"

	"t20engine/app/campaign"
	"t20engine/app/character"
	"t20engine/app/combat"
	"t20engine/app/initiative"
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
	Boards() *boards.Store
	Sessions() *session.Store
	// Presence é quem está online na sala; SSE são os leitores por sessão e
	// papel; Bus é o que aconteceu na mesa.
	Presence() *live.PresenceRegistry
	SSE() *live.SSEHub
	Bus() *events.Bus
	// CurrentUserID é quem está pedindo, pelo ID e não pelo usuário inteiro —
	// o tipo do usuário é do hospedeiro, e uma porta que o devolvesse não é
	// porta.
	CurrentUserID(r *http.Request) int64

	// PUBLICAR é do hospedeiro, e continua sendo depois da camada de aplicação
	// (ALE-344). Não é sobra: o `PublishSessionState` conta no contador de
	// goroutines que o `Shutdown` do servidor espera, e esse contador é do
	// PROCESSO — um caso de uso que o carregasse estaria segurando o
	// desligamento.
	//
	// Os três saem quando os STORES saírem, e não antes: eles são a gravação dos
	// stores, não um gesto. Ver a nota do `app/` no guia.
	PublishSessionState(sessionID int64, state *live.SessionRuntimeState)
	PublishBoardState(ctx context.Context, sessionID int64, board *board.BoardState)
	PublishWhatIsLeft(ctx context.Context, sessionID int64)
	// CharacterChanged é a regra da FICHA, que a Mesa pede emprestada: avisa a
	// tela de quem tem aquela ficha aberta. Ela fica aqui porque o gesto que a
	// dispara — mexer nos vitais de uma linha — ainda é uma chamada de STORE, e
	// um caso de uso que só notificasse não seria um caso de uso. Ela vai junto
	// com o store da fila.
	CharacterChanged(characterID int64)

	// PlayerSheet é a ficha EMBUTIDA, pedida PRONTA: montá-la aqui obrigaria a
	// Mesa a cumprir a `sheetui.Deps` inteira para desenhar um painel. Nulo é
	// caminho normal — mestre não tem ficha na mesa.
	PlayerSheet(r *http.Request, characterID int64) *sheetui.View
	// BookAddress é o endereço do livro, ou o zero quando não há `LIVRO_PDF`.
	// A cena não pergunta "há livro?": o valor já responde, como o leitor.
	BookAddress() bookui.BookAddress
	// Asset é o endereço VERSIONADO de um estático: os arquivos são `go:embed`
	// do hospedeiro.
	Asset(file string) string
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, body templ.Component)
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
	party rest.Party
	// queue é o CASO DE USO de quem entra na fila.
	queue initiative.Queue
	// cast é o CASO DE USO do elenco de NPCs da campanha. Ele é da CAMPANHA e
	// não desta cena: o NPC preparado na quinta sobrevive à sessão de sábado, e
	// a Mesa é uma entrada do gesto (ALE-353).
	cast campaign.Cast
	// plays são os GESTOS da ficha, e chegam como os outros casos de uso: a
	// condição de um personagem marcada pela Mesa grava na ficha (ALE-368).
	plays character.Plays
	// strike é o CASO DE USO de atacar, e chega pelo mesmo caminho dos outros:
	// por parâmetro, porque o `app/` está abaixo desta cena.
	strike combat.Strike
	// gestures são os gestos que escrevem no tabuleiro E na fila, e por isso
	// abrem a TRANSAÇÃO (ALE-376) — a cena não abre nenhuma. Eles chegam por
	// parâmetro como os outros casos de uso, porque o `app/` está abaixo desta
	// cena e não há ciclo a desviar.
	gestures boards.Gestures
	// access é a TRAVA da sessão, e ela é o MESMO objeto que os casos de uso
	// usam por dentro (ALE-344). A cena a chama para decidir o que DESENHAR —
	// o rodapé do mestre, a recusa antes do gesto —, e quem decide se o gesto
	// pode é o caso de uso. Duas perguntas, uma implementação.
	access session.Access
	// Os dois mapas por `(sessão, pessoa)` vêm de FORA, dentro do
	// `EphemeralTableState`: o fim de uma sessão tem de alcançá-los, e quem os
	// montasse aqui dentro não teria como ser avisado — ver o
	// `ephemeral_state.go`. A cena guarda as duas metades direto porque é assim
	// que ela as usa; o objeto inteiro é do hospedeiro.
	lenses     *lenses
	chosenTabs *chosenTabs
}

func New(
	d Deps, cycle session.Lifecycle, ephemeral *EphemeralTableState, group rest.Party,
	queue initiative.Queue, cast campaign.Cast, plays character.Plays, strike combat.Strike,
	gestures boards.Gestures,
) Scene {
	return Scene{
		deps: d, lifecycle: cycle, party: group, queue: queue, cast: cast, plays: plays, strike: strike,
		gestures: gestures,
		access:   cycle.Access(),
		lenses:   ephemeral.lenses, chosenTabs: ephemeral.chosenTabs,
	}
}
