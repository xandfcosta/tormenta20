package live

// O TESTE ROLADO NA MESA (p220-221).
//
// Ele é o irmão informativo do `PendingAttack`: mesma faixa, mesma forma de
// conta, e NENHUM verbo — um teste não muda PV, condição nem turno, então não há
// o que confirmar nem o que cancelar. Guardá-lo com a mesma cerimônia do ataque
// daria à mesa dois botões que não fazem nada.

// SkillTestRoll é um teste como a mesa o lê.
//
// OS NÚMEROS VÊM CRUS e não como o tipo do motor, e isto não é escolha: o
// `TestTheLiveRuntimeDoesNotKnowTheOtherContexts` proíbe o regime de importar o
// `domain/engine`, e com razão — o regime é o que a mesa GUARDA, e amarrá-lo ao
// motor faria uma mudança de regra mexer no formato persistido. O
// `PendingAttack` ao lado carrega `Roll`, `Total` e `Damage` pelo mesmo motivo.
// A conversão acontece UMA vez, na borda que rola.
type SkillTestRoll struct {
	// Who é o nome de quem rolou, e ele viaja PRONTO.
	//
	// O ataque tira o nome da fila a cada desenho, porque a linha pode ter sido
	// renomeada entre rolar e mostrar. Aqui não dá: um teste sai da FICHA e o
	// personagem pode nem estar na fila — a mesa fora de combate rola testes o
	// tempo todo.
	Who   string `json:"who"`
	Skill string `json:"skill"`
	// A CONTA inteira, e não só o total: a mesa lê "d20 14 + 7 = 21" e não "21".
	Roll     int `json:"roll"`
	Modifier int `json:"modifier"`
	Total    int `json:"total"`
	// Natural20 e Natural1 são do DADO e nunca os dois (p221).
	Natural20 bool `json:"natural20,omitempty"`
	Natural1  bool `json:"natural1,omitempty"`
	// ByHand: o d20 veio da MESA, não do servidor.
	//
	// A faixa DIZ isso, e não é enfeite: sem a marca ninguém sabe se o número
	// saiu do app ou do dado que rolou na mesa, e as duas coisas têm confianças
	// diferentes — uma a mesa audita olhando, a outra não.
	ByHand bool `json:"byHand,omitempty"`
	// ByUserID é quem rolou.
	ByUserID int64 `json:"byUserId"`
}

// RecordSkillTest guarda o teste rolado, substituindo o anterior.
//
// SEM VALIDAÇÃO DE FILA, ao contrário do `ProposeAttack`: quem rola um teste não
// precisa estar em combate, e exigir linha na fila proibiria o teste de
// Percepção que abre a cena.
func RecordSkillTest(st *SessionRuntimeState, roll SkillTestRoll) error {
	st.LastSkillTest = &roll
	return nil
}
