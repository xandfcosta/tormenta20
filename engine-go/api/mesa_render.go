package api

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
)

// Os arquivos do piloto Datastar (ALE-219). Embutidos, como os catálogos: o
// binário continua sendo UM arquivo, que é a premissa de produção deste
// projeto.
//
//go:embed mesa/tmpl/*.html mesa/static/*
var mesaFS embed.FS

// mesaTemplates é parseado UMA vez no init: um erro de sintaxe de template é
// erro de programação, e descobri-lo na primeira requisição de um jogador em
// vez de no boot é o defeito que o `assertSchema` já ensinou a não repetir
// (ALE-154).
var mesaTemplates = template.Must(
	template.New("mesa").Funcs(mesaFuncs).ParseFS(mesaFS, "mesa/tmpl/*.html"),
)

// mesaFuncs — só o `dict`, e só porque `html/template` não sabe passar dois
// valores para um sub-template. Nenhuma REGRA mora numa função de template: o
// que decide mora em `mesa_view.go`, onde se testa sem HTML.
var mesaFuncs = template.FuncMap{
	"dict": func(pares ...any) (map[string]any, error) {
		if len(pares)%2 != 0 {
			return nil, fmt.Errorf("dict quer pares chave/valor, recebeu %d argumentos", len(pares))
		}
		out := make(map[string]any, len(pares)/2)
		for i := 0; i < len(pares); i += 2 {
			chave, ok := pares[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict: chave %d não é string, é %T", i, pares[i])
			}
			out[chave] = pares[i+1]
		}
		return out, nil
	},
}

// renderMesaPage escreve o documento inteiro — a carga fria.
func renderMesaPage(view mesaView) ([]byte, error) {
	var buf bytes.Buffer
	if err := mesaTemplates.ExecuteTemplate(&buf, "page", view); err != nil {
		return nil, fmt.Errorf("render da página da mesa: %w", err)
	}
	return buf.Bytes(), nil
}

// renderMesaFragment escreve só o <main id="mesa"> — o que viaja no SSE.
//
// Aparado nas pontas porque o `{{define}}` deixa uma quebra de linha antes e
// depois, e cada uma delas vira uma linha `data: elements ` VAZIA no fio, a cada
// quadro, para sempre.
func renderMesaFragment(view mesaView) ([]byte, error) {
	var buf bytes.Buffer
	if err := mesaTemplates.ExecuteTemplate(&buf, "mesa", view); err != nil {
		return nil, fmt.Errorf("render do fragmento da mesa: %w", err)
	}
	return []byte(strings.TrimSpace(buf.String())), nil
}
