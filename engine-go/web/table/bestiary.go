package table

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"t20engine/book"
	"t20engine/web/master"
	"t20engine/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/aovivo"
)

// O BESTIÁRIO DENTRO DA MESA (ALE-263).
//
// A cena do bestiário já existe em `/mestre/bestiario`, e o caminho mais
// barato teria sido mandar o mestre até lá. Não é o que a mesa quer: uma
// emboscada é UMA viagem, não seis — o mestre abre, escolhe o ogro, manda dois,
// vê que faltou um goblin e manda mais. Sair da fila a cada bicho paga o custo
// justamente no momento em que a mesa está esperando.
//
// O DESENHO é o mesmo, e é isso que a `bestiarioView.Base` comprou: a lista, os
// filtros e o bloco são os mesmos componentes da cena do mestre, apontando para
// as rotas daqui.
//
// O painel mora FORA do `<main id="mesa">`, e essa é a diferença que faz ele
// funcionar. A cena é remendada inteira a cada mudança de qualquer um na mesa; o
// que o painel guarda — a busca digitada, o filtro aceso, a criatura aberta — é
// estado de TELA e não da sessão, e um remendo da mesa o apagaria a cada turno
// que alguém avançasse.

// tableBestiaryRoute é a base das rotas do painel, montada por mesa.
func tableBestiaryRoute(campaignID, sessionID int64) string {
	return fmt.Sprintf("/mesa/%d/%d/bestiario", campaignID, sessionID)
}

func (s Scene) TableBestiaryRoutes(r chi.Router) {
	r.Get("/mesa/{campaignId}/{sessionId}/bestiario", s.handleBestiaryTable)
	r.Post("/mesa/{campaignId}/{sessionId}/bestiario/tipo/{tipo}", s.handleKindBestiaryTable)
	r.Post("/mesa/{campaignId}/{sessionId}/bestiario/enviar", s.gmCommand(sendsForTable))
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
// Remendar o `#mesa` junto seria redesenhar a fila inteira a cada tecla da
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
	rascunho := signalsDraft(r)
	v := s.forTableBestiary(r, campaignID, sessionID)

	sse := datastar.NewSSE(w, r)
	if fragmento, err := ui.RenderFragment(r.Context(), tableBestiary(v)); err == nil {
		_ = sse.PatchElements(fragmento)
	}
	// O PAINEL É O DONO DO RASCUNHO: os campos de PV, iniciativa e quantas
	// nascem do bloco do livro a cada criatura ABERTA. Sem isto, o PV que o
	// mestre baixou para um ogro reapareceria no próximo bicho e ele não teria
	// como saber que carregou (é a mesma regra que o diálogo da SPA carrega).
	//
	// A comparação com o `rascunhode` é o que separa "abriu outra criatura" de
	// "digitou na busca": só a primeira semeia. Sem ela, filtrar apagaria o PV
	// que o mestre acabou de ajustar.
	if v.Chosen != nil && rascunho != v.Chosen.ID {
		_ = sse.MarshalAndPatchSignals(entryDraft(*v.Chosen))
	}
}

// entryDraft são os três campos do ajuste, nascidos do livro.
//
// A INICIATIVA é um d20 ROLADO e não o bônus da criatura, e é o que a SPA faz:
// o mestre quer a linha entrando com uma rolagem, e ajusta se rolou nos dados de
// verdade em cima da mesa. Rolar no SERVIDOR e não na página é o mesmo princípio
// do d20 do jogador (ALE-213) — a página não faz conta que vale.
func entryDraft(m book.Entry) map[string]any {
	return map[string]any{
		"pvdoverbete":     m.HP,
		"inidoverbete":    rand.IntN(20) + 1,
		"copiasdoverbete": 1,
		"rascunhode":      m.ID,
	}
}

