package api

// campanhaNovaView é o formulário da folha em branco (ALE-246).
//
// Ele carrega de volta o que a pessoa digitou, e isso é o mínimo: recusar um
// nome de 121 caracteres e devolver a folha VAZIA faria perder a descrição
// inteira junto — que é o campo caro de reescrever.
type campanhaNovaView struct {
	Nome      string
	Descricao string
	Erros     FieldErrorMap
	// Aviso é a recusa do formulário inteiro, quando nenhum campo é dono do
	// problema. Mesma divisão da porta.
	Aviso string
}
