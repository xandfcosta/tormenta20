package api

import (
	"net/http"
	"net/url"
)

// OS ENDEREÇOS ANTIGOS (ALE-272, fatia 10).
//
// Cada cena que saiu da SPA deixou um endereço para trás, e alguém pode tê-lo
// guardado: o link da campanha mandado no grupo, o favorito do grimório, o
// e-mail de convite que leva a `/register?convite=…`. Endereço publicado não se
// apaga — se ele deixar de responder, a culpa cai na mesa e não na migração.
//
// Enquanto a SPA existiu, quem desviava era ELA: cada rota portada virou uma
// casca com `entregaAPorta` no `beforeLoad`. Isso custava caro e, pior, morria
// junto com o `git rm` — o desvio em JavaScript obriga o navegador a BAIXAR o
// aplicativo inteiro para sair dele, e os endereços antigos são a única coisa
// da SPA que precisa sobreviver a ela.
//
// A tabela abaixo é a MESMA lista das cascas, lida uma a uma. O que cada uma
// preservava de busca (`?tab=`, `?token=`, `?convite=`, `?redirect=`) está
// preservado aqui, porque é justamente o parâmetro que faz o endereço valer:
// um convite sem token é uma tela de erro.

// legacyAddress é um endereço que a SPA atendia e para onde ele leva hoje.
type legacyAddress struct {
	// Padrao é o do `http.ServeMux` (Go 1.22): `/campaigns/{id}` captura o
	// segmento, e literal ganha de curinga na hora de casar — por isso
	// `/campaigns/new` não cai no `{id}`.
	Padrao string
	// Destino monta o endereço novo a partir do pedido.
	Destino func(*http.Request) string
}

// legacyAddresses é a tabela inteira, na ordem em que as fatias portaram.
//
// TRÊS SAÍRAM na ALE-280, quando as cenas subiram para a raiz: `/admin`,
// `/grimorio` e `/redefinir-senha` eram endereços que a SPA e o piloto
// escreviam IGUAIS, e sem o prefixo `/piloto` o destino virou a própria origem.
// No mux isso não é uma entrada inútil — é um laço: o padrão literal ganha do
// `"/"` das cenas, então o desvio responderia 302 para si mesmo para sempre, e a
// tela nunca apareceria. Quem atende esses três agora é o roteador das cenas,
// direto.
var legacyAddresses = []legacyAddress{
	{"/campaigns", fixed("/campanhas")},
	{"/campaigns/{$}", fixed("/campanhas")},
	{"/campaigns/new", fixed("/campanhas/nova")},
	{"/campaigns/join", withSearch("/campanhas/entrar", "token")},
	{"/campaigns/{id}", withSegment("/campanhas/", "id", "tab")},
	{"/characters", fixed("/personagens")},
	{"/characters/{$}", fixed("/personagens")},
	// As TRÊS últimas entraram na fatia 10c, com o `git rm` da SPA: enquanto ela
	// existia, estes eram os endereços das telas que ainda viviam lá — a ficha,
	// a forja e a sessão ao vivo. Desviá-los antes teria tornado a tela antiga
	// inalcançável enquanto ela ainda era a única.
	{"/characters/new", fixed("/personagens/nova")},
	{"/characters/new/{passo}", fixed("/personagens/nova")},
	{"/characters/{id}", withSegment("/personagens/", "id", "tab")},
	{"/campaigns/{id}/sessions/{sid}", func(r *http.Request) string {
		return "/mesa/" + url.PathEscape(r.PathValue("id")) + "/" + url.PathEscape(r.PathValue("sid"))
	}},
	{"/gm", fixed("/mestre/bestiario")},
	{"/gm/{$}", fixed("/mestre/bestiario")},
	{"/gm/{tool}", withSegment("/mestre/", "tool")},
	{"/login", withSearch("/entrar", "redirect")},
	{"/register", withSearch("/criar-conta", "convite")},
	// `/join/{token}` era um desvio DUPLO na SPA: ela mandava para
	// `/campaigns/join?token=…`, que por sua vez mandava para o piloto. Aqui ele
	// vai direto — dois saltos existiam porque eram duas rotas dela, não porque
	// alguém precisava passar pelo meio.
	{"/join/{token}", func(r *http.Request) string {
		return "/campanhas/entrar?token=" + url.QueryEscape(r.PathValue("token"))
	}},
}

// MountLegacyAddresses registra os desvios no mux da RAIZ.
//
// Na raiz e não sob `/` porque é lá que eles moravam: quem tem o link
// antigo digita `/grimorio`, e não `/grimorio`.
func MountLegacyAddresses(mux *http.ServeMux) {
	for _, endereco := range legacyAddresses {
		destino := endereco.Destino
		mux.HandleFunc(endereco.Padrao, func(w http.ResponseWriter, r *http.Request) {
			// 302 e não 301, e a diferença importa numa migração: o permanente
			// fica GRAVADO no navegador de quem visitou uma vez, e voltar atrás
			// depois disso exige limpar o cache de cada pessoa da mesa. É a
			// mesma escolha que o desvio da raiz já fazia.
			http.Redirect(w, r, destino(r), http.StatusFound)
		})
	}
}

// fixed é o destino que não depende do pedido.
func fixed(destino string) func(*http.Request) string {
	return func(*http.Request) string { return destino }
}

// withSearch leva só as chaves de busca nomeadas, e não a query inteira: o que
// não estava na casca da SPA não passa a valer agora por acidente.
func withSearch(destino string, chaves ...string) func(*http.Request) string {
	return func(r *http.Request) string {
		return destino + preservedSearch(r, chaves)
	}
}

// withSegment acrescenta um segmento capturado do caminho.
func withSegment(prefixo, nomeDoSegmento string, chaves ...string) func(*http.Request) string {
	return func(r *http.Request) string {
		return prefixo + url.PathEscape(r.PathValue(nomeDoSegmento)) + preservedSearch(r, chaves)
	}
}

// preservedSearch devolve "?a=1&b=2" só com as chaves pedidas que vieram
// preenchidas, ou "" quando nenhuma veio.
func preservedSearch(r *http.Request, chaves []string) string {
	guardadas := url.Values{}
	for _, chave := range chaves {
		if valor := r.URL.Query().Get(chave); valor != "" {
			guardadas.Set(chave, valor)
		}
	}
	if len(guardadas) == 0 {
		return ""
	}
	return "?" + guardadas.Encode()
}
