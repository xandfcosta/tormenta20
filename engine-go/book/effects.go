package book

import (
	"sync"
)

// efeitoDoLivro é um tipo de efeito — a família que a condição carrega no rodapé.
type EffectKind struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	BookPage    int    `json:"bookPage"`
}

// nomeDoEfeito resolve o id que a condição guarda (`tags: ["medo"]`) no nome que
// se lê. Id desconhecido volta como veio: a tag é dado, e dado envelhece — some
// o elo, não o rótulo.
func EffectName(id string) string {
	for _, e := range EffectKinds() {
		if e.ID == id {
			return e.Name
		}
	}
	return id
}

func EffectFields(e EffectKind) []string { return []string{e.Name, e.Description} }

// ── escola de magia ──────────────────────────────────────────────────────────

// escolaDeMagia é a família de uma magia (T20 p172). Ela mora aqui, ao lado do
// tipo de efeito, porque nasceu pela mesma razão: a magia a CITA, e citação sem
// destino é texto morto. E as duas coisas se tocam — o livro diz que "escolas de
// magia contam como tipos de efeitos".
type SpellSchool struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Abrev é a forma curta que o livro imprime nas tabelas ("Abjur"). Ilusão
	// não tem — o livro não deu —, e vazio aqui é ausência e não dado.
	Abrev    string `json:"abrev,omitempty"`
	BookPage int    `json:"bookPage"`
}

// nomeDaEscola resolve o id que a magia guarda ("evocacao") no nome que se lê.
//
// Sai do CATÁLOGO e não de uma tabela no código, e isso é o conserto de uma
// duplicação que eu mesmo criei duas horas antes: escrevi as oito à mão para
// rotular o filtro, e agora elas têm verbete. Duas listas dos mesmos oito nomes
// divergem na primeira correção de acento.
func SchoolName(id string) string {
	for _, e := range SpellSchools() {
		if e.ID == id {
			return e.Name
		}
	}
	return id
}

func SchoolFields(e SpellSchool) []string {
	return []string{e.Name, e.Abrev, e.Description}
}

// ── o texto com elos dentro ──────────────────────────────────────────────────

var (
	efeitosUmaVez   sync.Once
	efeitosDoAcervo []EffectKind
)

func EffectKinds() []EffectKind {
	efeitosUmaVez.Do(func() {
		efeitosDoAcervo = ListOf[EffectKind]("tipos-de-efeito")
	})
	return efeitosDoAcervo
}

var (
	escolasUmaVez  sync.Once
	escolasDoLivro []SpellSchool
)

func SpellSchools() []SpellSchool {
	escolasUmaVez.Do(func() {
		escolasDoLivro = ListOf[SpellSchool]("escolas-de-magia")
	})
	return escolasDoLivro
}
