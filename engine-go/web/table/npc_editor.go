package table

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"t20engine/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/aovivo"
	"t20engine/creature"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// O EDITOR DE BLOCO (ALE-269): mexer nos números da cópia, e escrever do zero.
//
// É a resposta à queixa que abriu a ALE-137 — *"o mestre acaba tendo que
// imaginar, ou inventar, ou anotar em algum lugar os itens, PM, perícias"* —, e
// é a segunda metade da superfície 6b: lá o NPC nasce como cópia do livro, aqui
// ele vira o NPC daquela campanha. Os dois caminhos terminam neste formulário; o
// que muda é a SEMENTE.
//
// # O RASCUNHO mora no NAVEGADOR, e essa é a decisão que desenha o arquivo
//
// Decisão do dono: "Salvar explícito, e Cancelar desfaz de verdade". Um rascunho
// no servidor precisaria de dono, de prazo de validade e de uma resposta para
// "duas abas editando o mesmo NPC" — e ainda assim Cancelar teria de apagá-lo.
// Num sinal do navegador, Cancelar não precisa desfazer nada: NADA FOI ESCRITO.
//
// O preço é conhecido e aceito: recarregar a página no meio da edição perde o
// rascunho, exatamente como fechar o diálogo da SPA perde o dele.
//
// # Por que as LISTAS passam pelo servidor mesmo assim
//
// Datastar não tem laço no cliente, então uma lista de ataques de tamanho
// variável só existe como HTML que alguém escreveu. Quem escreve é o servidor.
// Acrescentar um ataque é: mandar o rascunho inteiro, o servidor mexe na LISTA e
// devolve o rascunho novo mais as linhas redesenhadas.
//
// Isso NÃO é gravar: o banco não é tocado. É o servidor emprestando as mãos para
// uma mudança de forma que o navegador não sabe fazer sozinho — e é por isso que
// estes caminhos não passam pelo `gmCommand`, que redesenha a Mesa
// inteira. O único que grava é o `saveDraft`.

func (s Scene) RoutesEditorNpc(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/elenco/npc"
	r.Post(base+"/{npcId}/editar", s.openDraft)
	r.Post(base+"/novo", s.openDraft)
	r.Post(base+"/rascunho/{lista}/nova", s.moveList(addList))
	r.Post(base+"/rascunho/{lista}/{indice}/remover", s.moveList(tiraDaLista))
	r.Post(base+"/rascunho/salvar", s.gmCommand(saveDraft))
}

// npcDraft é o formulário aberto, e não o bloco guardado.
//
// A diferença tem duas linhas e as duas importam. `ID` zero quer dizer NOVO — é
// o que separa criar de editar sem um segundo sinal para o modo. E `Conjura` é
// booleano enquanto o `CreatureBlock.PM` é PONTEIRO: um formulário não sabe
// digitar "ausente", então a caixa guarda um número e o interruptor diz se ele
// conta. Quem traduz de volta é o `draftBlock`, num lugar só.
type npcDraft struct {
	ID      int64          `json:"id"`
	Nome    string         `json:"nome"`
	Conjura bool           `json:"conjura"`
	Bloco   creature.Block `json:"bloco"`
}

