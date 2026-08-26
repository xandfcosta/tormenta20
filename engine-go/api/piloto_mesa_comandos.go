package api

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/aovivo"
)

// OS COMANDOS DO MESTRE na Mesa em Datastar (ALE-265).
//
// Eles NÃO reusam as rotas da API JSON, e a escolha é a mesma das catorze
// fatias: a cena tem rota própria que chama a MESMA regra. O que impede as duas
// telas de divergirem não é compartilhar a rota — é compartilhar a regra e o
// store. A ALE-122 aconteceu com dois transportes escrevendo em dois LUGARES,
// não em dois caminhos.
//
// A autorização é a que já existe: o `sessionForCaller` resolve o papel pelo
// mesmo caminho que a API usa, e papel desconhecido cai em jogador. Esconder o
// botão é UX; a trava é aqui.

func (s *Server) rotasDosComandosDaMesa(r chi.Router) {
	r.Post("/mesa/{campaignId}/{sessionId}/initiative/next-turn", s.comandoDoMestre(
		func(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.NextTurn(c.SessionID)
		}))
	r.Post("/mesa/{campaignId}/{sessionId}/initiative/previous-turn", s.comandoDoMestre(
		func(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.PreviousTurn(c.SessionID)
		}))
	r.Post("/mesa/{campaignId}/{sessionId}/scene/start", s.comandoDoMestre(
		func(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.StartScene(c.SessionID)
		}))
	r.Post("/mesa/{campaignId}/{sessionId}/scene/end", s.comandoDoMestre(encerraACena))
	r.Post("/mesa/{campaignId}/{sessionId}/initiative/populate", s.comandoDoMestre(trazOGrupo))
	r.Post("/mesa/{campaignId}/{sessionId}/initiative/add", s.comandoDoMestre(acrescentaCombatente))
	// DOIS caminhos e não um `/rest` com o escopo no corpo, que é a forma da API
	// JSON: aqui o VERBO é o caminho, como em `scene/start` e `scene/end`. A
	// gramática desta superfície já foi escolhida, e misturar as duas faria a
	// próxima pessoa ter de descobrir qual vale onde.
	r.Post("/mesa/{campaignId}/{sessionId}/rest/scene", s.comandoDoMestre(descansaOGrupo("scene")))
	r.Post("/mesa/{campaignId}/{sessionId}/rest/day", s.comandoDoMestre(descansaOGrupo("day")))
	// O QUE O MESTRE MEXE EM CADA LINHA. O `entryId` vem do caminho como os
	// outros dois ids, e a autorização é a mesma dos comandos da mesa: o
	// `comandoDoMestre` já barra quem não é mestre.
	//
	// Mais restrito que a API JSON de propósito. Lá o `assertVitalsEditableFor`
	// deixa o jogador mexer nos vitais do PRÓPRIO personagem, porque lá existe a
	// tela do jogador que faz isso. Aqui a superfície do jogador é leitura mais
	// registrar iniciativa (ALE-213), e uma segunda regra de escrita seria uma
	// porta que nenhuma tela usa.
	r.Route("/mesa/{campaignId}/{sessionId}/initiative/{entryId}", func(r chi.Router) {
		r.Post("/vitals/harm/{passo}", s.comandoDoMestre(mexeNosVitais(-1)))
		r.Post("/vitals/heal/{passo}", s.comandoDoMestre(mexeNosVitais(+1)))
		r.Post("/vitals/hidden", s.comandoDoMestre(alternaOOlho))
		r.Post("/edit", s.comandoDoMestre(editaOCombatente))
		r.Post("/remove", s.comandoDoMestre(tiraDaFila))
	})
}

