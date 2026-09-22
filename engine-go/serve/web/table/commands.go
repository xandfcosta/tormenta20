package table

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"t20engine/serve/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app"
	"t20engine/app/initiative"
	"t20engine/domain/live"
)

// OS COMANDOS DO MESTRE na Mesa.
//
// Eles NÃO reusam as rotas da API JSON: a cena tem rota própria que chama a
// MESMA regra. O que impede duas telas de divergirem não é compartilhar a rota
// — é compartilhar a regra e o store.
//
// Esconder o botão é UX; a trava é aqui.

func (s Scene) TableCommandRoutes(r chi.Router) {
	r.Post(sessionPattern+"/iniciativa/proxima-vez", s.gmCommand(
		func(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
			return st.deps.Sessions().NextTurn(c.SessionID)
		}))
	r.Post(sessionPattern+"/iniciativa/vez-anterior", s.gmCommand(
		func(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
			return st.deps.Sessions().PreviousTurn(c.SessionID)
		}))
	// O TIPO DA CENA vai no CAMINHO, como o estado da cortina: nesta superfície
	// o verbo é o caminho, e os três tipos do livro (p252) são três verbos
	// diferentes — abrir um combate e abrir uma conversa não são o mesmo gesto.
	r.Post(sessionPattern+"/cena/iniciar/{tipo}", s.gmCommand(
		func(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
			kind, err := sceneKindOf(chi.URLParam(c.R, "tipo"))
			if err != nil {
				return nil, err
			}
			return st.deps.Sessions().StartScene(c.SessionID, kind)
		}))
	r.Post(sessionPattern+"/cena/encerrar", s.sceneCommand(endsTheScene))
	r.Post(sessionPattern+"/iniciativa/por-no-mapa", s.gmCommand(bringParty))
	r.Post(sessionPattern+"/iniciativa/adicionar", s.gmCommand(addCombatant))
	// DOIS caminhos e não um `/descanso` com o escopo no corpo, que é a forma da
	// API JSON: nesta superfície o VERBO é o caminho, e misturar as duas
	// gramáticas faria a próxima pessoa ter de descobrir qual vale onde.
	r.Post(sessionPattern+"/descanso/cena", s.sceneCommand(expiresTheSceneOfTheParty))
	r.Post(sessionPattern+"/descanso/dia", s.sceneCommand(restsForTheDay))
	// O QUE O MESTRE MEXE EM CADA LINHA — mais restrito que a API JSON de
	// propósito. Lá o `assertVitalsEditableFor` deixa o jogador mexer nos vitais
	// do PRÓPRIO personagem, porque lá existe a tela do jogador que faz isso.
	// Aqui a superfície do jogador é leitura mais registrar iniciativa, e uma
	// segunda regra de escrita seria uma porta que nenhuma tela usa.
	r.Route(sessionPattern+"/iniciativa/{entryId}", func(r chi.Router) {
		r.Post("/vitais/{pool}/ferir/{step}", s.gmCommand(moveVitals(-1)))
		r.Post("/vitais/{pool}/curar/{step}", s.gmCommand(moveVitals(+1)))
		r.Post("/vitais/{pool}/oculto", s.gmCommand(toggleEye))
		r.Post("/editar", s.gmCommand(editaOCombatente))
		r.Post("/remover", s.gmCommand(tiraDaFila))
	})
}

