package table

import (
	"fmt"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/aovivo"
	"t20engine/book"
	"t20engine/catalog"
)

// AS CONDIÇÕES do combatente na Mesa (ALE-122, portadas na ALE-269).
//
// O que se aplica aqui é RASTREIO e não regra, e a distinção é do desenho e não
// uma limitação: os números de um bloco de criatura são escritos à mão pelo
// mestre, então o motor não recalcula Defesa nem ataque a partir da condição —
// dizer que recalcularia seria mentir sobre um número que ninguém derivou. Por
// isso o crachá carrega o EFEITO por extenso: é o mestre quem aplica.
//
// O PC continua aplicando condição pela FICHA, que é onde elas moram e onde o
// motor as lê para mexer nos números. A linha da fila é o caminho do NPC, que
// ficha não tem.

// conditionEffect é o que o crachá diz ao passar o mouse.
//
// A lista e o NOME não moram aqui: `book.Catalogs().Condicoes` já é lida do
// catálogo e já vem ordenada por um collator pt-BR, e o `book.ConditionName` já
// resolve id → palavra do livro. Eu tinha escrito as três de novo neste arquivo
// antes de procurar — e uma segunda cópia da tabela do livro é uma cópia que
// desvia, que é exatamente o defeito que a ALE-122 pagou.
func conditionEffect(id string) string {
	for _, c := range book.Catalogs().Condicoes {
		if c.ID == id {
			return c.Description
		}
	}
	return ""
}

func (s Scene) ConditionRoutes(r chi.Router) {
	r.Post("/mesa/{campaignId}/{sessionId}/initiative/{entryId}/condicao/{id}",
		s.gmCommand(toggleCondition))
}

// toggleCondition liga ou desliga uma condição na linha.
//
// ALTERNA no servidor, e o conjunto final é montado AQUI — o `EntryPatch`
// substitui a lista inteira, e a SPA manda o conjunto pronto porque a tela dela
// o tem em mãos. A Mesa manda só QUAL condição o mestre clicou, e é o servidor
// que lê a lista atual e devolve a nova: o clique carrega a intenção, não o
// estado, e uma tela que mandasse o conjunto inteiro apagaria a condição que
// outro remendo acabou de acrescentar.
func toggleCondition(st Scene, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	entryID := chi.URLParam(c.R, "entryId")
	id := chi.URLParam(c.R, "id")
	// A VALIDAÇÃO é do catálogo e não de uma lista daqui, pela razão do
	// `book.Catalogs`. Sem ela, um id inventado entraria na linha e a tela o
	// desenharia como se fosse condição do livro.
	if !catalog.IsCondition(id) {
		return nil, fmt.Errorf("%q não é uma condição do livro (p394-395)", id)
	}
	estado := st.deps.Sessions().GetState(c.SessionID)
	i := aovivo.FindEntryIndex(estado, entryID)
	if i < 0 {
		return nil, fmt.Errorf("combatente %q não está na fila", entryID)
	}

	atuais := estado.Initiative[i].Conditions
	novas := make([]string, 0, len(atuais)+1)
	achou := false
	for _, atual := range atuais {
		if atual == id {
			achou = true
			continue
		}
		novas = append(novas, atual)
	}
	if !achou {
		novas = append(novas, id)
	}
	estadoNovo, err := st.deps.Sessions().UpdateInitiativeEntry(c.SessionID, entryID,
		aovivo.EntryPatch{Conditions: &novas})
	if err != nil {
		return estadoNovo, err
	}
	// O CONJUNTO NOVO VOLTA NUM SINAL, e sem isto o diálogo aberto mente: os
	// crachás dele são pintados a partir do sinal que a ABERTURA escreveu, e
	// depois de um clique aquele sinal descreve o estado de antes. O mestre
	// aplicaria "abalado", veria o crachá apagado, e clicaria de novo — tirando
	// a condição que ele acabou de pôr.
	c.Sinais["condicoesdalinha"] = strings.Join(novas, ",")
	return estadoNovo, nil
}
