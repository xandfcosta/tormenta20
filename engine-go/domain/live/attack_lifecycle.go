package live

import "fmt"

// O CICLO DO PROVISÓRIO: propor, trocar o dado, julgar, rolar o dano, confirmar e cancelar. O vocabulário
// dele — o que o provisório leva, o recibo do que ele pagou, a condição prevista — mora no `attack.go`; aqui
// moram as transições, cada uma conferindo quem pode fazê-la (ver a divisa no cabeçalho de lá).

// ProposeAttack guarda o ataque rolado, sem tocar em PV nenhum.
//
// No máximo UM por sessão: dois provisórios simultâneos são duas verdades sobre a mesma
// cena, e a mesa não teria como saber qual confirmar.
func ProposeAttack(st *SessionRuntimeState, attack PendingAttack) error {
	if err := RefusesANewAttack(st); err != nil {
		return err
	}
	if FindEntryIndex(st, attack.AttackerEntryID) < 0 {
		return fmt.Errorf("quem ataca (%s) não está na fila", attack.AttackerEntryID)
	}
	// O ALVO TEM DUAS ESPÉCIES, e só uma delas está na fila: criatura tem linha
	// porque tem turno, e OBJETO não (p239). Conferir a fila dos dois recusaria
	// todo ataque a cenário — e exigir um dos dois é o que impede o provisório
	// sem alvo nenhum, que a faixa desenharia como "acertou" o quê.
	if attack.TargetTokenID == "" && FindEntryIndex(st, attack.TargetEntryID) < 0 {
		return fmt.Errorf("o alvo (%s) não está na fila", attack.TargetEntryID)
	}
	if attack.TargetTokenID != "" && attack.TargetEntryID != "" {
		return fmt.Errorf("o ataque veio com linha (%s) E peça (%s) como alvo, e são "+
			"endereços da mesma coisa: um deles está errado",
			attack.TargetEntryID, attack.TargetTokenID)
	}
	st.PendingAttack = &attack
	return nil
}

// RefusesANewAttack recusa o golpe novo enquanto o anterior não acabou. A PROPOSTA É
// IMUTÁVEL, e a próxima só existe depois da resposta do mestre (decisão do dono). O erro
// dele já encerrou o golpe, e é o único que o golpe novo substitui; o acerto ainda espera
// o dano e o ✓.
func RefusesANewAttack(st *SessionRuntimeState) error {
	if attackIsOpen(st) {
		return fmt.Errorf("já há um ataque em andamento na mesa: o próximo abre depois da resposta do mestre")
	}
	return nil
}

func attackIsOpen(st *SessionRuntimeState) bool {
	return st != nil && st.PendingAttack != nil && st.PendingAttack.Ruling != RulingMiss
}

// ReplaceAttackRoll troca o dado da proposta pelo do "rolar de novo" (p226), que é pago
// e vale antes da decisão. É a única mudança que a proposta aceita, e ela traz exatamente
// um rolar de novo a mais: outra aba pode ter rolado depois da leitura de quem pediu, e
// gravar por cima apagaria um PM já pago da lista que o cancelamento devolve.
func ReplaceAttackRoll(st *SessionRuntimeState, attack PendingAttack) error {
	open := st.PendingAttack
	if open == nil {
		return fmt.Errorf("não há ataque proposto para rolar de novo")
	}
	if open.Ruling != "" {
		return fmt.Errorf("o mestre já decidiu este ataque, e rolar de novo vale antes da decisão (p226)")
	}
	if len(attack.Rerolls) != len(open.Rerolls)+1 {
		return fmt.Errorf("o ataque já foi rolado de novo %d vez(es) enquanto este pedido chegava: olhe o dado novo antes de pagar outra",
			len(open.Rerolls))
	}
	// O QUE JÁ FOI PAGO vem da mesa, e não de quem leu a proposta antes.
	attack.Paid = open.Paid
	st.PendingAttack = &attack
	return nil
}