// acrescentaCombatente é o capanga digitado na hora: nome, iniciativa, PV e se é
// PC ou NPC.
//
// A VALIDAÇÃO é do `aovivo` e não daqui, e essa é a extração de sempre: os
// limites viviam como atributos dos campos do formulário da SPA, que é UI e não
// trava — quem postasse na mão passava por cima dos quatro. Com eles no pacote
// do estado, as duas telas param de poder discordar sobre o que é um combatente
// aceitável.
//
// Quem MONTA a linha continua sendo o `materializeEntry`, que é o caminho que a
// API já usa: sem `characterId` ele cai no NPC, e o PV só entra quando foi
// digitado. Escrever a montagem aqui seria a segunda cópia da mesma regra, que é
// como a ALE-122 começou.
func acrescentaCombatente(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	novo, err := combatenteDosSinais(c.R)
	if err != nil {
		return nil, err
	}
	if err := aovivo.ValidaCombatenteNovo(novo); err != nil {
		return nil, err
	}
	entrada := map[string]any{
		"label":      strings.TrimSpace(novo.Rotulo),
		"initiative": novo.Iniciativa,
		"type":       novo.Tipo,
	}
	// PV ZERO fica de fora em vez de virar 0/0: "sem vida registrada" é a
	// ausência do campo, e uma barra 0/0 diria que o capanga já está morto.
	if novo.PV > 0 {
		entrada["hpCurrent"] = novo.PV
		entrada["hpMax"] = novo.PV
	}
	linha, err := st.materializeEntry(c.R.Context(), c.User.ID, c.CampaignID, entrada)
	if err != nil {
		return nil, err
	}
	estado, err := st.sessions.AddInitiativeEntry(c.SessionID, linha)
	if err != nil {
		return nil, err
	}
	// O formulário volta ao zero, como na SPA: sem isto o nome fica no campo e o
	// clique seguinte acrescenta o MESMO capanga de novo — e no meio de um
	// combate ninguém confere a fila antes de clicar. Volta para NPC porque é o
	// caso comum; o PC digitado à mão é a exceção.
	c.Sinais["novonome"] = ""
	c.Sinais["novainiciativa"] = 10
	c.Sinais["novopv"] = 0
	c.Sinais["novotipo"] = "npc"
	return estado, nil
}

