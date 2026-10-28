package api

import (
	"net/http"
	"strings"

	"t20engine/serve/web/ui"
)

// O MODO DE COR é escolha da PESSOA, e mora num cookie (ALE-479).
//
// Cookie e não `localStorage`: quem desenha a página é o servidor, e ele só
// consegue entregar o `<html>` já no modo certo se a escolha viajar com o
// pedido. Decidir no navegador desenharia um quadro no modo errado antes de
// trocar — o "piscar" que todo tema escuro feito em JavaScript tem.
//
// Ele é ANÔNIMO: a porta também veste o sistema, e quem ainda não entrou tem
// tanto direito ao escuro quanto quem entrou.
const colorModeCookie = "color_mode"

// colorModeOf lê a escolha do pedido. Sem cookie, ou com um valor que não é um
// dos dois modos, é "siga o sistema operacional".
func colorModeOf(r *http.Request) ui.ColorMode {
	c, err := r.Cookie(colorModeCookie)
	if err != nil {
		return ui.ColorModeSystem
	}
	return ui.ParseColorMode(c.Value)
}

// handleColorMode grava a escolha e volta para a tela de onde ela veio.
//
// É um FORMULÁRIO de verdade (`POST /modo`, campos `modo` e `voltar`), e não um
// gesto do Datastar: trocar o modo repinta a página inteira, e um formulário
// faz isso sem JavaScript e sem remendo nenhum para dar errado.
//
// `modo=sistema` APAGA o cookie, em vez de gravar "sistema": quem volta a
// seguir o sistema operacional não tem escolha guardada, e é isso que o
// `ColorModeSystem` vazio já diz.
func (s *Server) handleColorMode(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	cookie := &http.Cookie{
		Name:     colorModeCookie,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	switch raw := r.PostFormValue("modo"); raw {
	case "sistema":
		cookie.MaxAge = -1
	case "claro", "escuro":
		cookie.Value = string(formModes[raw])
		cookie.MaxAge = 365 * 24 * 60 * 60
	default:
		http.Error(w, `modo "`+raw+`" desconhecido: use "claro", "escuro" ou "sistema"`, http.StatusBadRequest)
		return
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, localReturn(r.PostFormValue("voltar")), http.StatusSeeOther)
}

// formModes traduz o campo do formulário para o valor do cookie. O formulário
// fala português, como os da porta (`senha`, `destino`); o cookie é fronteira
// lida por código, e fala inglês.
var formModes = map[string]ui.ColorMode{
	"claro":  ui.ColorModeLight,
	"escuro": ui.ColorModeDark,
}

// localReturn só aceita volta para DENTRO do app. Um `voltar` vindo do
// formulário é texto do cliente, e redirecionar para ele sem conferir é o
// redirecionamento aberto de manual: `//outro.site` também começa com barra.
func localReturn(raw string) string {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/\\") {
		return "/"
	}
	return raw
}