// addCombatant é o capanga digitado na hora: nome, iniciativa, PV e se é
// PC ou NPC.
//
// A VALIDAÇÃO é do `live` e não daqui: limite escrito como atributo de campo é
// UI e não trava, e quem posta na mão passa por cima dele.
//
// Quem MONTA a linha é o `Roster.Entry`, que é o caminho único dos três pedidos
// — este, o NPC do elenco e o verbete do bestiário. Escrever a montagem aqui
// seria a segunda cópia da mesma regra.
func addCombatant(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	novo, err := signalsCombatant(c.R)
	if err != nil {
		return nil, err
	}
	if err := live.ValidateCombatantDraft(novo); err != nil {
		return nil, err
	}
	rolled := int64(novo.Initiative)
	requested := initiative.EntryRequest{
		Label: strings.TrimSpace(novo.Label), Initiative: &rolled, Kind: novo.Kind,
	}
	// PV ZERO fica de fora em vez de virar 0/0: "sem vida registrada" é a
	// ausência do campo, e uma barra 0/0 diria que o capanga já está morto.
	if novo.HP > 0 {
		pv := int64(novo.HP)
		requested.HpCurrent, requested.HpMax = &pv, &pv
	}
	row, err := st.queue.Roster().Entry(c.R.Context(), app.Caller{ID: c.User}, c.CampaignID, requested)
	if err != nil {
		return nil, err
	}
	state, err := st.deps.Sessions().AddInitiativeEntry(c.SessionID, row)
	if err != nil {
		return nil, err
	}
	// O formulário volta ao zero: sem isto o nome fica no campo e o clique
	// seguinte acrescenta o MESMO capanga de novo — e no meio de um combate
	// ninguém confere a fila antes de clicar. Volta para NPC porque é o caso
	// comum; o PC digitado à mão é a exceção.
	c.Signals["new_name"] = ""
	c.Signals["new_initiative"] = 10
	c.Signals["new_hp"] = 0
	c.Signals["new_type"] = "npc"
	return state, nil
}

