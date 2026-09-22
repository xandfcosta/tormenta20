package live

import "fmt"

// O TESTE DE QUEM SANGRA, pendente na vez dele (T20 p236): "No início de seu
// turno, faça um teste de Constituição (CD 15). Se passar, você estabiliza […].
// Se falhar, você perde 1d6 pontos de vida."
//
// O JOGADOR rola e digita (decisão do dono, ALE-366), então o teste é um estado
// da vez que espera: primeiro o d20, e na falha o d6. Ele mora na cena, como o
// extrato da manutenção, porque quem recarrega a página no meio da vez tem de ler
// a mesma pergunta. A vez que gira o leva: o mestre conduz, e a mesa não trava.
type BleedingCheck struct {
	EntryID     string `json:"entryId"`
	CharacterID int64  `json:"characterId"`
	Label       string `json:"label"`
	// AwaitingD6 é o segundo passo: o d20 falhou e falta o dano.
	AwaitingD6 bool `json:"awaitingD6,omitempty"`
	// Roll e Total são o d20 que chegou e a soma com a Constituição — a faixa
	// mostra a conta, e não só o veredito.
	Roll  int `json:"roll,omitempty"`
	Total int `json:"total,omitempty"`
	// Outcome é a frase do resultado, e ele encerra o teste.
	Outcome string `json:"outcome,omitempty"`
}

// Resolved diz se o teste já terminou — a frase do resultado é o fim.
func (c *BleedingCheck) Resolved() bool { return c != nil && c.Outcome != "" }

// BleedingLine é a frase da faixa, uma para cada passo.
func BleedingLine(c *BleedingCheck, dc int) string {
	switch {
	case c == nil:
		return ""
	case c.Outcome != "":
		return c.Outcome
	case c.AwaitingD6:
		return fmt.Sprintf("%s falhou (%d contra %d) — role 1d6 de dano", c.Label, c.Total, dc)
	}
	return fmt.Sprintf("%s sangra — teste de Constituição (CD %d)", c.Label, dc)
}

// PendingBleeding devolve o teste da vez se ele espera o passo pedido — o d20
// (`awaitingD6` falso) ou o d6 —, e nil em qualquer outro caso: fora de cena,
// sem teste, ou já resolvido.
func (s *Scene) PendingBleeding(awaitingD6 bool) *BleedingCheck {
	if s == nil || s.Bleeding == nil || s.Bleeding.Resolved() || s.Bleeding.AwaitingD6 != awaitingD6 {
		return nil
	}
	return s.Bleeding
}
