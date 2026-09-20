package board

import (
	"encoding/json"
	"fmt"
)

// QUEM O MESTRE ESCOLHEU mandar para o mapa: o conjunto, e a leitura dele no
// corpo da mensagem.
//
// A leitura do corpo mora aqui e não na plataforma porque ela DEVOLVE um tipo do
// tabuleiro: ler campo genérico é plataforma, mas interpretar o que os campos
// SIGNIFICAM é domínio.
//
// Nada aqui sabe por onde o pedido entrou. Quem chama estas funções hoje é a
// cena da Mesa.

// EntrySelection nomeia as linhas da iniciativa que o mestre escolheu trazer.
//
// Nil é TODAS, e isso não é conveniência: é o que `board-Populate` sem
// `entryIds` significa, e uma aba antiga continua mandando exatamente isso.
// Lista VAZIA é diferente de ausente — "não escolhi" não é "escolhi ninguém".
type EntrySelection map[string]bool

func (s EntrySelection) wants(entryID string) bool { return s == nil || s[entryID] }

// ParseScene lê a cena montada do corpo da mensagem. Passa pelo JSON de novo
// porque o corpo chega como `map[string]any` genérico, e reconstruir o
// `BoardState` campo a campo aqui seria uma segunda definição do formato de fio.
func ParseScene(raw any) (*BoardState, error) {
	blob, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("cena ilegível: %w", err)
	}
	var cena BoardState
	if err := json.Unmarshal(blob, &cena); err != nil {
		return nil, fmt.Errorf("cena ilegível: %w", err)
	}
	if cena.Tokens == nil {
		cena.Tokens = []BoardToken{}
	}
	return &cena, nil
}

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