// signalsCombatant lê o formulário da página.
//
// TODOS OS NOMES EM `snake_case`, e isso é obrigatório e não estilo: eles são
// chaves de `data-bind:`, e nome de atributo é minusculado pelo analisador de
// HTML. Caixa alta na chave liga um sinal NOVO e deixa o declarado intocado — o
// fio leva os dois e o servidor lê o errado, sem erro em lugar nenhum.
func signalsCombatant(r *http.Request) (live.CombatantDraft, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var signals struct {
		Name       string `json:"new_name"`
		Initiative int    `json:"new_initiative"`
		PV         int64  `json:"new_hp"`
		Kind       string `json:"new_type"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return live.CombatantDraft{}, fmt.Errorf("não entendi o combatente enviado: %v", err)
	}
	return live.CombatantDraft{
		Label: signals.Name, Initiative: signals.Initiative, HP: signals.PV, Kind: signals.Kind,
	}, nil
}

// moveVitals é o dano e a cura de UMA linha, e o PASSO vem do CAMINHO.
//
// Não é um número que a página manda: sinal é a superfície onde a página e o
// servidor discordam em silêncio, e um passo em sinal teria de ser validado aqui
// de qualquer jeito. No caminho, o que não é 1 nem 5 não casa rota nenhuma.
//
// O sinal do delta é do FECHAMENTO e não de uma comparação de string: ferir e
// curar são rotas diferentes, então não há o que comparar nem como escrever a
// terceira palavra que não existe.
//
// Quem sabe somar é o store: com personagem atrás da linha quem manda é a FICHA
// (o dano drena PV temporários) e a entrada espelha o resultado. Não há uma
// segunda conta aqui.
func moveVitals(sign int64) func(Scene, commandCtx) (*live.SessionRuntimeState, error) {
	return func(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
		raw := chi.URLParam(c.R, "step")
		step, ok := vitalSteps[raw]
		if !ok {
			return nil, fmt.Errorf("passo %q não existe; a tela oferece 1 (clique) e 5 (Shift+clique)", raw)
		}
		delta := sign * step
		hp, mp, ok := poolDeltas(chi.URLParam(c.R, "pool"), delta)
		if !ok {
			return nil, fmt.Errorf("pool %q não existe; a fila mexe em 'hp' e em 'mp'", chi.URLParam(c.R, "pool"))
		}
		entryID := chi.URLParam(c.R, "entryId")
		state, err := st.deps.Sessions().DeltaVitals(c.SessionID, entryID, hp, mp)
		// QUANDO HÁ FICHA ATRÁS DA LINHA, quem levou o dano foi o PERSONAGEM e
		// não o rastreador (ver `DeltaVitals`) — então a ficha de quem está na
		// mesa mudou, e a tela dele precisa saber. NPC não tem ficha: ali o
		// `CharacterIDOf` devolve nulo e não há quem avisar.
		if err == nil {
			if charID := st.deps.Sessions().CharacterIDOf(c.SessionID, entryID); charID != nil {
				st.deps.CharacterChanged(*charID)
			}
		}
		return state, err
	}
}

// vitalSteps são os DOIS que a tela oferece: o clique e o Shift+clique.
var vitalSteps = map[string]int64{"1": 1, "5": 5}

// poolDeltas manda o passo para o pool que a URL nomeia — o outro vai nulo,
// que é como o `DeltaVitals` diz "não mexe".
func poolDeltas(pool string, delta int64) (hp, mp *int64, ok bool) {
	switch pool {
	case "hp":
		return &delta, nil, true
	case "mp":
		return nil, &delta, true
	}
	return nil, nil, false
}

// toggleEye esconde e revela os PV de uma linha para os JOGADORES.
//
// O servidor lê o estado atual e o INVERTE, em vez de a página mandar o valor
// que ela quer. Dois mestres na mesma mesa — ou a mesma aba com o remendo
// atrasado — mandariam "esconder" duas vezes, e a segunda desfaria a primeira
// sem ninguém ter pedido. Quem sabe o estado é quem o guarda.
func toggleEye(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	entryID := chi.URLParam(c.R, "entryId")
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	i := live.FindEntryIndex(state, entryID)
	if i < 0 {
		return nil, fmt.Errorf("combatente %q não está na fila", entryID)
	}
	// A ALTERNÂNCIA parte do que a MESA VÊ hoje, e não do ponteiro: cada pool tem
	// um padrão próprio, e nulo NÃO quer dizer "visível" em todos. Invertendo o
	// ponteiro, o PV do NPC — que já nasce escondido — receberia "esconder" sobre
	// uma linha escondida, e o mestre clicaria no olho sem a tela mudar nada. É o
	// defeito mais difícil de reportar: o botão parece morto.
	e := state.Initiative[i]
	choice, visibleByDefault := e.MpHidden, false
	pool := chi.URLParam(c.R, "pool")
	if pool == "hp" {
		choice, visibleByDefault = e.HpHidden, e.Type == "character"
	} else if pool != "mp" {
		return nil, fmt.Errorf("pool %q não existe; a fila esconde 'hp' e 'mp'", pool)
	}
	visible := visibleByDefault
	if choice != nil {
		visible = !*choice
	}
	hidden := visible
	patch := live.EntryPatch{MpHidden: &hidden}
	if pool == "hp" {
		patch = live.EntryPatch{HpHidden: &hidden}
	}
	return st.deps.Sessions().UpdateInitiativeEntry(c.SessionID, entryID, patch)
}

// editaOCombatente corrige a iniciativa e o PV de quem já está na fila. "Pôr no
// mapa" entra com iniciativa 0, e sem esta porta a única saída seria remover e
// acrescentar de novo — perdendo PV e condições no caminho.
//
// QUEM DECIDE SE HÁ PV PARA EDITAR é o servidor, olhando a linha, e não um sinal
// que a página mande junto: uma tela defasada diria "tem" sobre um combatente
// que acabou de perder a barra, e a escrita inventaria um pool. É a mesma
// escolha do olho, pelo mesmo motivo.
//
// A ordem também importa: a iniciativa primeiro, porque ela REORDENA a fila, e
// os vitais depois, pelo id — que não muda com a reordenação.
func editaOCombatente(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	entryID := chi.URLParam(c.R, "entryId")
	edit, err := edicaoDosSinais(c.R)
	if err != nil {
		return nil, err
	}
	if err := live.ValidateInitiative(edit.Initiative); err != nil {
		return nil, err
	}
	before, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	i := live.FindEntryIndex(before, entryID)
	if i < 0 {
		return nil, fmt.Errorf("combatente %q não está na fila", entryID)
	}
	hasVitals := before.Initiative[i].HpMax != nil

	state, err := st.deps.Sessions().UpdateInitiativeEntry(c.SessionID, entryID,
		live.EntryPatch{Initiative: &edit.Initiative})
	if err != nil {
		return nil, err
	}
	if !hasVitals {
		return state, nil
	}
	// Com personagem atrás da linha o `PatchVitals` escreve na FICHA e espelha,
	// como o delta faz. Quem prende o valor ao teto é ele, não uma conta aqui.
	return st.deps.Sessions().PatchVitals(c.SessionID, entryID, &edit.PV, nil)
}

// edicaoDosSinais lê o diálogo de editar. Nomes minúsculos pelo mesmo motivo de
// sempre: são chaves de `data-bind:`, e nome de atributo é minusculado.
func edicaoDosSinais(r *http.Request) (struct {
	Initiative int
	PV         int64
}, error) {
	var outside struct {
		Initiative int
		PV         int64
	}
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var signals struct {
		Initiative int   `json:"edit_initiative"`
		PV         int64 `json:"edit_hp"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return outside, fmt.Errorf("não entendi a edição enviada: %v", err)
	}
	outside.Initiative, outside.PV = signals.Initiative, signals.PV
	return outside, nil
}