// pageDraft lê o rascunho que veio nos sinais.
//
// Datastar manda TODOS os sinais da página em cada `@post`, então o formulário
// não precisa ser um `<form>` nem juntar campo por campo: ele chega inteiro, com
// a última tecla incluída. É isso que faz "acrescentar um ataque" não perder o
// que estava sendo digitado nos outros campos.
func pageDraft(r *http.Request) (npcDraft, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var sinais struct {
		Rascunho npcDraft `json:"rascunho"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return npcDraft{}, fmt.Errorf("não entendi o formulário: %v", err)
	}
	return sinais.Rascunho, nil
}

// openDraft serve os dois caminhos, e é de propósito que seja um só.
//
// Com `npcId` no caminho, a semente é o bloco guardado; sem ele, o bloco em
// branco. Dois handlers seriam dois lugares para o formulário nascer diferente —
// e o defeito apareceria como "criar do zero não tem a aba de perícias".
func (s Scene) openDraft(w http.ResponseWriter, r *http.Request) {
	c, ok := s.tableGm(w, r)
	if !ok {
		return
	}
	rascunho := paraOFormulario(0, "", blocoEmBranco())
	if chi.URLParam(r, "npcId") != "" {
		linha, bloco, err := s.campaignNpc(c)
		if err != nil {
			writeSignals(w, r, map[string]any{"erroDoComando": err.Error()})
			return
		}
		rascunho = paraOFormulario(linha.ID, linha.Name, bloco)
	}
	s.respondDraft(w, r, c, rascunho)
}

// moveList é o corpo dos quatro gestos de forma — acrescentar e tirar, em
// ataques, perícias e habilidades.
//
// Um handler e não seis: os seis fazem a MESMA coisa (ler o rascunho, mexer numa
// lista, devolver o rascunho e as linhas), e seis cópias dariam seis lugares para
// alguém esquecer de devolver o fragmento — com o sintoma de a linha nova só
// aparecer no próximo clique.
func (s Scene) moveList(
	mexer func(*npcDraft, string, int) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := s.tableGm(w, r)
		if !ok {
			return
		}
		rascunho, err := pageDraft(r)
		if err != nil {
			writeSignals(w, r, map[string]any{"erroDoRascunho": err.Error()})
			return
		}
		// O índice é opcional: acrescentar não tem um. `-1` e não zero, porque
		// zero é a primeira linha — um erro de leitura silencioso apagaria a
		// linha de cima em vez de não fazer nada.
		indice := -1
		if bruto := chi.URLParam(r, "indice"); bruto != "" {
			if n, err := strconv.Atoi(bruto); err == nil {
				indice = n
			}
		}
		if err := mexer(&rascunho, chi.URLParam(r, "lista"), indice); err != nil {
			writeSignals(w, r, map[string]any{"erroDoRascunho": err.Error()})
			return
		}
		s.respondDraft(w, r, c, rascunho)
	}
}

// addList põe uma linha VAZIA no fim.
//
// Vazia e não preenchida com um exemplo: um "Clava +7" de mentira que o mestre
// esquecesse de trocar viraria um ataque de verdade na mesa. O `validateCreature`
// recusa ataque sem nome, então a linha em branco não chega ao banco — ela é uma
// pergunta, não um dado.
func addList(rascunho *npcDraft, lista string, _ int) error {
	switch lista {
	case listaDeAtaques:
		rascunho.Bloco.Attacks = append(rascunho.Bloco.Attacks, creature.Attack{})
	case listaDePericias:
		rascunho.Bloco.Skills = append(rascunho.Bloco.Skills, creature.Skill{})
	case listaDeHabilidades:
		rascunho.Bloco.SpecialAbilities = append(rascunho.Bloco.SpecialAbilities, "")
	default:
		return fmt.Errorf("lista %q não existe no bloco; são %s, %s e %s",
			lista, listaDeAtaques, listaDePericias, listaDeHabilidades)
	}
	return nil
}

// tiraDaLista remove UMA linha pelo índice.
func tiraDaLista(rascunho *npcDraft, lista string, indice int) error {
	switch lista {
	case listaDeAtaques:
		fora, err := itemWithout(rascunho.Bloco.Attacks, indice)
		rascunho.Bloco.Attacks = fora
		return err
	case listaDePericias:
		fora, err := itemWithout(rascunho.Bloco.Skills, indice)
		rascunho.Bloco.Skills = fora
		return err
	case listaDeHabilidades:
		fora, err := itemWithout(rascunho.Bloco.SpecialAbilities, indice)
		rascunho.Bloco.SpecialAbilities = fora
		return err
	default:
		return fmt.Errorf("lista %q não existe no bloco; são %s, %s e %s",
			lista, listaDeAtaques, listaDePericias, listaDeHabilidades)
	}
}

// itemWithout devolve a fatia sem a posição pedida, ou a mesma fatia e um erro.
//
// FATIA NOVA e nunca o `append(s[:i], s[i+1:]...)` no lugar: aquele escreve por
// cima da memória de quem chamou, e aqui quem chamou é o rascunho que ainda pode
// ser recusado no passo seguinte. Um índice fora da faixa é o botão de uma linha
// que outra aba já apagou — recusa com o número, não estoura.
func itemWithout[T any](itens []T, indice int) ([]T, error) {
	if indice < 0 || indice >= len(itens) {
		return itens, fmt.Errorf("a linha %d não existe: a lista tem %d", indice+1, len(itens))
	}
	fora := make([]T, 0, len(itens)-1)
	fora = append(fora, itens[:indice]...)
	return append(fora, itens[indice+1:]...), nil
}

// saveDraft é o ÚNICO caminho desta tela que toca o banco.
//
// Cria quando o rascunho não tem id e atualiza quando tem, e a escolha é do
// próprio rascunho e não de uma rota diferente: o formulário é o mesmo, e uma
// segunda rota seria um segundo lugar para validar.
//
// Passa pelo `gmCommand` — ao contrário dos gestos de forma — porque aqui
// a CENA muda: a lista do elenco ganha ou perde uma linha, e ela é uma região
// que precisa ser redesenhada.
func saveDraft(st Scene, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	// A RECUSA VAI PARA O EDITOR, e não para o `erroDoComando` do rodapé — que é
	// a saída normal do `gmCommand`. É o mesmo argumento do
	// `erroDoMovimento`: quem lê a frase está com o formulário aberto POR CIMA do
	// rodapé, e uma recusa escrita atrás do diálogo é uma recusa que ninguém lê.
	// Medido no navegador antes de virar isto: "salvar sem nome" não dizia nada.
	//
	// Por isso o erro é devolvido como SINAL e a função sai sem erro: o comando
	// não falhou, ele recusou — e quem tinha de saber já soube.
	if err := st.triesSaveDraft(c); err != nil {
		c.Sinais["erroDoRascunho"] = err.Error()
		return st.deps.Sessions().GetState(c.SessionID), nil
	}
	// FECHA o editor no mesmo passo em que grava, e não num clique à parte: o
	// gesto do mestre é "salvar e voltar", e deixar o formulário aberto sobre uma
	// lista já atualizada faria ele clicar em Salvar de novo por não saber se
	// pegou.
	c.Sinais["rascunhoaberto"] = false
	c.Sinais["erroDoRascunho"] = ""
	// O ELENCO NÃO É ESTADO DE SESSÃO — guardar um NPC não muda a fila nem o
	// mapa. O estado volta mesmo assim porque é dele que o `gmCommand`
	// redesenha as regiões, e sem isso a lista só mostraria a mudança no F5.
	return st.deps.Sessions().GetState(c.SessionID), nil
}

// triesSaveDraft é o caminho inteiro do salvar, do sinal ao banco.
//
// Separado do `saveDraft` para que TODA recusa saia por um lugar só — o
// `erroDoRascunho` do editor. Com as validações espalhadas em `return nil, err`,
// cada uma teria de lembrar de escrever no sinal certo, e a que esquecesse
// falaria atrás do diálogo.
func (s Scene) triesSaveDraft(c commandCtx) error {
	rascunho, err := pageDraft(c.R)
	if err != nil {
		return err
	}
	bloco := draftBlock(rascunho)
	if err := creature.Validate(rascunho.Nome, &bloco); err != nil {
		return err
	}
	creature.Normalize(&bloco)
	blob, err := json.Marshal(bloco)
	if err != nil {
		return fmt.Errorf("não deu para guardar o bloco de %q", rascunho.Nome)
	}
	return s.gravaOBloco(c, rascunho, string(blob))
}

func (s Scene) gravaOBloco(c commandCtx, rascunho npcDraft, blob string) error {
	agora := plataforma.NowISO()
	if rascunho.ID == 0 {
		_, err := s.deps.Queries().CreateCampaignCreature(c.R.Context(), sqlcgen.CreateCampaignCreatureParams{
			Campaignid: c.CampaignID, Name: rascunho.Nome, Block: blob,
			Createdat: agora, Updatedat: agora,
		})
		if err != nil {
			return fmt.Errorf("não deu para guardar %q no elenco: %v", rascunho.Nome, err)
		}
		return nil
	}
	// A CONFERÊNCIA de campanha é a mesma do `campaignNpc`, e ela é a trava:
	// o id vem do rascunho, que vem do navegador, e sem ela o mestre de uma mesa
	// reescreveria o elenco de outra — que é o material mais privado que um
	// mestre tem.
	if _, _, err := s.idCampaignNpc(c, rascunho.ID); err != nil {
		return err
	}
	if _, err := s.deps.Queries().UpdateCampaignCreature(c.R.Context(), sqlcgen.UpdateCampaignCreatureParams{
		ID: rascunho.ID, Name: rascunho.Nome, Block: blob, Updatedat: agora,
	}); err != nil {
		return fmt.Errorf("não deu para atualizar %q: %v", rascunho.Nome, err)
	}
	return nil
}

// paraOFormulario traduz o modelo para o formulário, e é a metade que faltava do
// `draftBlock`.
//
// Ela existe por causa do PM e de uma armadilha que só aparece na tela: o bloco
// guarda a AUSÊNCIA de mana (o Bandido não tem a linha), e um ponteiro nulo vira
// `pm` ausente no sinal — que chega ao navegador como `undefined`. A caixa
// nasceria com a palavra escrita dentro no instante em que o mestre marcasse
// "Conjura". O formulário guarda sempre um número; quem diz se ele conta é o
// interruptor.
func paraOFormulario(id int64, nome string, bloco creature.Block) npcDraft {
	conjura := bloco.PM != nil
	if bloco.PM == nil {
		zero := 0
		bloco.PM = &zero
	}
	// As três listas nunca chegam nulas ao navegador: `$rascunho.bloco.attacks.length`
	// numa lista ausente estoura a expressão do contador da aba, e o número some
	// sem erro em lugar nenhum.
	creature.Normalize(&bloco)
	return npcDraft{ID: id, Nome: nome, Conjura: conjura, Bloco: bloco}
}

// draftBlock traduz o formulário de volta para o modelo.
//
// A única tradução é o PM, e ela é a razão de o rascunho não ser o bloco: o
// livro escreve a linha de mana só em quem conjura (o Centauro Xamã tem 20 PM,
// p290; o Bandido não tem linha nenhuma), e um zero ali diria "tem mana e está
// sem" — que é outro estado. O formulário guarda um número e um interruptor; o
// bloco guarda a AUSÊNCIA.
func draftBlock(rascunho npcDraft) creature.Block {
	bloco := rascunho.Bloco
	if !rascunho.Conjura {
		bloco.PM = nil
		return bloco
	}
	if bloco.PM == nil {
		zero := 0
		bloco.PM = &zero
	}
	return bloco
}

// respondDraft devolve o rascunho E as linhas redesenhadas.
//
// AS DUAS COISAS, sempre. Só os sinais deixaria a lista com o número de linhas
// antigo — o ataque existiria no rascunho e não na tela. Só o HTML deixaria a
// linha nova ligada a um caminho de sinal que não existe, e ela nasceria muda.
func (s Scene) respondDraft(w http.ResponseWriter, r *http.Request, c commandCtx, rascunho npcDraft) {
	sse := datastar.NewSSE(w, r)
	for _, fragmento := range draftLists(c, rascunho) {
		if html, err := ui.RenderFragment(r.Context(), fragmento); err == nil {
			_ = sse.PatchElements(html)
		}
	}
	_ = sse.MarshalAndPatchSignals(map[string]any{
		"rascunho": rascunho,
		// ABRIR o editor e apagar a recusa anterior fazem parte da resposta: uma
		// frase de erro de dois gestos atrás sobre um formulário que acabou de
		// abrir é a recusa certa na tela errada.
		"rascunhoaberto": true,
		"erroDoRascunho": "",
	})
}

// tableGm é a leitura de acesso dos caminhos do rascunho.
//
// A trava é do SERVIDOR e não do botão escondido, como no resto da Mesa: o
// elenco é a preparação da campanha, e quem postar na mão leva 403.
func (s Scene) tableGm(w http.ResponseWriter, r *http.Request) (commandCtx, bool) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return commandCtx{}, false
	}
	userID := s.deps.CurrentUserID(r)
	_, papel, status, err := s.deps.SessionForCaller(r.Context(), userID, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return commandCtx{}, false
	}
	if papel != "gm" {
		http.Error(w, "só o mestre monta o elenco", http.StatusForbidden)
		return commandCtx{}, false
	}
	return commandCtx{R: r, User: userID, CampaignID: campaignID, SessionID: sessionID}, true
}
