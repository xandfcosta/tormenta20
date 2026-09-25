package engine

// A QUEM UMA EMENDA SE APLICA (ALE-387).
//
// O mestre muda regra para a mesa INTEIRA ou para UM personagem, e o escopo é
// propriedade da emenda — não o lugar onde ela é guardada. Foi essa a lição que
// derrubou o desenho anterior: com as emendas viajando no personagem, os dois
// escopos chegavam ao motor já misturados pelo carregamento e a procedência
// morria no caminho.
//
// # Por que um seletor, e não um booleano "vale para todos"
//
// Um booleano responde a pergunta de hoje e nenhuma de amanhã. O seletor é o
// mesmo encaixe onde entra "todo anão ignora a arma desbalanceada" — a regra que
// o mestre vai querer escrever assim que tiver os dois primeiros casos —, e
// acrescentá-la é uma variante a mais, não um desenho novo.

// SelectorKind é a espécie de alvo. Ela é string e não inteiro porque atravessa
// o banco: um número trocaria de sentido no dia em que a lista mudasse de ordem.
type SelectorKind string

const (
	// SelectorEveryone é toda ficha jogada nesta campanha.
	SelectorEveryone SelectorKind = "everyone"
	// SelectorCharacter é UMA ficha. O id é o do CLONE que vive na campanha, e
	// não o do molde do elenco — o molde não está em mesa nenhuma.
	SelectorCharacter SelectorKind = "character"
)

// Selector diz a quem uma emenda alcança.
//
// @example engine.Selector{Kind: engine.SelectorCharacter, CharacterID: 42}
type Selector struct {
	Kind        SelectorKind `json:"kind"`
	CharacterID int          `json:"characterId,omitempty"`
}

// EveryoneIn é o seletor da mesa inteira.
func EveryoneIn() Selector { return Selector{Kind: SelectorEveryone} }

// OnlyCharacter é o seletor de uma ficha.
func OnlyCharacter(id int) Selector {
	return Selector{Kind: SelectorCharacter, CharacterID: id}
}

// Matches diz se esta emenda alcança este personagem.
//
// ESPÉCIE DESCONHECIDA NÃO ALCANÇA NINGUÉM, e é o lado seguro dos dois jeitos:
// uma concessão que não se aplica deixa a ficha como o livro manda, e um
// silêncio que não se aplica também. A lista mora no banco e sobrevive a um
// rollback do binário, então uma espécie que este Go não conhece não pode mudar
// número de ficha às cegas.
//
// @example engine.EveryoneIn().Matches(ch) // true
func (s Selector) Matches(ch Character) bool {
	switch s.Kind {
	case SelectorEveryone:
		return true
	case SelectorCharacter:
		// O zero é recusado de propósito: um `characterId` que não foi preenchido
		// casaria com o `Character{}` dos fixtures e dos testes, e a emenda de
		// uma mesa apareceria no oráculo.
		return s.CharacterID != 0 && s.CharacterID == ch.ID
	}
	return false
}
