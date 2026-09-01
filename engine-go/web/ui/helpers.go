package ui

import (
	"strconv"
	"strings"
)

// Os auxiliares dos componentes templ (ALE-229). Aqui e não no `.templ` porque
// nenhum deles escreve CLASSE — o scanner do Tailwind varre só os `.templ`, e é
// por isso que o `classesDoBotao` teve de ficar lá.

// juntar cola pedaços de lista de classes ignorando os vazios, que é o `cn` da
// SPA sem o `tailwind-merge`: aqui não há sobreposição para resolver, porque
// quem monta a lista é o componente e não o chamador.
func Join(partes ...string) string {
	presentes := make([]string, 0, len(partes))
	for _, p := range partes {
		if p != "" {
			presentes = append(presentes, p)
		}
	}
	return strings.Join(presentes, " ")
}

func Int(n int) string { return strconv.Itoa(n) }

// tipoOuTexto existe porque `<input>` sem `type` é `text`, mas um atributo
// vazio no HTML não é: `type=""` faz o navegador cair no padrão por recuperação
// de erro, e depender de recuperação de erro é depender do navegador.
func InputType(tipo string) string {
	if tipo == "" {
		return "text"
	}
	return tipo
}
