package sheetui

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// OS COMANDOS DAS ABAS PERÍCIAS E PROFICIÊNCIAS.
//
// Nenhum deles decide nem grava: os cinco gestos são do `character.Plays`
// (ALE-350). O que mora aqui é o que é da TELA — de onde sai cada valor: o nome
// vem do caminho, o atributo vem do `<option>` que foi escolhido, e o par do
// ofício novo vem dos dois campos do diálogo.

// expertiseName lê o nome do caminho, desescapando como a API JSON fazia.
func expertiseName(r *http.Request) string {
	nome := chi.URLParam(r, "nome")
	if decodificado, err := url.PathUnescape(nome); err == nil {
		return decodificado
	}
	return nome
}

func toggleTraining(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	return s.plays.ToggleTraining(r.Context(), row.ID, expertiseName(r))
}

// swapAttribute repõe a perícia em outro atributo.
//
// O atributo vai no CAMINHO junto do nome: é o valor do `<option>` escolhido, e
// mandá-lo por sinal faria as opções de todas as linhas disputarem a mesma
// chave.
func swapAttribute(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	return s.plays.SwapAttribute(r.Context(), row.ID, expertiseName(r), chi.URLParam(r, "atributo"))
}

func removeCraft(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	return s.plays.RemoveCraft(r.Context(), row.ID, expertiseName(r))
}

// criaOOficio lê os dois campos do diálogo e manda o par.
//
// O atributo cai em Inteligência quando o sinal não veio ou não é atributo: é o
// padrão do formulário, e escolhê-lo aqui e não no caso de uso é deliberado —
// o `AddCraft` RECUSA o que não reconhece, porque um chamador que não é diálogo
// não tem por que herdar o padrão de um.
func criaOOficio(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	nome, atributo := "", "intelligence"
	if sinais.NovaPericia != nil {
		nome = strings.TrimSpace(*sinais.NovaPericia)
	}
	if sinais.NovoAtributo != nil && engine.IsAttributeKey(*sinais.NovoAtributo) {
		atributo = *sinais.NovoAtributo
	}
	return s.plays.AddCraft(r.Context(), row.ID, nome, atributo)
}

// toggleProficiency liga ou desliga UMA categoria.
//
// Manda a CATEGORIA e não o estado desejado, pela mesma razão do
// `toggleTraining`. O que a tela faz é DERIVAR a lista nova da ficha desenhada
// — a decisão de qual categoria entra e qual sai é do `proficiencySwap`, ao
// lado do resto da apresentação desta aba.
func toggleProficiency(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	depois, err := proficiencySwap(dto, chi.URLParam(r, "categoria"))
	if err != nil {
		return err
	}
	return s.plays.SaveProficiencies(r.Context(), row.ID, depois)
}

// restoresDefaultClass joga fora os ajustes manuais.
func restoresDefaultClass(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.plays.SaveProficiencies(r.Context(), row.ID, classDefault(dto))
}