// tiraDaFila remove o combatente. Sem confirmação: o gesto é do meio do
// combate, e a fila é remontável — o que não é remontável (encerrar a cena) é
// que ganhou dois verbos distintos em vez de um interruptor.
func tiraDaFila(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.deps.Sessions().RemoveInitiativeEntry(c.SessionID, chi.URLParam(c.R, "entryId"))
}

// restParty é a RECUPERAÇÃO (T20 p106): devolve PV e PM ao grupo inteiro.
//
// Os dois escopos dividem o corpo porque só diferem na qualidade, que só o de
// dia usa. Duas funções seriam duas chances de uma esquecer o aviso às fichas.
//
// O aviso é obrigatório e não é o `session-state`: o que muda no descanso é a
// FICHA, e ela não está no estado da fila. Sem o `session-rest`, quem está com a
// ficha aberta continua vendo o PV de antes até recarregar.
func expiresTheSceneOfTheParty(
	st Scene, r *http.Request, who app.Caller, campaignID, sessionID int64,
) (*live.SessionRuntimeState, error) {
	done, total, err := st.party.ExpireScene(r.Context(), who, campaignID, sessionID)
	if err != nil {
		return nil, err
	}
	return st.announcesTheRest(sessionID, "scene", "normal", done, total)
}

func restsForTheDay(
	st Scene, r *http.Request, who app.Caller, campaignID, sessionID int64,
) (*live.SessionRuntimeState, error) {
	quality, err := restQuality(r)
	if err != nil {
		return nil, err
	}
	done, total, err := st.party.RestForTheDay(r.Context(), who, campaignID, sessionID, quality)
	if err != nil {
		return nil, err
	}
	return st.announcesTheRest(sessionID, "day", quality, done, total)
}

// announcesTheRest avisa as fichas e devolve o estado com a contagem.
//
// O AVISO é obrigatório e não é o `session-state`: o que muda no descanso é a
// FICHA, e ela não está no estado da fila. Sem o `session-rest`, quem está com a
// ficha aberta continua vendo o PV de antes até recarregar.
func (s Scene) announcesTheRest(
	sessionID int64, scope, quality string, done, total int,
) (*live.SessionRuntimeState, error) {
	s.deps.SSE().Emit(sessionID, "", "session-rest", map[string]any{
		"sessionId": sessionID, "scope": scope, "condition": quality,
	})
	state, err := s.deps.Sessions().State(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}
	// O PARCIAL é contado e DITO: descartar a contagem faria o mestre ler
	// "descansou" com duas de cinco fichas de fora. Volta como recusa porque é o
	// caminho que acende a frase — e "3 de 5" é o que ele precisa ver para saber
	// que tem de olhar as outras duas.
	if done < total {
		return state, fmt.Errorf("%d de %d fichas descansaram; as outras %d falharam",
			done, total, total-done)
	}
	return state, nil
}

