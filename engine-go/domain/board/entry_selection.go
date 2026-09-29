package board

// QUEM O MESTRE ESCOLHEU mandar para o mapa: o conjunto, e a leitura dele no
// corpo da mensagem.
//
// A leitura do corpo mora aqui e não na plataforma porque ela DEVOLVE um tipo do
// tabuleiro: ler campo genérico é plataforma, mas interpretar o que os campos
// SIGNIFICAM é domínio.
//
// Nada aqui sabe por onde o pedido entrou. Quem chama estas funções hoje é a
// cena da Mesa.
//
// Aqui morava um `ParseScene` que remontava o `BoardState` inteiro a partir de
// um `map[string]any`. Ele ficou sem chamador quando a API JSON saiu (ALE-277).

// EntrySelection nomeia as linhas da iniciativa que o mestre escolheu trazer.
//
// Nil é TODAS, e isso não é conveniência: é o que `board-Populate` sem
// `entryIds` significa, e uma aba antiga continua mandando exatamente isso.
// Lista VAZIA é diferente de ausente — "não escolhi" não é "escolhi ninguém".
type EntrySelection map[string]bool

func (s EntrySelection) wants(entryID string) bool { return s == nil || s[entryID] }

// ChosenEntries lê do corpo as linhas que o mestre escolheu trazer.
//
// Ausente devolve nil — TODAS, que é o significado que esse campo sempre teve.
// Lista vazia devolve conjunto vazio, que não traz ninguém: os dois casos são
// diferentes de propósito.
func ChosenEntries(body map[string]any, key string) EntrySelection {
	raw, ok := body[key].([]any)
	if !ok {
		return nil
	}
	chosen := EntrySelection{}
	for _, item := range raw {
		if id, ok := item.(string); ok {
			chosen[id] = true
		}
	}
	return chosen
}
