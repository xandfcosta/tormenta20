package live

import "fmt"

// O ATAQUE PROPOSTO, e a divisa de quem decide sobre ele.
//
// A mesma do movimento no tabuleiro, e pela mesma razão: o que um jogador faz é
// uma PROPOSTA que a mesa vê, e quem transforma proposta em consequência é o
// mestre. Aqui a consequência é PV saindo de uma linha, que é a coisa menos
// reversível da sessão.
//
// # Por que o provisório mora no REGIME e não no tabuleiro
//
// O ataque é entre LINHAS DA FILA, não entre peças: uma mesa sem mapa aberto
// continua tendo combate, e o alvo de um ataque é um combatente, não um
// quadrado. Guardá-lo no `BoardState` tornaria o combate refém de ter um
// tabuleiro aberto — e o tabuleiro é uma superfície, não a cena.
//
// # Por que os NÚMEROS chegam prontos
//
// Quem resolve `d20` contra Defesa é o `engine.ResolveAttack`, e este contexto
// não o alcança (ver o `boundary_test.go`): o regime fala de quem está na mesa
// e de quanto PV cada um tem, e a regra do livro fala de dados. A tradução é da
// camada de caso de uso, que conhece os dois — é o mesmo desenho da porta de
// vitais da ficha.

// PendingAttack é um ataque rolado e ainda não confirmado.
//
// Ele carrega a CONTA INTEIRA e não só o dano, porque é isso que a mesa lê
// enquanto decide: "19 contra Defesa 17, acertou; 1d8 deu 8, mais 3; a RD comeu
// 5" é uma frase que dispensa perguntar ao servidor.
type PendingAttack struct {
	AttackerEntryID string `json:"attackerEntryId"`
	TargetEntryID   string `json:"targetEntryId"`
	Weapon          string `json:"weapon"`
	Roll            int    `json:"roll"`
	Total           int    `json:"total"`
	Hit             bool   `json:"hit"`
	Critical        bool   `json:"critical"`
	Dice            []int  `json:"dice,omitempty"`
	RawDamage       int    `json:"rawDamage"`
	Absorbed        int    `json:"absorbed"`
	Damage          int    `json:"damage"`
	// ByUserID é quem rolou. O mestre confirma por qualquer um; quem propôs
	// cancela o que é dele.
	ByUserID int64 `json:"byUserId"`
}

// Attacker descreve quem está agindo sobre o provisório.
type Attacker struct {
	UserID int64
	Role   string
}

// ProposeAttack guarda o ataque rolado, sem tocar em PV nenhum.
//
// No máximo UM por sessão, e o novo substitui o antigo — dois provisórios
// simultâneos são duas verdades sobre a mesma cena, e a mesa não teria como
// saber qual confirmar. É a mesma decisão do `ProposeMove`.
func ProposeAttack(st *SessionRuntimeState, ataque PendingAttack) error {
	if FindEntryIndex(st, ataque.AttackerEntryID) < 0 {
		return fmt.Errorf("quem ataca (%s) não está na fila", ataque.AttackerEntryID)
	}
	if FindEntryIndex(st, ataque.TargetEntryID) < 0 {
		return fmt.Errorf("o alvo (%s) não está na fila", ataque.TargetEntryID)
	}
	st.PendingAttack = &ataque
	return nil
}

// CommitAttack aplica o dano na linha do alvo e some com o provisório.
//
// SÓ O MESTRE, e é a mesma frase do `CommitMove`: o que o jogador rolou é um
// rascunho para a mesa ver, e quem diz que aconteceu é quem toca a cena. Um
// ataque que ERRA também se confirma — errar é uma coisa que acontece, e o
// provisório tem de sair da tela do mesmo jeito.
func CommitAttack(st *SessionRuntimeState, quem Attacker) error {
	ataque := st.PendingAttack
	if ataque == nil {
		return fmt.Errorf("não há ataque proposto para confirmar")
	}
	if quem.Role != "gm" {
		return fmt.Errorf("só o mestre põe o dano na ficha: o seu ataque é um rascunho para a mesa ver")
	}
	// O ALVO É CONFERIDO DE NOVO na confirmação: entre rolar e confirmar ele
	// pode ter saído da fila, e o que vale é a mesa no instante em que o PV
	// muda.
	if FindEntryIndex(st, ataque.TargetEntryID) < 0 {
		return fmt.Errorf("o alvo saiu da fila entre a rolagem e a confirmação")
	}
	if ataque.Damage > 0 {
		perda := int64(-ataque.Damage)
		if err := DeltaEntryVitals(st, ataque.TargetEntryID, &perda, nil, 0); err != nil {
			return err
		}
	}
	st.PendingAttack = nil
	return nil
}

// CancelAttack descarta o provisório sem mexer em ninguém. O mestre cancela por
// qualquer um — é ele quem toca a mesa quando o jogador caiu da rede —, e o
// jogador só o que ele mesmo rolou.
func CancelAttack(st *SessionRuntimeState, quem Attacker) error {
	ataque := st.PendingAttack
	if ataque == nil {
		return fmt.Errorf("não há ataque proposto para cancelar")
	}
	if quem.Role != "gm" && ataque.ByUserID != quem.UserID {
		return fmt.Errorf("o ataque proposto não é seu")
	}
	st.PendingAttack = nil
	return nil
}
