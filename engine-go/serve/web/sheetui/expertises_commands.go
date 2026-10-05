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
	name := chi.URLParam(r, "nome")
	if decoded, err := url.PathUnescape(name); err == nil {
		return decoded
	}
	return name
}

func toggleTraining(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	return s.plays.ToggleTraining(r.Context(), row.ID, expertiseName(r))
}

// rollExpertise rola o teste e o põe na MESA (p220-221).
//
// O D20 vem do sinal e é OPCIONAL: nulo é o servidor rolar, e um valor é o que a
// mesa rolou no dado de verdade. Os dois são de primeira classe por decisão do
// dono, e a faixa da mesa DIZ qual dos dois foi.
//
// PUBLICAR É PELA PORTA, e não daqui: a ficha não conhece a mesa e não deve
// conhecer — quem sabe em que sessão este personagem está é o hospedeiro. É a
// mesma divisa do `ActionFitsOnTurn`.
func rollExpertise(s Scene, r *http.Request, row sqlcgen.Character, sig Signals) error {
	nome := expertiseName(r)
	daMesa := theDieFromTheTable(sig.ExpertiseD20)
	teste, err := s.plays.RollExpertise(r.Context(), row, nome, daMesa)
	if err != nil {
		return err
	}
	return s.deps.PublishSkillTest(r.Context(), row.ID, nome, teste, daMesa != nil)
}

// theDieFromTheTable estreita o d20 que chegou do fio, e trata o ZERO como
// "não informado".
//
// O zero é a convenção desta cena — os sinais numéricos nascem em 0, como o
// `item_roll_hp` ao lado —, e aqui ele não é ambíguo: **0 não é face de d20**.
// Declarar o sinal como nulo seria a alternativa, e ela custa mais do que
// resolve: um campo numérico vazio devolve `null` em alguns navegadores e `”`
// noutros, e o `ResolveSkillTest` recusaria os dois com a frase do dado.
func theDieFromTheTable(n *int64) *int {
	if n == nil || *n == 0 {
		return nil
	}
	narrowed := int(*n)
	return &narrowed
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
func criaOOficio(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	name, attribute := "", "intelligence"
	if signals.NewExpertise != nil {
		name = strings.TrimSpace(*signals.NewExpertise)
	}
	if signals.NewAttribute != nil && engine.IsAttributeKey(*signals.NewAttribute) {
		attribute = *signals.NewAttribute
	}
	return s.plays.AddCraft(r.Context(), row.ID, name, attribute)
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
	after, err := proficiencySwap(dto, chi.URLParam(r, "categoria"))
	if err != nil {
		return err
	}
	return s.plays.SaveProficiencies(r.Context(), row.ID, after)
}

// restoresDefaultClass joga fora os ajustes manuais.
func restoresDefaultClass(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.plays.SaveProficiencies(r.Context(), row.ID, classDefault(dto))
}
