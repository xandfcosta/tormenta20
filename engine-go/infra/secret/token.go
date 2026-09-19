// Package secret é o sorteio de um segredo que alguém vai receber por link.
//
// Ele existe porque a MESMA função nasceu duas vezes: o convite de campanha e o
// convite de conta cunham os dois um token aleatório e urlsafe, e as duas
// cópias tinham o mesmo corpo. Duas cópias de um gerador de segredo divergem na
// primeira vez que uma delas for "melhorada", e a que ficar para trás continua
// parecendo certa.
//
// Ele é INFRAESTRUTURA e não domínio: o que ele carrega é uma decisão de
// segurança — a fonte de aleatoriedade e o tamanho —, e nenhuma regra do livro
// tem opinião sobre isso.
package secret

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// sizeInBytes é o tamanho do segredo antes de ele virar texto.
//
// Vinte e quatro bytes dão 32 caracteres em base64 urlsafe, e o número é o que
// as duas cópias já usavam. Quem adivinha o token entra: é por isso que ele é
// grande e vem do `crypto/rand`.
const sizeInBytes = 24

// Token sorteia um segredo urlsafe.
//
// A FALHA SOBE, e é a diferença para as cópias que ela substitui: uma delas
// fazia `_, _ = rand.Read(b)`, e uma leitura falha teria gravado um token de
// vinte e quatro zeros — o mesmo para toda mesa aberta naquele instante. Não
// acontece no Linux; custa uma linha garantir.
func Token() (string, error) {
	b := make([]byte, sizeInBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("sortear um segredo de %d bytes: %w", sizeInBytes, err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