// combatenteDosSinais lê o formulário da página.
//
// TODOS OS NOMES SÃO MINÚSCULOS, e isso é obrigatório e não estilo: eles são
// chaves de `data-bind:`, e nome de atributo é minusculado pelo analisador de
// HTML. Um `data-bind:novoNome` liga um sinal `novonome` e deixa o declarado
// intocado — o fio leva os dois e o servidor lê o errado. Foi exatamente isso
// que aconteceu com a qualidade do descanso, e o navegador foi a única
// testemunha.
func combatenteDosSinais(r *http.Request) (aovivo.CombatenteNovo, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var sinais struct {
		Nome       string `json:"novonome"`
		Iniciativa int    `json:"novainiciativa"`
		PV         int64  `json:"novopv"`
		Tipo       string `json:"novotipo"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return aovivo.CombatenteNovo{}, fmt.Errorf("não entendi o combatente enviado: %v", err)
	}
	return aovivo.CombatenteNovo{
		Rotulo: sinais.Nome, Iniciativa: sinais.Iniciativa, PV: sinais.PV, Tipo: sinais.Tipo,
	}, nil
}

// mexeNosVitais é o dano e a cura de UMA linha, e o PASSO vem do CAMINHO.
//
// Não é um número que a página manda, e a escolha é a lição desta fatia: sinal é
// a superfície onde a página e o servidor discordam em silêncio — o
// `qualidadedodescanso` chegou a viajar com DOIS nomes no fio, e o servidor leu
// o que ninguém tinha tocado. Um passo em sinal teria de ser validado aqui de
// qualquer jeito; no caminho, o que não é 1 nem 5 não casa rota nenhuma.
//
// O sinal do delta é do FECHAMENTO e não de uma comparação de string: "harm" e
// "heal" são rotas diferentes, então não há o que comparar nem como escrever a
// terceira palavra que não existe.
//
// Quem sabe somar é o store: com personagem atrás da linha quem manda é a FICHA
// (o dano drena PV temporários) e a entrada espelha o resultado (ALE-122). O
// piloto não tem uma segunda conta.
func mexeNosVitais(sinal int64) func(*Server, mesaComando) (*aovivo.SessionRuntimeState, error) {
	return func(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
		bruto := chi.URLParam(c.R, "passo")
		passo, ok := passosDoVital[bruto]
		if !ok {
			return nil, fmt.Errorf("passo %q não existe; a tela oferece 1 (clique) e 5 (Shift+clique)", bruto)
		}
		delta := sinal * passo
		return st.sessions.DeltaVitals(c.SessionID, chi.URLParam(c.R, "entryId"), &delta, nil)
	}
}

// passosDoVital são os DOIS que a tela oferece: o clique e o Shift+clique.
//
// Espelham o `STEP`/`SHIFT_STEP` da SPA, e serem os mesmos números importa pelo
// motivo de sempre nesta migração — duas escadas diferentes fariam as duas telas
// chamarem de "um golpe" coisas diferentes.
var passosDoVital = map[string]int64{"1": 1, "5": 5}

// alternaOOlho esconde e revela os PV de uma linha para os JOGADORES.
//
// O servidor lê o estado atual e o INVERTE, em vez de a página mandar o valor
// que ela quer. Dois mestres na mesma mesa — ou a mesma aba com o remendo
// atrasado — mandariam "esconder" duas vezes, e a segunda desfaria a primeira
// sem ninguém ter pedido. Quem sabe o estado é quem o guarda.
func alternaOOlho(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	entryID := chi.URLParam(c.R, "entryId")
	estado := st.sessions.GetState(c.SessionID)
	i := aovivo.FindEntryIndex(estado, entryID)
	if i < 0 {
		return nil, fmt.Errorf("combatente %q não está na fila", entryID)
	}
	// Nil é "nunca escondido", que é o estado de nascença de toda linha — e o
	// `DerefOr` da casa só serve para int64.
	atual := estado.Initiative[i].HpHidden
	oculto := atual == nil || !*atual
	return st.sessions.UpdateInitiativeEntry(c.SessionID, entryID, aovivo.EntryPatch{HpHidden: &oculto})
}

// editaOCombatente corrige a iniciativa e o PV de quem já está na fila.
//
// A iniciativa é o gesto que a ALE-122 nomeou e deixou sem saída: "Adicionar
// grupo" entra com 0 e o mestre não tinha como consertar, então a única saída
// era remover e acrescentar de novo — perdendo PV e condições no caminho.
//
// QUEM DECIDE SE HÁ PV PARA EDITAR é o servidor, olhando a linha, e não um sinal
// que a página mande junto: uma tela defasada diria "tem" sobre um combatente
// que acabou de perder a barra, e a escrita inventaria um pool. É a mesma
// escolha do olho, pelo mesmo motivo.
//
// A ordem também importa: a iniciativa primeiro, porque ela REORDENA a fila, e
// os vitais depois, pelo id — que não muda com a reordenação.
func editaOCombatente(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	entryID := chi.URLParam(c.R, "entryId")
	edicao, err := edicaoDosSinais(c.R)
	if err != nil {
		return nil, err
	}
	if err := aovivo.ValidaIniciativa(edicao.Iniciativa); err != nil {
		return nil, err
	}
	antes := st.sessions.GetState(c.SessionID)
	i := aovivo.FindEntryIndex(antes, entryID)
	if i < 0 {
		return nil, fmt.Errorf("combatente %q não está na fila", entryID)
	}
	temVitais := antes.Initiative[i].HpMax != nil

	estado, err := st.sessions.UpdateInitiativeEntry(c.SessionID, entryID,
		aovivo.EntryPatch{Initiative: &edicao.Iniciativa})
	if err != nil {
		return nil, err
	}
	if !temVitais {
		return estado, nil
	}
	// O `PatchVitals` é o caminho da casa e vale por si: com personagem atrás da
	// linha ele escreve na FICHA e espelha, como o delta faz (ALE-122). Quem
	// prende o valor ao teto é ele, não uma conta escrita aqui.
	return st.sessions.PatchVitals(c.SessionID, entryID, &edicao.PV, nil)
}

// edicaoDosSinais lê o diálogo de editar. Nomes minúsculos pelo mesmo motivo de
// sempre: são chaves de `data-bind:`, e nome de atributo é minusculado.
func edicaoDosSinais(r *http.Request) (struct {
	Iniciativa int
	PV         int64
}, error) {
	var fora struct {
		Iniciativa int
		PV         int64
	}
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var sinais struct {
		Iniciativa int   `json:"edicaoiniciativa"`
		PV         int64 `json:"edicaopv"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return fora, fmt.Errorf("não entendi a edição enviada: %v", err)
	}
	fora.Iniciativa, fora.PV = sinais.Iniciativa, sinais.PV
	return fora, nil
}

// tiraDaFila remove o combatente. Sem confirmação, como na SPA: o gesto é do
// meio do combate, e a fila é remontável — o que não é remontável (encerrar a
// cena) é que ganhou dois verbos distintos em vez de um interruptor.
func tiraDaFila(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	return st.sessions.RemoveInitiativeEntry(c.SessionID, chi.URLParam(c.R, "entryId"))
}

