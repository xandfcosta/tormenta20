package sheetui

import (
	"net/http"

	"github.com/go-chi/chi/v5"

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
func castSpellFromSheet(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.plays.Cast(r.Context(), dto, chi.URLParam(r, "magia"), sinais.augments())
}