// signalsDraft lê de QUAL criatura o rascunho na tela é.
func signalsDraft(r *http.Request) string {
	var sinais struct {
		De string `json:"rascunhode"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return ""
	}
	return sinais.De
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
	criterios := master.BestiaryCriteriaFromRequest(r)
	tipos, err := master.ToggleType(criterios.Types, chi.URLParam(r, "tipo"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	criterios.Types = tipos
	v := master.LoadBestiaryFrom(
		tableBestiaryRoute(campaignID, sessionID), s.deps.BookAddress(),
		criterios.Term, criterios.Types, criterios.CRMin, criterios.CRMax, criterios.Chosen,
	)

	sse := datastar.NewSSE(w, r)
	if fragmento, err := ui.RenderFragment(r.Context(), tableBestiary(v)); err == nil {
		_ = sse.PatchElements(fragmento)
	}
	// O sinal volta porque o crachá é a ÚNICA coisa que muda a lista sem passar
	// por um campo ligado: sem isto, a próxima busca mandaria os tipos velhos.
	_ = sse.MarshalAndPatchSignals(map[string]any{"tipos": criterios.Types})
}

// sendsForTable põe N cópias do verbete na fila.
//
// UMA ENTRADA POR CÓPIA, e quem numera os repetidos é o SERVIDOR (ALE-192): a
// tela não pode adivinhar um número que outro cliente acabou de usar. Todas
// entram com a MESMA iniciativa — é o que a mesa faz com um bando.
//
// O `monsterId` viaja junto porque é ele que liga a linha ao verbete do livro, e
// é o que faz o painel do combatente mostrar o bloco depois.
func sendsForTable(st Scene, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	envio, err := envioDosSinais(c.R)
	if err != nil {
		return nil, err
	}
	m := book.EntryByID(envio.Criatura)
	if m == nil {
		return nil, fmt.Errorf("criatura %q não está no bestiário", envio.Criatura)
	}
	if err := aovivo.ValidateCombatantDraft(aovivo.CombatantDraft{
		Label: m.Name, Initiative: envio.Iniciativa, HP: envio.PV, Kind: "npc",
	}); err != nil {
		return nil, err
	}
	if envio.Copias < 1 || envio.Copias > maxCopiasDeUmVerbete {
		return nil, fmt.Errorf("quantas %d está fora da faixa de 1 a %d", envio.Copias, maxCopiasDeUmVerbete)
	}

	var estado *aovivo.SessionRuntimeState
	for i := 0; i < envio.Copias; i++ {
		linha, err := st.deps.MaterializeEntry(c.R.Context(), c.User, c.CampaignID, map[string]any{
			"label": m.Name, "initiative": envio.Iniciativa, "type": "npc",
			"monsterId": m.ID, "hpCurrent": envio.PV, "hpMax": envio.PV,
		})
		if err != nil {
			return estado, err
		}
		// O parcial volta junto com o erro: quatro goblins que entraram são o
		// estado da mesa, e o `gmCommand` o transmite (ALE-155).
		if estado, err = st.deps.Sessions().AddInitiativeEntry(c.SessionID, linha); err != nil {
			return estado, err
		}
	}
	return estado, nil
}

// maxCopiasDeUmVerbete é o teto de cópias num gesto.
//
// Não é regra do livro: é o que separa "quatro goblins" de um zero a mais que
// enche a fila e o mestre tem de desfazer linha por linha. O número é o mesmo da
// SPA. A fila tem teto próprio no servidor, e ele responde por si.
const maxCopiasDeUmVerbete = 12

type envioDoVerbete struct {
	Criatura   string
	PV         int64
	Iniciativa int
	Copias     int
}

func envioDosSinais(r *http.Request) (envioDoVerbete, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var sinais struct {
		Criatura   string `json:"criatura"`
		PV         int64  `json:"pvdoverbete"`
		Iniciativa int    `json:"inidoverbete"`
		Copias     int    `json:"copiasdoverbete"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return envioDoVerbete{}, fmt.Errorf("não entendi o envio: %v", err)
	}
	return envioDoVerbete{
		Criatura: sinais.Criatura, PV: sinais.PV,
		Iniciativa: sinais.Iniciativa, Copias: sinais.Copias,
	}, nil
}

// tableGmOrRefusal resolve a mesa e exige o papel, para as rotas do painel
// que NÃO passam pelo `gmCommand` — as que só leem.
func (s Scene) tableGmOrRefusal(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return 0, 0, false
	}
	_, papel, status, err := s.deps.SessionForCaller(r.Context(), s.deps.CurrentUserID(r), campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return 0, 0, false
	}
	// O bestiário é do mestre INTEIRO, não só o mandar para a mesa: a lista diz
	// o PV e a defesa de cada bicho, e é isso que o mestre esconde quando aperta
	// o olho numa linha.
	if papel != "gm" {
		http.Error(w, "o bestiário da mesa é do mestre", http.StatusForbidden)
		return 0, 0, false
	}
	return campaignID, sessionID, true
}
