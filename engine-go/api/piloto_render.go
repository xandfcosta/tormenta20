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
//go:embed piloto/tmpl/*.html piloto/static/*
var pilotoFS embed.FS

// pilotoTemplates é parseado UMA vez no init: um erro de sintaxe de template é
// erro de programação, e descobri-lo na primeira requisição de um jogador em
// vez de no boot é o defeito que o `assertSchema` já ensinou a não repetir
// (ALE-154).
var pilotoTemplates = template.Must(
	template.New("piloto").Funcs(pilotoFuncs).ParseFS(pilotoFS, "piloto/tmpl/*.html"),
)

// pilotoFuncs — só o `dict`, e só porque `html/template` não sabe passar dois
// valores para um sub-template. Nenhuma REGRA mora numa função de template: o
// que decide mora em `mesa_view.go`, onde se testa sem HTML.
var pilotoFuncs = template.FuncMap{
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

// paginaPiloto é a casca de uma tela do piloto: o que o layout precisa saber
// para montar o `<head>` e o `<body>` em volta do corpo já renderizado.
//
// O corpo chega PRONTO, como `template.HTML`, porque o `html/template` não sabe
// invocar um sub-template por nome dinâmico. Render em dois passos é a saída, e
// ela é honesta: o corpo passou pelo escape do próprio template dele, então o
// que o layout recebe já é HTML confiável — e não texto de usuário.
type paginaPiloto struct {
	// Titulo é o do `<title>`; TituloVisivel é o do cabeçalho da cena. Vazio
	// significa "sem cabeçalho" — a Mesa não tem, porque a faixa AO VIVO dela já
	// é o cabeçalho e duas fileiras de cromo tirariam o palco do combate.
	Titulo        string
	TituloVisivel string
	Voltar        string
	Sinais        string
	Init          string
	Corpo         template.HTML
}

// renderPagina embrulha um corpo já renderizado no layout comum.
func renderPagina(p paginaPiloto) ([]byte, error) {
	var buf bytes.Buffer
	if err := pilotoTemplates.ExecuteTemplate(&buf, "layout", p); err != nil {
		return nil, fmt.Errorf("render do layout do piloto: %w", err)
	}
	return buf.Bytes(), nil
}

// renderFragmento escreve UM template pelo nome.
//
// Aparado nas pontas porque o `{{define}}` deixa uma quebra de linha antes e
// depois, e cada uma delas vira uma linha `data: elements ` VAZIA no fio, a
// cada quadro, para sempre.
func renderFragmento(nome string, view any) ([]byte, error) {
	var buf bytes.Buffer
	if err := pilotoTemplates.ExecuteTemplate(&buf, nome, view); err != nil {
		return nil, fmt.Errorf("render do fragmento %q: %w", nome, err)
	}
	return []byte(strings.TrimSpace(buf.String())), nil
}
