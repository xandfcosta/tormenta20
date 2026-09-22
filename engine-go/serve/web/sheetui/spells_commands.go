package sheetui

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// OS COMANDOS DA ABA MAGIAS.
//
// Nenhum deles decide nem grava: as quatro regras do grimório são do
// `character.Plays` (ALE-350). O que mora aqui é de onde sai cada valor — a
// magia vem do caminho, e os aprimoramentos vêm dos sinais do diálogo.

func learnSpell(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	return s.plays.LearnSpell(r.Context(), row.ID, chi.URLParam(r, "magia"))
}

func forgetSpell(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	return s.plays.ForgetSpell(r.Context(), row.ID, chi.URLParam(r, "magia"))
}

func togglePrepared(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	return s.plays.TogglePrepared(r.Context(), row.ID, chi.URLParam(r, "magia"))
}

// castSpellFromSheet conjura, cobrando o PM.
//
// A ficha COMPUTADA atravessa porque a conta do custo precisa dela inteira — os
// aprimoramentos, o teto da p224 e as reduções saem do que o motor já montou, e
// recomputá-la dentro do caso de uso faria o mesmo trabalho duas vezes no mesmo
// pedido.
func castSpellFromSheet(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	spellID := chi.URLParam(r, "magia")
	spell, known := catalog.LookupSpell(spellID)
	if !known {
		return fmt.Errorf("a magia %q não existe no livro", spellID)
	}
	// A HORA É PERGUNTADA ANTES e a ação é cobrada DEPOIS. A execução da magia
	// diz o que ela custa do turno (p233), e a conjuração ainda pode ser
	// recusada pelo grimório, pela preparação ou pelo PM — cobrar antes tiraria
	// a padrão de alguém por uma magia que nunca saiu.
	cost := engine.ActionCost(spell.Execution)
	if err := s.deps.ActionFitsOnTurn(r.Context(), row.ID, cost); err != nil {
		return err
	}
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	if err := s.plays.Cast(r.Context(), dto, spellID, signals.augments()); err != nil {
		return err
	}
	return s.deps.SpendActionOnTurn(r.Context(), row.ID, cost)
}
