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
	// Defense é a Defesa que o ataque enfrentou. Ela viaja porque a mesa lê a
	// COMPARAÇÃO — "24 vs 17" é o que explica o veredicto, e sem o 17 a faixa
	// afirma um acerto sem dizer contra o quê.
	Defense int `json:"defense"`
	// NonLethal é quanto do `Damage` não conta para sangrar nem para morrer
	// (p236) — tudo, quando a arma é Piedosa (p336). Viaja no provisório porque
	// quem aplica é a CONFIRMAÇÃO, e entre rolar e confirmar o ataque pode ser
	// cancelado.
	NonLethal int `json:"nonLethal,omitempty"`
	// Situations são os rótulos da Tabela 5-3 que valeram neste ataque, e a
	// faixa os escreve. Sem eles a mesa lê uma Defesa que a ficha não tem, ou
	// um ataque que bateu a Defesa e errou (p238), e vai procurar o defeito.
	Situations []string `json:"situations,omitempty"`
	Hit        bool     `json:"hit"`
	Critical   bool     `json:"critical"`
	Dice       []int    `json:"dice,omitempty"`
	Faces      int      `json:"faces,omitempty"`
	RawDamage  int      `json:"rawDamage"`
	Absorbed   int      `json:"absorbed"`
	Damage     int      `json:"damage"`
	// Maneuver é a conta da MANOBRA (p234), e ela é nula num golpe comum.
	//
	// Uma manobra É um ataque corpo a corpo — o livro abre a página dizendo isso
	// —, e por isso ela mora no MESMO provisório em vez de num irmão: a divisa de
	// quem propõe e quem confirma é a mesma, e os verbos da faixa também. Dois
	// provisórios simultâneos seriam duas verdades sobre a mesma cena, que é o
	// que o `ProposeAttack` já existe para não permitir.
	//
	// Num golpe, os campos de dano falam e este é nulo; numa manobra, o inverso.
	// É ele que diz à faixa qual das duas frases escrever.
	Maneuver *ManeuverRoll `json:"maneuver,omitempty"`
	// ByUserID é quem rolou. O mestre confirma por qualquer um; quem propôs
	// cancela o que é dele.
	ByUserID int64 `json:"byUserId"`
}

// ManeuverRoll é o teste OPOSTO de uma manobra, do jeito que a mesa lê.
//
// Os números de quem ATACA não se repetem aqui: o `Roll` e o `Total` do
// provisório já são dele. O que falta é o outro lado, e é isso que este tipo
// acrescenta — a mesma razão pela qual a `Defense` viaja num golpe: a mesa lê a
// COMPARAÇÃO, e um total sozinho não explica o veredito.
type ManeuverRoll struct {
	// Kind é a manobra do livro: agarrar, derrubar, desarmar, empurrar, quebrar.
	Kind string `json:"kind"`
	// OpposedRoll é o d20 NATURAL de quem se defende, e Opposed é o total dele
	// já com o Luta.
	//
	// Os dois viajam porque a mesa lê a CONTA: "14 (d20 8 +6) contra 12 (d20 4
	// +8)" diz quem teve sorte e quem tem perícia, e dois totais sozinhos não.
	// O natural também é o único sentinela honesto para "este lado foi
	// rolado" — um TOTAL pode ser zero de verdade, e é: um arcanista sem
	// proficiência na espada ataca com −5 (p142), e um d20 de 5 dá zero.
	OpposedRoll int `json:"opposedRoll"`
	Opposed     int `json:"opposed"`
	// Margin é a diferença, e ela é REGRA: cinco pontos ou mais dão efeito extra
	// ao derrubar e ao desarmar (p234). Negativa quando quem tentou perdeu.
	Margin int  `json:"margin"`
	Won    bool `json:"won"`
	// Imposes é a condição que a vitória deixa no alvo, e ela viaja no
	// provisório porque quem a aplica é a CONFIRMAÇÃO: entre rolar e confirmar
	// a manobra pode ser cancelada, e uma condição aplicada na proposta ficaria
	// no alvo de um gesto que não aconteceu.
	Imposes string `json:"imposes,omitempty"`
	// AnotherRoll é o empate que a página manda repetir — totais iguais E bônus
	// iguais. Não é derrota de quem tentou, e a faixa tem de dizer isso em vez
	// de anunciar um vencedor que a regra não deu.
	AnotherRoll bool `json:"anotherRoll,omitempty"`
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
func ProposeAttack(st *SessionRuntimeState, attack PendingAttack) error {
	if FindEntryIndex(st, attack.AttackerEntryID) < 0 {
		return fmt.Errorf("quem ataca (%s) não está na fila", attack.AttackerEntryID)
	}
	if FindEntryIndex(st, attack.TargetEntryID) < 0 {
		return fmt.Errorf("o alvo (%s) não está na fila", attack.TargetEntryID)
	}
	st.PendingAttack = &attack
	return nil
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
	// O ALVO É CONFERIDO DE NOVO: entre rolar e confirmar ele pode ter saído da
	// fila, e o que vale é a mesa no instante em que o PV muda.
	if FindEntryIndex(st, attack.TargetEntryID) < 0 {
		return PendingAttack{}, fmt.Errorf("o alvo saiu da fila entre a rolagem e a confirmação")
	}
	return *attack, nil
}

// ClearPendingAttack tira o provisório da mesa. Um ataque que ERRA também sai
// por aqui: errar é uma coisa que acontece, e o provisório tem de sumir da tela
// do mesmo jeito.
func ClearPendingAttack(st *SessionRuntimeState) { st.PendingAttack = nil }

// CancelAttack descarta o provisório sem mexer em ninguém. O mestre cancela por
// qualquer um — é ele quem toca a mesa quando o jogador caiu da rede —, e o
// jogador só o que ele mesmo rolou.
func CancelAttack(st *SessionRuntimeState, who Attacker) error {
	attack := st.PendingAttack
	if attack == nil {
		return fmt.Errorf("não há ataque proposto para cancelar")
	}
	if who.Role != "gm" && attack.ByUserID != who.UserID {
		return fmt.Errorf("o ataque proposto não é seu")
	}
	st.PendingAttack = nil
	return nil
}
