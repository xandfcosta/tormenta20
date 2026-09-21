package table

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"t20engine/app"
	"t20engine/app/initiative"
	"t20engine/domain/book"
	"t20engine/serve/web/master"
	"t20engine/serve/web/routes"
	"t20engine/serve/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/domain/live"
)

// O BESTIÁRIO DENTRO DA MESA.
//
// A cena do bestiário já existe em `/mestre/bestiario`, e o caminho mais barato
// teria sido mandar o mestre até lá. Não é o que a mesa quer: uma emboscada é
// UMA viagem, não seis — ele abre, escolhe o ogro, manda dois, vê que faltou um
// goblin e manda mais. Sair da fila a cada bicho paga o custo justamente no
// momento em que a mesa está esperando.
//
// O DESENHO é o mesmo, e é isso que a `bestiarioView.Base` comprou: a lista, os
// filtros e o bloco são os mesmos componentes da cena do mestre, apontando para
// as rotas daqui.
//
// O painel mora FORA do `<main id="table">`, e essa é a diferença que faz ele
// funcionar. A cena é remendada inteira a cada mudança de qualquer um na mesa; o
// que o painel guarda — a busca digitada, o filtro aceso, a criatura aberta — é
// estado de TELA e não da sessão, e um remendo da mesa o apagaria a cada turno
// que alguém avançasse.

// tableBestiaryRoute é a base das rotas do painel, montada por mesa.
func tableBestiaryRoute(campaignID, sessionID int64) string {
	return routes.Session(campaignID, sessionID) + "/bestiario"
}

func (s Scene) TableBestiaryRoutes(r chi.Router) {
	r.Get(sessionPattern+"/bestiario", s.handleBestiaryTable)
	r.Post(sessionPattern+"/bestiario/tipo/{tipo}", s.handleKindBestiaryTable)
	r.Post(sessionPattern+"/bestiario/enviar", s.gmCommand(sendsForTable))
}

// forTableBestiary monta a view do painel para esta mesa.
func (s Scene) forTableBestiary(r *http.Request, campaignID, sessionID int64) master.BestiaryView {
	c := master.BestiaryCriteriaFromRequest(r)
	v := master.LoadBestiaryFrom(tableBestiaryRoute(campaignID, sessionID), s.deps.BookAddress(), c.Term, c.Types, c.CRMin, c.CRMax, c.Chosen)
	v.Open = c.Open
	return v
}

// handleBestiaryTable redesenha o PAINEL e mais nada.
//
// Remendar o `#table` junto seria redesenhar a fila inteira a cada tecla da
// busca — e pior, apagaria o que o mestre está digitando, porque o campo vive
// dentro do painel.
func (s Scene) handleBestiaryTable(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := s.tableGmOrRefusal(w, r)
	if !ok {
		return
	}
	// LER OS SINAIS ANTES do `NewSSE`, que assume a resposta. Aqui é GET, então
	// eles vêm da query string e reler é barato — mas a ordem continua sendo a
	// regra, e ela já custou um defeito que só apareceu no navegador (ver o
	// comentário do `action.go`).
	draft := signalsDraft(r)
	v := s.forTableBestiary(r, campaignID, sessionID)

	sse := datastar.NewSSE(w, r)
	if fragment, err := ui.RenderFragment(r.Context(), tableBestiary(v)); err == nil {
		_ = sse.PatchElements(fragment)
	}
	// O PAINEL É O DONO DO RASCUNHO: os campos de PV, iniciativa e quantas
	// nascem do bloco do livro a cada criatura ABERTA. Sem isto, o PV que o
	// mestre baixou para um ogro reapareceria no próximo bicho e ele não teria
	// como saber que carregou.
	//
	// A comparação com o `rascunhode` é o que separa "abriu outra criatura" de
	// "digitou na busca": só a primeira semeia. Sem ela, filtrar apagaria o PV
	// que o mestre acabou de ajustar.
	if v.Chosen != nil && draft != v.Chosen.ID {
		_ = sse.MarshalAndPatchSignals(entryDraft(*v.Chosen))
	}
}

// entryDraft são os três campos do ajuste, nascidos do livro.
//
// A INICIATIVA é um d20 ROLADO e não o bônus da criatura: o mestre quer a linha
// entrando com uma rolagem, e ajusta se rolou nos dados de verdade em cima da
// mesa. Rolar no SERVIDOR e não na página é o mesmo princípio do d20 do jogador
// — a página não faz conta que vale.
func entryDraft(m book.Entry) map[string]any {
	return map[string]any{
		"entry_hp":         m.HP,
		"entry_initiative": rand.IntN(20) + 1,
		"entry_copies":     1,
		"draft_of":         m.ID,
	}
}