// descansaOGrupo é a RECUPERAÇÃO (T20 p105): devolve PV e PM ao grupo inteiro.
//
// Os dois escopos dividem o corpo porque só diferem em duas coisas — a
// qualidade, que só o de dia usa, e o que o `restParty` faz lá dentro. Duas
// funções seriam duas chances de uma esquecer o aviso às fichas.
//
// O aviso é obrigatório e não é o `session-state`: o que muda no descanso é a
// FICHA, e ela não está no estado da fila. Sem o `session-rest`, quem estivesse
// com a ficha aberta na SPA continuaria vendo o PV de antes até recarregar.
func descansaOGrupo(escopo string) func(*Server, mesaComando) (*aovivo.SessionRuntimeState, error) {
	return func(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
		qualidade := "normal"
		if escopo == "day" {
			lida, err := qualidadeDoDescanso(c.R)
			if err != nil {
				return nil, err
			}
			qualidade = lida
		}
		feitos, total, err := st.restParty(c.User, c.CampaignID, c.SessionID, escopo, qualidade)
		if err != nil {
			return nil, errors.New("não deu para carregar o grupo desta campanha")
		}
		st.sse.Emit(c.SessionID, "", "session-rest", map[string]any{
			"sessionId": c.SessionID, "scope": escopo, "condition": qualidade,
		})
		estado := st.sessions.GetState(c.SessionID)
		// O PARCIAL é contado e DITO, que é a lição da ALE-155: antes o
		// encerrar-cena era `_, _ =` e o mestre lia "descansou" enquanto duas de
		// cinco fichas não tinham descansado. Volta como recusa porque é o
		// caminho que acende a frase — e "3 de 5" é exatamente o que ele precisa
		// ver para saber que tem de olhar as outras duas.
		if feitos < total {
			return estado, fmt.Errorf("%d de %d fichas descansaram; as outras %d falharam", feitos, total, total-feitos)
		}
		return estado, nil
	}
}

// qualidadesDoDescanso são as quatro do livro (T20 p105), e a lista existe aqui
// para RECUSAR o que não é uma delas.
//
// O `restMultiplier` do motor cai em "normal" quando não reconhece a palavra, e
// para o piloto isso não serve: um sinal adulterado faria o grupo descansar em
// "normal" enquanto o mestre pediu "luxuosa", e ninguém veria a diferença — um
// número plausível no lugar do certo é o desfecho que esta migração mais paga
// para evitar.
var qualidadesDoDescanso = map[string]bool{"ruim": true, "normal": true, "confortavel": true, "luxuosa": true}

