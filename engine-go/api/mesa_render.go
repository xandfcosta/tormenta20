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
func renderMesaFragment(view mesaView) ([]byte, error) {
	var buf bytes.Buffer
	if err := mesaTemplates.ExecuteTemplate(&buf, "mesa", view); err != nil {
		return nil, fmt.Errorf("render do fragmento da mesa: %w", err)
	}
	return buf.Bytes(), nil
}

// patchElementsEvent embrulha um fragmento no evento SSE do Datastar.
//
// O formato é por LINHA: cada linha do HTML vai num `data: elements ` próprio, e
// o cliente as junta de volta com "\n" (confirmado lendo o bundle v1.0.2, que
// corta cada linha no primeiro espaço e reagrupa por chave). Mandar HTML com
// quebra de linha num `data:` só produziria um evento truncado em silêncio —
// que é o pior jeito de errar isto.
//
// Dois "\n" finais fecham o evento; um só o deixa pendurado no buffer do
// cliente até o próximo chegar.
func patchElementsEvent(fragment []byte) string {
	var sb strings.Builder
	sb.WriteString("event: datastar-patch-elements\n")
	for _, linha := range strings.Split(strings.TrimRight(string(fragment), "\n"), "\n") {
		sb.WriteString("data: elements ")
		sb.WriteString(linha)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

// patchSignalsEvent manda sinais de volta ao cliente — o caminho pelo qual a
// RECUSA do servidor chega à tela.
//
// Isto fecha, dentro do piloto, um buraco que a ALE-213 deixou anotado: no
// socket o cliente NÃO escuta o `exception`, então uma mutação recusada some em
// silêncio e o jogador clica olhando para uma tela que não muda. Aqui a resposta
// do POST é o próprio caminho de volta, e não há como esquecer de ouvi-la.
func patchSignalsEvent(json string) string {
	return "event: datastar-patch-signals\ndata: signals " + json + "\n\n"
}
