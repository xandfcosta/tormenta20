// Package wire é o vocabulário da FRONTEIRA sem nada de transporte: ler um
// campo do corpo decodificado e recusar um campo pelo nome.
//
// Ele é folha de propósito — o `domain` o alcança —, e é isso que separa a
// RECUSA (que é regra) de como ela é DESENHADA numa resposta HTTP (que é do
// `infra/httpio`).
package wire

// FieldErrorMap é a recusa por campo: nome do campo → as frases que a pessoa lê.
type FieldErrorMap map[string][]string

// FieldError é uma recusa de domínio que carrega detalhe por campo.
//
// Ela existe para uma regra sem transporte poder recusar com corpo rico sem
// conhecer `http.ResponseWriter`: quem fala HTTP desenha o envelope; outro
// transporte lê só o `Error()`.
type FieldError struct {
	status  int
	message string
	fields  FieldErrorMap
}

// NewFieldError é a ÚNICA porta de montagem, e os três campos ficam não
// exportados por isso: com a struct montada à mão em cada chamador, o
// compilador não tem como impedir um envelope pela metade.
func NewFieldError(status int, message string, fields FieldErrorMap) *FieldError {
	return &FieldError{status: status, message: message, fields: fields}
}

func (e *FieldError) Error() string { return e.message }

// Status, Message e Fields são LEITURA para quem desenha a resposta.
//
// Eles existem desde que o renderizador saiu daqui para o `infra/httpio`: campo
// não exportado não atravessa pacote. Ler é seguro — o que continua impossível
// é montar um envelope pela metade, porque só o construtor escreve.
func (e *FieldError) Status() int           { return e.status }
func (e *FieldError) Message() string       { return e.message }
func (e *FieldError) Fields() FieldErrorMap { return e.fields }