// HoldsTheCharacter diz se o ataque na mesa é deste personagem e ainda não acabou.
// O erro já encerrou o golpe (ver `RuleAttack`): o provisório dele só fica para o
// jogador ler a decisão.
func HoldsTheCharacter(st *SessionRuntimeState, characterID int64) bool {
	if !attackIsOpen(st) {
		return false
	}
	at := FindEntryIndex(st, st.PendingAttack.AttackerEntryID)
	return at >= 0 && st.Initiative[at].CharacterID != nil && *st.Initiative[at].CharacterID == characterID
}

// AttackToCommit responde "esta pessoa pode confirmar este ataque agora?" e
// devolve o provisório — sem aplicar nada.
//
// SÓ O MESTRE, e é a mesma frase do `CommitMove`: o que o jogador rolou é um
// rascunho para a mesa ver, e quem diz que aconteceu é quem toca a cena.
//
// # Por que ela NÃO tira o PV, sendo essa a consequência inteira do gesto
//
// Porque o regime não sabe de onde o PV do alvo sai. Quando há uma FICHA atrás
// da linha, quem manda é a ficha — o dano drena PV temporário, e a fila
// ESPELHA o resultado; quando é um NPC, o rastreador é o próprio registro. Os
// dois caminhos moram no store, que tem a porta da ficha, e é o
// `DeltaVitals` que já os separa.
//
// Aplicar aqui daria o caso do NPC certo e o do PC errado EM SILÊNCIO: a linha
// da fila mostraria o dano e a ficha do jogador continuaria cheia, que é
// exatamente a divergência que a fila espelhada existe para não ter.
func AttackToCommit(st *SessionRuntimeState, who Attacker) (PendingAttack, error) {
	attack := st.PendingAttack
	if attack == nil {
		return PendingAttack{}, fmt.Errorf("não há ataque proposto para confirmar")
	}
	if who.Role != "gm" {
		return PendingAttack{}, fmt.Errorf("só o mestre põe o dano na ficha: o seu ataque é um rascunho para a mesa ver")
	}
	// O GOLPE PASSA PELOS DOIS TEMPOS antes do ✓: a decisão do mestre e o dano
	// que ela pede. A manobra não tem dano, e o ✓ dela é a decisão.
	if attack.Maneuver == nil {
		switch {
		case attack.Ruling == RulingMiss:
			return PendingAttack{}, fmt.Errorf("o mestre já disse que este ataque errou: não há dano a aplicar")
		case attack.Ruling == "":
			return PendingAttack{}, fmt.Errorf("decida antes se o ataque errou, acertou ou foi crítico: o dano só existe depois")
		case !attack.DamageRolled:
			return PendingAttack{}, fmt.Errorf("o dano ainda não foi rolado")
		}
	}
	// O ALVO É CONFERIDO DE NOVO: entre rolar e confirmar ele pode ter saído da
	// fila, e o que vale é a mesa no instante em que o PV muda.
	//
	// SÓ O ALVO-CRIATURA, porque é só ele que tem fila de onde sair. O OBJETO
	// (p239) é conferido pelo tabuleiro, por quem escreve o PV dele — e medi-lo
	// aqui recusaria todo ataque a cenário com uma frase sobre uma fila em que
	// ele nunca esteve.
	if attack.TargetTokenID == "" && FindEntryIndex(st, attack.TargetEntryID) < 0 {
		return PendingAttack{}, fmt.Errorf("o alvo saiu da fila entre a rolagem e a confirmação")
	}
	return *attack, nil
}

// ClearPendingAttack tira o provisório da mesa depois do ✓. O ataque que o
// mestre disse ERRADO não passa por aqui: ele fica com a decisão para o jogador
// ler (`RuleAttack`), e sai quando a vez muda, quando o próximo ataque o
// substitui ou quando quem atacou o fecha.
func ClearPendingAttack(st *SessionRuntimeState) { st.PendingAttack = nil }