// restQualities são as quatro do livro (T20 p106), e a lista existe aqui para
// RECUSAR o que não é uma delas: a conta do livro cai em "normal"
// quando não reconhece a palavra, então um sinal adulterado faria o grupo
// descansar em "normal" com o mestre tendo pedido "luxuosa", e ninguém veria a
// diferença.
var restQualities = map[string]bool{"ruim": true, "normal": true, "confortavel": true, "luxuosa": true}

// restQuality lê o sinal da página.
//
// Lê ANTES do `NewSSE`, obrigatoriamente: o SDK assume a resposta e fecha o
// corpo do pedido, e um `ReadSignals` depois dele encontra o corpo fechado. Quem
// garante a ordem é o `gmCommand`, que chama a mutação primeiro. A inversão
// passa VERDE em teste de handler e só falha no navegador.
func restQuality(r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // o mesmo teto de 1 MB do `httpio.DecodeJSON`
	var signals struct {
		Quality string `json:"rest_quality"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return "", fmt.Errorf("não entendi a qualidade do descanso: %v", err)
	}
	if !restQualities[signals.Quality] {
		return "", fmt.Errorf("qualidade %q não existe; o livro tem ruim, normal, confortavel e luxuosa (p106)", signals.Quality)
	}
	return signals.Quality, nil
}

// bringParty põe na fila cada personagem do grupo que ainda não está lá.
//
// É idempotente — o `populateParty` pula quem já está —, e é por isso que o
// botão continua clicável em vez de apagar depois do primeiro uso: o mestre que
// aceitou um jogador atrasado clica de novo e leva só o que faltava.
//
// QUEM é o grupo é do `PartyCombatants` e não daqui: são TODOS os membros da
// campanha, inclusive o PC que o mestre também joga. Uma segunda opinião sobre
// quem é o grupo faria duas superfícies responderem diferente à mesma pergunta.
func bringParty(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	combatants, err := st.queue.Roster().PartyCombatants(c.R.Context(), c.CampaignID)
	if err != nil {
		return nil, errors.New("não deu para carregar o grupo desta campanha")
	}
	// O erro vem JUNTO com o estado parcial de propósito: pôr quatro dos cinco e
	// tropeçar no quinto deixa a mesa com quatro combatentes novos, e é esse o
	// estado que as outras telas precisam receber.
	state, err := st.queue.PopulateParty(c.SessionID, combatants)
	if state == nil {
		fresh, stateErr := st.deps.Sessions().State(c.R.Context(), c.SessionID)
		if stateErr != nil {
			return nil, stateErr
		}
		state = fresh
	}
	return state, err
}

// commandCtx é o que a mutação de um comando do mestre recebe. É um struct e
// não o id da sessão solto: comando que precisa da CAMPANHA — encerrar a cena,
// que expira as fichas do grupo — com uma assinatura estreita acaba chamando o
// helper que não precisa dela e faz menos.
type commandCtx struct {
	R    *http.Request
	User int64
	// Role é "gm" ou "player", resolvido contra o banco. Ele viaja porque há
	// gesto que os DOIS emitem e a regra separa — o ataque é proposto por quem
	// joga e confirmado por quem mestra.
	Role       string
	CampaignID int64
	SessionID  int64
	// BoardID é a ABA em que este comando age, e ela é a aba que QUEM CLICOU
	// está olhando.
	//
	// Ela não vem do caminho nem de um sinal da página: o gateway a resolve no
	// servidor, pelo `chosenTabs`. A afirmação que isso faz é de domínio e é
	// forte — **não se pinta um tabuleiro que não se está olhando** —, e uma aba
	// no caminho deixaria essa porta aberta sem nenhum gesto que a abrisse.
	//
	// Vazia significa a aba PADRÃO — quem entrou na sessão e ainda não escolheu.
	BoardID string
	// Signals é o que a cena recebe de volta ALÉM do HTML, e a mutação escreve
	// nele quando quer mexer no estado do CLIENTE.
	//
	// Hoje só o formulário de acrescentar usa, e o motivo dele é o que justifica
	// o campo: ele se limpa DEPOIS de o servidor aceitar. Limpar no clique
	// custaria o que a pessoa digitou toda vez que a validação recusasse — e a
	// recusa mais comum é o nome, que é o campo mais caro de redigitar no meio
	// de um combate.
	//
	// QUEM GARANTE que a recusa não limpa nada é a ORDEM, e não um descarte:
	// a mutação só escreve neste mapa depois de a sua própria escrita ter dado
	// certo, então numa recusa ele chega vazio.
	Signals map[string]any
}

// endScene é o gesto INTEIRO, e por isso é função nomeada e não um literal na
// lista de rotas: ela faz duas coisas que os outros comandos não fazem.
//
// A primeira é NÃO chamar o `EndScene` do store direto — ele só mexe no
// rastreador, e a fila zeraria na tela com a bênção de duração "cena" viva na
// FICHA. O livro não deixa margem: "a habilidade dura uma cena inteira,
// encerrando-se quando esse momento da história acaba" (p227). O
// `Party.EndScene` do `app/rest` é o caminho único que expira as fichas do
// grupo ANTES de desligar a cena, e a ordem é dele — não desta função.
//
// A segunda é o aviso: as fichas não estão no estado do rastreador, então sem o
// `session-rest` o efeito morto e o "usado 1/cena" ficam na tela até alguém
// recarregar.
func endsTheScene(
	st Scene, r *http.Request, who app.Caller, campaignID, sessionID int64,
) (*live.SessionRuntimeState, error) {
	state, err := st.party.EndScene(r.Context(), who, campaignID, sessionID)
	if err != nil {
		return nil, err
	}
	st.deps.SSE().Emit(sessionID, "", "session-rest", map[string]any{
		"sessionId": sessionID, "scope": "scene",
	})
	return state, nil
}

// gmCommand é o caminho único dos comandos do mestre.
//
// Eles só diferem na MUTAÇÃO, e o resto — resolver a mesa, exigir o papel,
// publicar o estado, redesenhar a cena — é idêntico. Sem o parâmetro seriam N
// cópias, e é numa delas que alguém esquece de publicar e a mesa fica vendo o
// turno velho.
func (s Scene) gmCommand(
	mutate func(Scene, commandCtx) (*live.SessionRuntimeState, error),
) http.HandlerFunc {
	return s.stateCommand(mutate, true)
}

// tableStateCommand é o irmão do `gmCommand` para o que o JOGADOR também faz —
// a mesma divisão que o `tableCommand` faz no tabuleiro, e pela mesma razão.
//
// Ele NÃO exige papel porque a recusa é da REGRA, e a regra escreve a frase
// certa: "só o mestre põe o dano na ficha" diz o que aconteceu, e um 403 diria
// "proibido" a quem está fazendo exatamente o que o desenho prevê — propor.
func (s Scene) tableStateCommand(
	mutate func(Scene, commandCtx) (*live.SessionRuntimeState, error),
) http.HandlerFunc {
	return s.stateCommand(mutate, false)
}

func (s Scene) stateCommand(
	mutate func(Scene, commandCtx) (*live.SessionRuntimeState, error),
	gmOnly bool,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, sessionID, ok := tableParams(w, r)
		if !ok {
			return
		}
		userID := s.deps.CurrentUserID(r)
		_, role, err := s.access.Session(r.Context(), app.Caller{ID: userID}, campaignID, sessionID)
		status := statusOf(err)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		// A trava é aqui e não na tela: quem postar na mão leva 403, e o botão
		// escondido é só cortesia para quem não pode.
		if gmOnly && role != "gm" {
			http.Error(w, "só o mestre comanda a mesa", http.StatusForbidden)
			return
		}

		signals := map[string]any{}
		state, err := mutate(s, commandCtx{
			R: r, User: userID, Role: role, CampaignID: campaignID, SessionID: sessionID, Signals: signals,
		})
		// O que POUSOU se transmite mesmo quando a chamada devolveu erro, e o
		// `bringParty` é quem o exige: ele põe quatro dos cinco e tropeça no
		// quinto, e os quatro já são o estado da mesa. Segurar a transmissão
		// porque houve erro deixaria as outras telas com a fila de antes —
		// best-effort é sobre continuar apesar da falha, não sobre escondê-la.
		if state != nil {
			s.deps.PublishSessionState(sessionID, state)
		}
		s.respondGm(w, r, userID, campaignID, sessionID, err, signals)
	}
}

// respondGm devolve a cena remendada E a frase da recusa — as duas sempre, e as
// duas por SSE.
//
// A recusa NÃO pode ser um `http.Error`: o Datastar não desenha corpo de
// resposta 4xx, então ela não chegaria a lugar nenhum e o mestre clicaria
// olhando para uma tela que não muda.
//
// O 403 é a exceção e continua sendo `http.Error`: ele é para quem posta na mão,
// e a tela de quem não é mestre nunca teve o botão.
//
// A cena é remendada NA HORA em vez de esperar o próximo tique do stream, que
// avisa-e-relê e calaria pelo hash. E o remendo vale também na recusa:
// redesenhar mostra que a cena continua ABERTA, que é a verdade que o mestre
// precisa ver ao lado da frase.
func (s Scene) respondGm(
	w http.ResponseWriter, r *http.Request,
	userID int64, campaignID, sessionID int64, refusal error, signals map[string]any,
	onlyRegions ...string,
) {
	sse := datastar.NewSSE(w, r)
	if view, _, err := s.LoadView(r.Context(), userID, campaignID, sessionID); err == nil {
		// Por PADRÃO manda TODAS as regiões e não só as que mudaram, ao contrário
		// do stream: aqui não há digital anterior para comparar — este caminho
		// responde a um pedido, não mantém uma conexão. Vale porque quem recebe
		// acabou de CLICAR.
		//
		// O GESTO CONTÍNUO é a exceção que criou o `onlyRegions`: no arrasto do
		// pincel cada casa cruzada devolveria a Mesa inteira, 353 KB por casa.
		//
		// Falhar ao redesenhar não desfaz a mutação, que já aconteceu e já foi
		// transmitida; o stream corrige no próximo tique. Por isso é best-effort e
		// a frase sai de qualquer jeito.
		for _, region := range TableRegions(view) {
			if !pedidaOuTodas(region.ID, onlyRegions) {
				continue
			}
			if fragment, err := ui.RenderFragment(r.Context(), region.No); err == nil {
				_ = sse.PatchElements(fragment)
			}
		}
	}
	sentence := ""
	if refusal != nil {
		sentence = refusal.Error()
	}
	// Sai nos DOIS caminhos: no da recusa para acender a frase, e no do acerto
	// para APAGAR a anterior. Um sinal que só se escreve quando dá errado deixa
	// a recusa de dois cliques atrás acesa sobre um comando que funcionou.
	signals["command_error"] = sentence
	_ = sse.MarshalAndPatchSignals(signals)
}

// pedidaOuTodas: lista vazia quer dizer "a Mesa inteira", que é o padrão de
// quase todo comando.
func pedidaOuTodas(id string, requested []string) bool {
	if len(requested) == 0 {
		return true
	}
	return slices.Contains(requested, id)
}
