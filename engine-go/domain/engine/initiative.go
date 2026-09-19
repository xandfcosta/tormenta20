package engine

// O BÔNUS DE INICIATIVA, lido da ficha COMPUTADA.
//
// Ele é o total da perícia **Iniciativa** — ½ nível + atributo + treino +
// itens —, e é regra do livro. Computá-lo na tela seria uma segunda
// implementação livre para divergir do motor: aqui é a MESMA decomposição que a
// ficha inteira mostra.

// initiativeExpertise é o nome da perícia no catálogo.
//
// Escrito UMA vez porque a string literal em dois lugares é como um typo
// sobrevive: a busca não acharia nada e devolveria zero em silêncio, e o
// jogador entraria na fila com o d20 pelado sem ninguém notar.
const initiativeExpertise = "Iniciativa"

// InitiativeTotal é o bônus que se soma ao d20.
//
// FICHA SEM A PERÍCIA devolve zero, e isso é resposta e não falha: ficha sem
// classe não tem perícia nenhuma computada, e recusar deixaria o jogador sem
// conseguir entrar na fila por causa de uma ficha incompleta — o que o mestre
// resolve na hora arrastando a ordem.
//
//	engine.InitiativeTotal(ficha) // 8, para o Arcanista Nv9 do oráculo
func InitiativeTotal(ficha ComputedSheetV2) int {
	for _, pericia := range ficha.Expertises {
		if pericia.Name == initiativeExpertise {
			return pericia.Total
		}
	}
	return 0
}
