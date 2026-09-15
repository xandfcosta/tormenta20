package board

import (
	"encoding/json"
	"fmt"
)

// Quem o mestre escolheu mandar para o mapa, e as regras do tabuleiro: quem se
// move, quanto anda, o que vira peça.
//
// A leitura do corpo mora aqui e não na plataforma porque ela DEVOLVE um tipo do
// tabuleiro: ler campo genérico é plataforma, mas interpretar o que os campos
// SIGNIFICAM é domínio.
//
// Nada aqui sabe por onde o pedido entrou. Quem chama estas funções hoje é a
// cena da Mesa.

// parseScene lê a cena montada do corpo da mensagem. Passa pelo JSON de novo
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