// CancelAttack descarta o provisório. O cancelamento do mestre é uma resposta: ele
// cancela por qualquer um, e é quem toca a mesa quando o jogador caiu da rede.
//
// O jogador só fecha o erro dele, que já acabou. Retirar a proposta sem resposta seria
// um d20 novo de graça: propor, ler o dado, cancelar e rolar de novo.
func CancelAttack(st *SessionRuntimeState, who Attacker) error {
	attack := st.PendingAttack
	if attack == nil {
		return fmt.Errorf("não há ataque proposto para cancelar")
	}
	if who.Role != "gm" && attack.ByUserID != who.UserID {
		return fmt.Errorf("o ataque proposto não é seu")
	}
	if who.Role != "gm" && attack.Ruling != RulingMiss {
		return fmt.Errorf("a proposta de ataque só sai da mesa com a resposta do mestre")
	}
	st.PendingAttack = nil
	return nil
}

// RuleAttack guarda a decisão do mestre sobre o golpe: erro, acerto ou crítico.
// No acerto e no crítico, `toRoll` é o dano que o jogador vai rolar.
//
// Uma vez só: a decisão muda o que o jogador rola, e trocá-la depois de ele
// pegar os dados é a mesa discordando dela mesma. Quem decidiu errado cancela, e
// o jogador propõe de novo.
func RuleAttack(st *SessionRuntimeState, who Attacker, ruling string, toRoll *DamageToRoll) error {
	attack := st.PendingAttack
	if attack == nil {
		return fmt.Errorf("não há ataque na mesa para decidir")
	}
	if who.Role != "gm" {
		return fmt.Errorf("quem decide se o ataque acertou é o mestre")
	}
	if attack.Maneuver != nil {
		return fmt.Errorf("a manobra não rola dano: confirme ou cancele")
	}
	if attack.Ruling != "" {
		return fmt.Errorf("o mestre já decidiu este ataque (%s)", attack.Ruling)
	}
	switch ruling {
	case RulingMiss:
		attack.Ruling, attack.Hit, attack.Critical = RulingMiss, false, false
	case RulingHit, RulingCritical:
		if toRoll == nil {
			return fmt.Errorf("o %s veio sem o dano a rolar", ruling)
		}
		attack.Ruling, attack.Hit, attack.Critical = ruling, true, ruling == RulingCritical
		attack.DamageToRoll = toRoll
	default:
		return fmt.Errorf("a decisão %q não existe: é %q, %q ou %q", ruling, RulingMiss, RulingHit, RulingCritical)
	}
	return nil
}

// AttackAwaitingDamage responde "esta pessoa pode rolar o dano agora?" e
// devolve o provisório. Rola quem atacou, e o mestre por ele — é quem toca a
// mesa quando o jogador caiu da rede.
func AttackAwaitingDamage(st *SessionRuntimeState, who Attacker) (PendingAttack, error) {
	attack := st.PendingAttack
	if attack == nil {
		return PendingAttack{}, fmt.Errorf("não há ataque na mesa para rolar o dano")
	}
	switch {
	case attack.Ruling == RulingMiss:
		return PendingAttack{}, fmt.Errorf("o mestre disse que o ataque errou: não há dano")
	case attack.Ruling == "":
		return PendingAttack{}, fmt.Errorf("o dano vem depois de o mestre dizer se o ataque acertou")
	case attack.DamageRolled:
		return PendingAttack{}, fmt.Errorf("o dano deste ataque já foi rolado")
	case who.Role != "gm" && attack.ByUserID != who.UserID:
		return PendingAttack{}, fmt.Errorf("quem rola o dano é quem atacou")
	}
	return *attack, nil
}

// RecordAttackDamage põe o dano rolado no provisório, conferindo DE NOVO que
// ele ainda espera o dano: entre a leitura e a gravação, o mesmo dano pode ter
// sido rolado por outra aba.
func RecordAttackDamage(st *SessionRuntimeState, who Attacker, rolled PendingAttack) error {
	if _, err := AttackAwaitingDamage(st, who); err != nil {
		return err
	}
	rolled.DamageRolled = true
	st.PendingAttack = &rolled
	return nil
}