// signalsDraft lê de QUAL criatura o rascunho na tela é.
func signalsDraft(r *http.Request) string {
	var signals struct {
		De string `json:"draft_of"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return ""
	}
	return signals.De
}

// handleKindBestiaryTable liga ou desliga um crachá de tipo.
//
// Espelha o `handleBestiaryType` da cena do mestre; o que muda é o que ele
// remenda. Não dá para reusar aquele porque ele responde com a cena inteira.
func (s Scene) handleKindBestiaryTable(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := s.tableGmOrRefusal(w, r)
	if !ok {
		return
	}
	criteria := master.BestiaryCriteriaFromRequest(r)
	kinds, err := master.ToggleType(criteria.Types, chi.URLParam(r, "tipo"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	criteria.Types = kinds
	v := master.LoadBestiaryFrom(
		tableBestiaryRoute(campaignID, sessionID), s.deps.BookAddress(),
		criteria.Term, criteria.Types, criteria.CRMin, criteria.CRMax, criteria.Chosen,
	)

	sse := datastar.NewSSE(w, r)
	if fragment, err := ui.RenderFragment(r.Context(), tableBestiary(v)); err == nil {
		_ = sse.PatchElements(fragment)
	}
	// O sinal volta porque o crachá é a ÚNICA coisa que muda a lista sem passar
	// por um campo ligado: sem isto, a próxima busca mandaria os tipos velhos.
	_ = sse.MarshalAndPatchSignals(map[string]any{"tipos": criteria.Types})
}

// sendsForTable põe N cópias do verbete na fila.
//
// UMA ENTRADA POR CÓPIA, e quem numera os repetidos é o SERVIDOR: a tela não
// pode adivinhar um número que outro cliente acabou de usar. Todas entram com a
// MESMA iniciativa — é o que a mesa faz com um bando.
//
// O `monsterId` viaja junto porque é ele que liga a linha ao verbete do livro, e
// é o que faz o painel do combatente mostrar o bloco depois.
func sendsForTable(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	send, err := envioDosSinais(c.R)
	if err != nil {
		return nil, err
	}
	m := book.EntryByID(send.Creature)
	if m == nil {
		return nil, fmt.Errorf("criatura %q não está no bestiário", send.Creature)
	}
	if err := live.ValidateCombatantDraft(live.CombatantDraft{
		Label: m.Name, Initiative: send.Initiative, HP: send.PV, Kind: "npc",
	}); err != nil {
		return nil, err
	}
	if send.Copies < 1 || send.Copies > maxCopiasDeUmVerbete {
		return nil, fmt.Errorf("quantas %d está fora da faixa de 1 a %d", send.Copies, maxCopiasDeUmVerbete)
	}

	var state *live.SessionRuntimeState
	for i := 0; i < send.Copies; i++ {
		init, pv := int64(send.Initiative), int64(send.PV)
		row, err := st.queue.Roster().Entry(c.R.Context(), app.Caller{ID: c.User}, c.CampaignID,
			initiative.EntryRequest{
				Label: m.Name, Initiative: &init, Kind: "npc",
				MonsterID: m.ID, HpCurrent: &pv, HpMax: &pv,
			})
		if err != nil {
			return state, err
		}
		// O parcial volta junto com o erro: quatro goblins que entraram são o
		// estado da mesa, e o `gmCommand` o transmite.
		if state, err = st.deps.Sessions().AddInitiativeEntry(c.SessionID, row); err != nil {
			return state, err
		}
	}
	return state, nil
}

// maxCopiasDeUmVerbete é o teto de cópias num gesto.
//
// Não é regra do livro: é o que separa "quatro goblins" de um zero a mais que
// enche a fila e o mestre tem de desfazer linha por linha. A fila tem teto
// próprio no servidor, e ele responde por si.
const maxCopiasDeUmVerbete = 12

type envioDoVerbete struct {
	Creature   string
	PV         int64
	Initiative int
	Copies     int
}

func envioDosSinais(r *http.Request) (envioDoVerbete, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var signals struct {
		Creature   string `json:"creature"`
		PV         int64  `json:"entry_hp"`
		Initiative int    `json:"entry_initiative"`
		Copies     int    `json:"entry_copies"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return envioDoVerbete{}, fmt.Errorf("não entendi o envio: %v", err)
	}
	return envioDoVerbete{
		Creature: signals.Creature, PV: signals.PV,
		Initiative: signals.Initiative, Copies: signals.Copies,
	}, nil
}

// tableGmOrRefusal resolve a mesa e exige o papel, para as rotas do painel
// que NÃO passam pelo `gmCommand` — as que só leem.
func (s Scene) tableGmOrRefusal(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return 0, 0, false
	}
	_, role, err := s.access.Session(r.Context(), app.Caller{ID: s.deps.CurrentUserID(r)}, campaignID, sessionID)
	status := statusOf(err)
	if err != nil {
		http.Error(w, err.Error(), status)
		return 0, 0, false
	}
	// O bestiário é do mestre INTEIRO, não só o mandar para a mesa: a lista diz
	// o PV e a defesa de cada bicho, e é isso que o mestre esconde quando aperta
	// o olho numa linha.
	if role != "gm" {
		http.Error(w, "o bestiário da mesa é do mestre", http.StatusForbidden)
		return 0, 0, false
	}
	return campaignID, sessionID, true
}
