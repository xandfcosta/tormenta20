package table

// A MEMÓRIA EFÊMERA DA MESA: o que o servidor guarda por sessão e que morre com
// o processo.
//
// São três — quem está com a LENTE acesa (`lens.go`), qual ABA cada pessoa
// está olhando (`tabs.go`) e o que a TELA DA MESA lembra (`display_state.go`)
// —, e cada um explica no arquivo dele por que não é sinal do navegador. O que
// este arquivo acrescenta é o FIM DA VIDA deles.

// EphemeralTableState junta os três para que o fim de uma sessão alcance
// todos, e é a razão de eles não nascerem mais dentro do `table.New`.
//
// Enquanto nasciam lá, o único jeito de esvaziá-los era o `endBoard` — e ele só
// os esvazia quando a ÚLTIMA cena fecha, que é certo para fechar uma aba e não
// cobre a sessão APAGADA. O hospedeiro monta este objeto antes do ciclo da
// sessão e o entrega aos dois, e assim a Mesa entra na lista de `memories` do
// `session.Lifecycle` como os stores (ALE-377).
//
// O vazamento não tinha sintoma na TELA, e é por isso que sobreviveu: quem lê a
// aba escolhida confere contra os tabuleiros abertos (`pullTab`), e quem lê a
// lente só acende botão de mestre. Era memória que nunca voltava.
type EphemeralTableState struct {
	lenses     *lenses
	chosenTabs *chosenTabs
	// displays é a TELA DA MESA, que é por sessão e não por pessoa — ver o
	// `display_state.go`. Mora aqui pela mesma razão das outras duas: a sessão
	// apagada tem de alcançá-la.
	displays *tableDisplays
}

func NewEphemeralTableState() *EphemeralTableState {
	return &EphemeralTableState{lenses: newLenses(), chosenTabs: newTabs(), displays: newTableDisplays()}
}

// SessionDeleted cumpre a porta `session.MemoryOfASession`: a sessão deixou de
// existir, e com ela qualquer motivo para lembrar quem estava olhando o quê.
func (e *EphemeralTableState) SessionDeleted(sessionID int64) {
	e.lenses.Erase(sessionID)
	e.chosenTabs.Erase(sessionID)
	e.displays.Erase(sessionID)
}

// Watching diz quantas entradas de memória efêmera esta sessão tem guardadas:
// uma por pessoa na lente e na aba, e as da tela da mesa.
//
// É a resposta com DENOMINADOR: um guarda que só afirmasse "não sobrou nada"
// depois de apagar não distinguiria o esquecimento de um caso que nunca guardou
// nada. Quem pergunta antes tem o número para comparar.
func (e *EphemeralTableState) Watching(sessionID int64) int {
	return e.lenses.HowMany(sessionID) + e.chosenTabs.HowMany(sessionID) + e.displays.HowMany(sessionID)
}
