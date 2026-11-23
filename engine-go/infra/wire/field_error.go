// Package wire é o vocabulário da FRONTEIRA sem nada de transporte: ler um
// campo do corpo decodificado e recusar um campo pelo nome.
//
// Ele é folha de propósito — o `domain` o alcança —, e é isso que separa a
// RECUSA (que é regra) de como ela é DESENHADA numa resposta HTTP (que é do
// `infra/httpio`).
package wire

// FieldErrorMap é a recusa por campo: nome do campo → as frases que a pessoa lê.
type FieldErrorMap map[string][]string
