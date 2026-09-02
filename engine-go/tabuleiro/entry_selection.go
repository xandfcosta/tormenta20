package tabuleiro

import (
	"encoding/json"
	"fmt"
)

// Quem o mestre escolheu mandar para o mapa (ALE-204).
//
// Veio do `api/board_rules.go` na ALE-254, e o compilador apontou: a função lê
// o corpo da requisição mas DEVOLVE um tipo do tabuleiro, e a decisão sobre o
// que é uma seleção válida é do contexto que a consome. Ler campo genérico é
// plataforma; interpretar o que os campos SIGNIFICAM é domínio.

// As regras do tabuleiro: quem se move, quanto anda, o que vira peça.
//
// Este arquivo é o que SOBROU de `realtime_board.go` quando o socket.io foi
// apagado (ALE-253). O corte foi pelo receptor: o que era `(g *realtimeGateway)`
// era transporte e morreu junto; o que está aqui é aplicação, e não mudou uma
// linha ao mudar de vizinho. As rotas HTTP em `session_commands.go` e
// `board_commands.go` chamam exatamente as mesmas funções que os eventos
// chamavam.

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

// chosenEntries lê do corpo as linhas que o mestre escolheu trazer (ALE-204).
//
// Ausente devolve nil — TODAS, o significado que o evento sempre teve e que uma
// aba aberta antes desta mudança ainda manda. Lista vazia devolve conjunto
// vazio, que não traz ninguém: os dois casos são diferentes de propósito.
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
