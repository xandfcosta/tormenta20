// Package httpio é a borda HTTP sem regra nenhuma: escrever resposta, ler
// corpo, e comprimir.
//
// Ele conhece `net/http`, e é por isso que ele NÃO é alcançado pelo `domain` —
// o que a regra precisa (a recusa por campo, os leitores de corpo) mora no
// `infra/wire`, que é folha.
package httpio

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"t20engine/infra/wire"
)

// WriteJSON serializa o corpo com o status dado. Corpo nulo escreve só a linha
// de status (o caso do 204).
//
// O escape de HTML fica DESLIGADO: `<`, `>` e `&` guardados dentro das colunas
// de JSON (modificadores, escolhas) chegam ao cliente como foram gravados, e
// não como `<`.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(body)
	}
}

// WriteError emite o envelope mínimo: `{"statusCode","message"}`. Quando a falha
// também tem detalhe por campo, quem responde é o `WriteFieldError`.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]any{"statusCode": status, "message": message})
}

// WriteFieldError emite o envelope de validação inteiro, para o handler que já
// tem a mensagem e o detalhe na mão e não tem erro de domínio para embrulhar.
//
// Sem ele a forma rica só seria alcançável por um `*wire.FieldError`, incômodo
// de montar na linha — e aí o handler escreve o mapa à mão e o envelope diverge.
func WriteFieldError(w http.ResponseWriter, status int, message string, fields wire.FieldErrorMap) {
	WriteJSON(w, status, map[string]any{
		"statusCode":  status,
		"error":       http.StatusText(status),
		"message":     message,
		"fieldErrors": fields,
	})
}

// WriteDomainError é a ÚNICA costura por onde um handler traduz o par
// `(status, erro)` de uma regra: um `*wire.FieldError` vira o envelope de
// validação com a mensagem dele; qualquer outro vira `{message}` no status dado.
func WriteDomainError(w http.ResponseWriter, status int, err error) {
	var fe *wire.FieldError
	if errors.As(err, &fe) {
		WriteFieldError(w, fe.Status(), fe.Message(), fe.Fields())
		return
	}
	WriteError(w, status, err.Error())
}

// maxBodyBytes é o teto de um corpo de requisição. Um megabyte é folgado para
// tudo que este app manda — a ficha inteira de nível 20 serializada dá ~40 KB —
// e o que ele impede é um corpo sem fim segurando memória e goroutine.
const maxBodyBytes = 1 << 20

// DecodeJSON lê o corpo para `dst`; corpo malformado escreve 400 e devolve
// falso, para o handler desistir.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		// Corpo grande demais tem resposta PRÓPRIA: dizer "JSON inválido" para
		// um JSON perfeitamente válido mandaria quem escreveu o cliente
		// procurar defeito onde não há.
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			WriteError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("O corpo da requisição passa de %d bytes", tooLarge.Limit))
			return false
		}
		WriteError(w, http.StatusBadRequest, "Invalid JSON body")
		return false
	}
	return true
}

// WriteValidationError emite o envelope de validação para o caso simples.
func WriteValidationError(w http.ResponseWriter, fields wire.FieldErrorMap) {
	WriteJSON(w, http.StatusBadRequest, map[string]any{
		"statusCode":  http.StatusBadRequest,
		"error":       "Bad Request",
		"message":     "Validation failed",
		"fieldErrors": fields,
	})
}

// ParseInt lê um parâmetro numérico de rota, tolerando espaço em volta.
func ParseInt(s string) (int, error) { return strconv.Atoi(strings.TrimSpace(s)) }
