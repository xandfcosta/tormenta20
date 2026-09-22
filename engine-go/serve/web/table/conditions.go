package table

import (
	"fmt"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/book"
	"t20engine/domain/catalog"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
)

// AS CONDIÇÕES do combatente na Mesa.
//
// O que se aplica aqui é RASTREIO e não regra, e a distinção é do desenho e não
// uma limitação: os números de um bloco de criatura são escritos à mão pelo
// mestre, então o motor não recalcula Defesa nem ataque a partir da condição —
// dizer que recalcularia seria mentir sobre um número que ninguém derivou. Por
// isso o crachá carrega o EFEITO por extenso: é o mestre quem aplica.
//
// A condição de um PERSONAGEM mora na FICHA, que é onde o motor a lê para mexer
// nos números e no instante de agir: na linha dele, o gesto da Mesa grava lá,
// pelo mesmo caso de uso da aba Efeitos (decisão do dono, ALE-368). A lista da
// linha da fila é o caminho do NPC, que ficha não tem. Antes a fila guardava
// uma lista própria também para PC, e o Atordoado marcado pela Mesa não
// atordoava ninguém.

// conditionEffect é o que o crachá diz ao passar o mouse.
//
// A lista e o NOME não moram aqui: `book.Catalogs().Condicoes` já é lida do
// catálogo e já vem ordenada por um collator pt-BR, e o `book.ConditionName` já
// resolve id → palavra do livro. Uma segunda cópia da tabela do livro é uma
// cópia que desvia.
func conditionEffect(id string) string {
	for _, c := range book.Catalogs().Conditions {
		if c.ID == id {
			return c.Description
		}
	}
	return ""
}

func (s Scene) ConditionRoutes(r chi.Router) {
	r.Post(sessionPattern+"/iniciativa/{entryId}/condicao/{id}",
		s.gmCommand(toggleCondition))
}

// toggleCondition liga ou desliga uma condição na linha.
//
// ALTERNA no servidor, e o conjunto final é montado AQUI: o clique manda só
// QUAL condição o mestre tocou, e quem lê a lista atual e devolve a nova é o
// servidor. O gesto carrega a INTENÇÃO e não o estado — uma tela que mandasse o
// conjunto inteiro apagaria a condição que outro remendo acabou de acrescentar.
func toggleCondition(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	entryID := chi.URLParam(c.R, "entryId")
	id := chi.URLParam(c.R, "id")
	// A VALIDAÇÃO é do catálogo e não de uma lista daqui, pela razão do
	// `book.Catalogs`. Sem ela, um id inventado entraria na linha e a tela o
	// desenharia como se fosse condição do livro.
	if !catalog.IsCondition(id) {
		return nil, fmt.Errorf("%q não é uma condição do livro (p394-395)", id)
	}
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	i := live.FindEntryIndex(state, entryID)
	if i < 0 {
		return nil, fmt.Errorf("combatente %q não está na fila", entryID)
	}

	if charID := state.Initiative[i].CharacterID; charID != nil {
		return toggleSheetCondition(st, c, *charID, id)
	}

	current := state.Initiative[i].Conditions
	fresh := make([]string, 0, len(current)+1)
	found := false
	for _, now := range current {
		if now == id {
			found = true
			continue
		}
		fresh = append(fresh, now)
	}
	if !found {
		fresh = append(fresh, id)
	}
	newState, err := st.deps.Sessions().UpdateInitiativeEntry(c.R.Context(), c.SessionID, entryID,
		live.EntryPatch{Conditions: &fresh})
	if err != nil {
		return newState, err
	}
	// O CONJUNTO NOVO VOLTA NUM SINAL, e sem isto o diálogo aberto mente: os
	// crachás dele são pintados a partir do sinal que a ABERTURA escreveu, e
	// depois de um clique aquele sinal descreve o estado de antes. O mestre
	// aplicaria "abalado", veria o crachá apagado, e clicaria de novo — tirando
	// a condição que ele acabou de pôr.
	c.Signals["row_conditions"] = strings.Join(fresh, ",")
	return newState, nil
}

// toggleSheetCondition alterna a condição NA FICHA do personagem da linha.
//
// A fila não muda — ela LÊ as condições da ficha a cada desenho —, e quem acorda
// as duas telas é o aviso de ficha mexida: a Mesa redesenha a linha e a ficha
// aberta do jogador redesenha a aba Efeitos.
func toggleSheetCondition(st Scene, c commandCtx, characterID int64, id string) (*live.SessionRuntimeState, error) {
	row, err := st.deps.Queries().GetCharacter(c.R.Context(), characterID)
	if err != nil {
		return nil, fmt.Errorf("ler a ficha %d: %w", characterID, err)
	}
	if err := st.plays.ToggleBookCondition(c.R.Context(), row, id); err != nil {
		return nil, err
	}
	st.deps.CharacterChanged(characterID)
	after, err := st.deps.Queries().GetCharacter(c.R.Context(), characterID)
	if err != nil {
		return nil, fmt.Errorf("reler a ficha %d: %w", characterID, err)
	}
	// O conjunto novo volta no sinal pela razão do ramo do NPC, logo acima.
	c.Signals["row_conditions"] = strings.Join(sheet.UnmarshalStrings(after.Activeconditions), ",")
	return st.deps.Sessions().State(c.R.Context(), c.SessionID)
}