// qualidadeDoDescanso lê o sinal da página.
//
// Lê ANTES do `NewSSE`, obrigatoriamente: o SDK assume a resposta e fecha o
// corpo do pedido, e um `ReadSignals` depois dele encontra o corpo fechado. Isso
// é garantido pela ordem no `comandoDoMestre`, que chama a mutação primeiro — e
// a armadilha está registrada no `piloto_mesa_action.go`, onde ela passou VERDE
// em teste de handler e falhou no navegador.
func qualidadeDoDescanso(r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // o mesmo teto de 1 MB do `plataforma.DecodeJSON`
	var sinais struct {
		Qualidade string `json:"qualidadedodescanso"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return "", fmt.Errorf("não entendi a qualidade do descanso: %v", err)
	}
	if !qualidadesDoDescanso[sinais.Qualidade] {
		return "", fmt.Errorf("qualidade %q não existe; o livro tem ruim, normal, confortavel e luxuosa (p105)", sinais.Qualidade)
	}
	return sinais.Qualidade, nil
}

// trazOGrupo põe na fila cada personagem do grupo que ainda não está lá.
//
// É idempotente — o `populateParty` pula quem já está —, e é por isso que o
// botão continua clicável em vez de apagar depois do primeiro uso: o mestre que
// aceitou um jogador atrasado clica de novo e leva só o que faltava.
//
// O filtro de PAPEL é do `listPlayerCombatants`, e não daqui: o mestre costuma
// ter um PC próprio no roster, e uma segunda opinião sobre quem é o grupo faria
// esta tela discordar da SPA sobre a mesma pergunta (ALE-212).
func trazOGrupo(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	combatentes, err := st.listPlayerCombatants(c.R.Context(), c.CampaignID)
	if err != nil {
		return nil, errors.New("não deu para carregar o grupo desta campanha")
	}
	// O erro vem JUNTO com o estado parcial de propósito: pôr quatro dos cinco e
	// tropeçar no quinto deixou a mesa com quatro combatentes novos, e é esse o
	// estado que as outras telas precisam receber. Quem transmite o parcial é o
	// `comandoDoMestre` — ver o comentário lá.
	estado, err := st.populateParty(c.SessionID, combatentes)
	if estado == nil {
		estado = st.sessions.GetState(c.SessionID)
	}
	return estado, err
}

// mesaComando é o que a mutação de um comando do mestre recebe.
//
// Os quatro primeiros só precisavam do id da SESSÃO, e a assinatura era um
// `int64` — foi essa economia que deixou passar o defeito que este arquivo
// acabou de consertar: `encerrar cena` precisa da CAMPANHA, porque é de lá que
// vem o grupo cujas fichas expiram, e não tendo como recebê-la ela chamou o
// helper que não precisa dela e faz menos.
type mesaComando struct {
	R          *http.Request
	User       AuthUser
	CampaignID int64
	SessionID  int64
	// Sinais é o que a cena recebe de volta ALÉM do HTML, e a mutação escreve
	// nele quando quer mexer no estado do CLIENTE.
	//
	// Hoje só o formulário de acrescentar usa, e o motivo dele é o que justifica
	// o campo: ele se limpa DEPOIS de o servidor aceitar. Limpar no clique
	// custaria o que a pessoa digitou toda vez que a validação recusasse — e a
	// recusa mais comum é o nome, que é o campo mais caro de redigitar no meio
	// de um combate.
	//
	// QUEM GARANTE que a recusa não limpa nada é a ORDEM, e não um descarte aqui:
	// a mutação só escreve neste mapa depois de a sua própria escrita ter dado
	// certo, então numa recusa ele chega vazio. Eu tinha posto o descarte por
	// via das dúvidas e a sabotagem mostrou que ele era código morto — nenhum
	// caminho o alcançava, e um comentário dizendo que ele protegia algo era
	// pior do que não tê-lo.
	Sinais map[string]any
}

// encerraACena é o gesto INTEIRO, e a razão de ser função nomeada em vez de um
// literal na lista acima é que ela faz duas coisas que as outras três não fazem.
//
// A primeira é a REGRESSÃO da ALE-220, reaberta por este piloto: `EndScene` do
// store só mexe no rastreador, então a fila zerava na tela e a bênção de duração
// "cena" continuava viva na FICHA. O livro não deixa margem — "a habilidade dura
// uma cena inteira, encerrando-se quando esse momento da história acaba" (p227)
// —, e o `endSceneForTable` é o caminho único que expira as fichas do grupo
// ANTES de desligar a cena. Aqui é a mesma chamada e não a mesma sequência
// reescrita: gesto repetido é gesto que diverge.
//
// A segunda é o aviso: as fichas não estão no estado do rastreador, então sem o
// `session-rest` o efeito morto e o "usado 1/cena" ficariam na tela da SPA até
// alguém recarregar.
func encerraACena(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	estado, err := st.endSceneForTable(c.User, c.CampaignID, c.SessionID)
	if err != nil {
		return nil, err
	}
	st.sse.Emit(c.SessionID, "", "session-rest", map[string]any{
		"sessionId": c.SessionID, "scope": "scene",
	})
	return estado, nil
}

// comandoDoMestre é o caminho único dos quatro comandos.
//
// Eles só diferem na MUTAÇÃO, e o resto — resolver a mesa, exigir o papel,
// publicar para a SPA, redesenhar a cena — é idêntico. Sem o parâmetro seriam
// quatro cópias, e é numa delas que alguém esquece de publicar e a mesa fica
// vendo o turno velho.
func (s *Server) comandoDoMestre(
	mutar func(*Server, mesaComando) (*aovivo.SessionRuntimeState, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, sessionID, ok := mesaParams(w, r)
		if !ok {
			return
		}
		user := currentUser(r)
		_, papel, status, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		// A trava é aqui e não na tela: quem postar na mão leva 403, e o botão
		// escondido é só cortesia para quem não pode.
		if papel != "gm" {
			http.Error(w, "só o mestre comanda a mesa", http.StatusForbidden)
			return
		}

		sinais := map[string]any{}
		estado, err := mutar(s, mesaComando{
			R: r, User: user, CampaignID: campaignID, SessionID: sessionID, Sinais: sinais,
		})
		// O que POUSOU se transmite mesmo quando a chamada devolveu erro, e o
		// `trazer o grupo` é quem o exige: ele põe quatro dos cinco e tropeça no
		// quinto, e os quatro já são o estado da mesa. Segurar a transmissão
		// porque houve erro deixaria as outras telas com a fila de antes —
		// best-effort é sobre continuar apesar da falha, não sobre escondê-la
		// (ALE-155).
		//
		// A SPA continua ouvindo o hub: enquanto as duas telas existirem, uma
		// escrita por aqui tem de chegar lá.
		if estado != nil {
			s.publishSessionState(sessionID, estado)
		}
		s.respondeAoMestre(w, r, user, campaignID, sessionID, err, sinais)
	}
}

// respondeAoMestre devolve a cena remendada E a frase da recusa — as duas
// sempre, e as duas por SSE.
//
// Os comandos respondiam `http.Error`, e isso era um beco: o Datastar não
// desenha corpo de resposta 4xx, então a recusa não chegava a lugar nenhum e o
// mestre clicava olhando para uma tela que não mudava. É o mesmo defeito que a
// ALE-213 anotou no socket, onde o cliente não escutava o `exception` — e ele
// ficou urgente com o conserto da ALE-220 neste arquivo, porque encerrar a cena
// passou a poder falhar DE PROPÓSITO e deixar a cena ligada.
//
// O 403 não vem por aqui e continua sendo `http.Error`: ele é para quem posta na
// mão, e a tela de quem não é mestre nunca teve o botão.
//
// A cena é remendada NA HORA em vez de esperar o próximo tique do stream. O
// stream avisa-e-relê, então ele veria a mesma coisa daqui a até 200ms e o hash
// o faria calar — o remendo aqui é o que torna o botão mais clicado da sessão
// instantâneo. E ele vale também na recusa: redesenhar mostra que a cena
// continua ABERTA, que é a verdade que o mestre precisa ver ao lado da frase.
func (s *Server) respondeAoMestre(
	w http.ResponseWriter, r *http.Request,
	user AuthUser, campaignID, sessionID int64, recusa error, sinais map[string]any,
	soAsRegioes ...string,
) {
	sse := datastar.NewSSE(w, r)
	if view, _, err := s.loadMesaView(r.Context(), user, campaignID, sessionID); err == nil {
		// Por PADRÃO manda TODAS as regiões e não só as que mudaram, ao contrário
		// do stream: aqui não há digital anterior para comparar — este caminho
		// responde a um pedido, não mantém uma conexão.
		//
		// O que justificava mandar tudo era que quem recebe acabou de CLICAR, e
		// ninguém está no meio de um arrasto no instante em que pede outra coisa.
		// O GESTO CONTÍNUO da ALE-203 quebrou essa frase — no arrasto do pincel a
		// pessoa está no meio do gesto, e cada casa cruzada devolvia a Mesa
		// inteira: 353 KB medidos por clique. Daí o `soAsRegioes`.
		//
		// Falhar ao redesenhar não desfaz a mutação, que já aconteceu e já foi
		// transmitida; o stream corrige no próximo tique. Por isso é best-effort e
		// a frase sai de qualquer jeito.
		for _, regiao := range regioesDaMesa(view) {
			if !pedidaOuTodas(regiao.ID, soAsRegioes) {
				continue
			}
			if fragmento, err := renderFragmento(r.Context(), regiao.No); err == nil {
				_ = sse.PatchElements(fragmento)
			}
		}
	}
	frase := ""
	if recusa != nil {
		frase = recusa.Error()
	}
	// Sai nos DOIS caminhos: no da recusa para acender a frase, e no do acerto
	// para APAGAR a anterior. Um sinal que só se escreve quando dá errado deixa
	// a recusa de dois cliques atrás acesa sobre um comando que funcionou.
	sinais["erroDoComando"] = frase
	_ = sse.MarshalAndPatchSignals(sinais)
}

// pedidaOuTodas: lista vazia quer dizer "a Mesa inteira", que é o padrão de
// quase todo comando. Escrito como função e não como `if len(...) == 0` no laço
// para o caso vazio ficar dito uma vez só, em vez de a cada leitura do laço.
func pedidaOuTodas(id string, pedidas []string) bool {
	if len(pedidas) == 0 {
		return true
	}
	return slices.Contains(pedidas, id)
}
