package table

// Os guardas das ABAS DE TABULEIRO (ALE-205).
//
// A regra que eles prendem não é "existe um segundo tabuleiro" — isso o store
// prova sozinho. É a separação que a issue existe para criar: **o que o mestre
// olha e o que a mesa vê deixaram de ser a mesma coisa**, e cada pessoa escolhe.
// Toda a família de defeito desta issue mora aí: um gesto de uma pessoa mexendo
// na tela das outras, ou um comando pousando na cena errada.

// openSecond põe outra cena na mesa e devolve as duas, na ordem de abertura.
