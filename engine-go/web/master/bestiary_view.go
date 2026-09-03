package master

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"t20engine/book"
	"t20engine/web/bookui"

	"github.com/a-h/templ"
)

// O BESTIÁRIO como dado (ALE-257).
//
// O catálogo deixa de ir ao navegador, e aqui isso pesa mais que na cena de
// personagens: são 62 KB de criaturas que a SPA baixa para PODER filtrar. No
// servidor ele já está em memória por `go:embed`, e o que atravessa a rede é a
// lista filtrada.
//
// Some junto a LISTA VIRTUALIZADA, e a razão é medida e não gosto: o bestiário
// tem 80 criaturas. Virtualização é maquinaria para lista que não cabe, e a 80
// ela cobra o custo sem cobrir problema nenhum — inclusive o de precisar de e2e
// porque jsdom mede zero.

// BestiaryView é o que a cena precisa para se desenhar inteira, numa resposta.
//
// A SPA baixa o catálogo (62 KB) para PODER filtrar; aqui a filtragem já
// aconteceu e o que atravessa é o resultado. `Total` viaja junto porque a linha
// "12 de 80" precisa dos dois números, e sem ele a tela não sabe se o filtro
// apertou muito ou se o bestiário é pequeno.
type BestiaryView struct {
	// Base é o prefixo das rotas que ESTA cena chama, e existe porque o mesmo
	// desenho serve dois lugares: a cena do mestre em `/mestre/bestiario`
	// e o painel da Mesa em `/mesa/{c}/{s}/bestiario`. O que muda entre
	// as duas é o ENDEREÇO, não a lista nem o bloco — e um segundo desenho seria
	// a mesma criatura mantida em dois lugares.
	//
	// Sem valor não há rota: o `BestiaryBase` recusa a string vazia em vez de
	// deixar o botão apontar para a página atual, que é o defeito silencioso
	// desta forma — o clique "funciona" e recarrega a cena.
	Base string
	// Livro é o endereço do PDF do livro (ALE-264), e o ZERO VALOR é o caso
	// normal: sem `LIVRO_PDF` configurado não há livro para abrir e o bloco não
	// desenha o botão. Ele vem pedido no construtor, ao lado da Base, pela mesma
	// razão que ela — são os dois endereços de que a cena depende, e um deles
	// esquecido some em silêncio.
	Book    bookui.BookAddress
	Entries []book.Entry
	Total   int
	Chosen  *book.Entry
	Term    string
	Types   []string
	CRMin   float64
	CRMax   float64
	// Abrir diz que ESTE pedido veio de um clique numa linha, e por isso a ficha
	// tem de nascer aberta. Vem da URL e não de um sinal: a MESMA rota serve a
	// busca e os filtros de tipo, e os dois mandam os sinais TODOS — inclusive o
	// `criatura` já escolhido. Um sinal não separaria "escolhi esta criatura" de
	// "digitei uma letra com uma criatura já escolhida", e a busca passaria a
	// abrir a ficha sozinha a cada tecla.
	Open bool
}

// chosenOrFirst: a cena SEMPRE mostra um bloco quando há lista.
//
// A SPA faz `shown().find(...) ?? shown()[0]`, e a razão de portar isso é que o
// painel vazio ao lado de uma lista cheia parece defeito. Quando o filtro muda
// e a criatura escolhida sai da lista, cai na primeira em vez de esvaziar.
func chosenOrFirst(lista []book.Entry, id string) *book.Entry {
	if len(lista) == 0 {
		return nil
	}
	for i := range lista {
		if lista[i].ID == id {
			return &lista[i]
		}
	}
	return &lista[0]
}

// loadBestiary monta a cena a partir do que veio na URL ou nos sinais.
// LoadBestiaryFrom exige a BASE como primeiro parâmetro, e isso é a lição de
// um guarda que acusou na hora: a primeira versão deixava o campo de fora e um
// teste de outra pasta montou a cena sem ele. Construtor que consegue produzir
// valor inválido é o próprio defeito — pedir aqui torna o esquecimento
// impossível em vez de detectável.
func LoadBestiaryFrom(base string, livro bookui.BookAddress, busca string, tipos []string, ndMin, ndMax float64, escolhido string) BestiaryView {
	todos := book.Creatures()
	lista := book.FilterCreatures(todos, book.CreatureFilter{Busca: busca, Tipos: tipos, NDMin: ndMin, NDMax: ndMax})
	return BestiaryView{
		Base:    base,
		Book:    livro,
		Entries: lista,
		Total:   len(todos),
		Chosen:  chosenOrFirst(lista, escolhido),
		Term:    busca,
		Types:   tipos,
		CRMin:   ndMin,
		CRMax:   ndMax,
	}
}

// typeOn diz se o crachá está ligado, para a cena não precisar de `slices`.
func (v BestiaryView) typeOn(tipo string) bool {
	return slices.Contains(v.Types, tipo)
}

// crInBox escreve o número do campo sem o `.0` que o float traria: a caixa
// diz "3", não "3.0", e o passo do livro é de um quarto.
func crInBox(nd float64) string {
	return strconv.FormatFloat(nd, 'g', -1, 64)
}

// BestiarySignals: o estado da tela que o Datastar mantém no cliente.
//
// São só os quatro CRITÉRIOS mais a criatura aberta — nada de lista, nada de
// bloco. O que se vê chega desenhado; o que viaja de volta é o que o mestre
// escolheu.
func BestiarySignals(v BestiaryView) string {
	tipos, _ := json.Marshal(v.Types)
	if v.Types == nil {
		tipos = []byte("[]")
	}
	escolhida := ""
	if v.Chosen != nil {
		escolhida = v.Chosen.ID
	}
	busca, _ := json.Marshal(v.Term)
	criatura, _ := json.Marshal(escolhida)
	// `fichaAberta` sai DAQUI e não de um evento de sinal separado, e a razão é o
	// que a medição mostrou: este `data-signals` mora no `#bestiario`, que É o
	// elemento remendado, então ele REDECLARA os sinais a cada remendo. Um
	// evento de sinal mandado depois do conteúdo era desfeito por esta linha —
	// o fio levava `{"fichaAberta":true}` e o diálogo continuava `display:none`.
	//
	// É o mesmo perigo que o `web/ui/layout.templ` já tinha escrito ao pôr os
	// sinais da página no `<body>`, que nunca é remendado. Aqui a saída não é
	// mover: é o servidor redeclarar com o valor CERTO, e aí o conteúdo e o
	// estado de aberto chegam no MESMO remendo — atômicos, sem janela em que um
	// esteja aplicado e o outro não.
	return fmt.Sprintf(`{busca: %s, ndMin: %s, ndMax: %s, tipos: %s, criatura: %s, fichaAberta: %t}`,
		busca, crInBox(v.CRMin), crInBox(v.CRMax), tipos, criatura, v.Open)
}

// BestiaryBase é o prefixo de rota da cena, e ele NÃO tem padrão.
//
// Uma base vazia produziria `@get(”)`, que o navegador resolve para a página
// ATUAL: o filtro pareceria funcionar (a página recarrega) e não filtraria nada.
// Um pânico no render é barulhento e acontece na primeira vez que alguém monta a
// cena sem dizer de onde ela fala.
func (v BestiaryView) BestiaryBase() string {
	if v.Base == "" {
		panic("bestiarioView sem Base: a cena não sabe para que rota falar")
	}
	return v.Base
}

// ToggleType liga ou desliga UM crachá de tipo no conjunto.
//
// Extraída porque tem dois chamadores desde que o painel da Mesa nasceu — a cena
// do mestre e ele —, e "alternar" é regra pequena o bastante para alguém
// reescrever sem notar que já existia, e grande o bastante para as duas cópias
// discordarem sobre o que fazer com um tipo repetido.
//
// Tipo que o catálogo não conhece é RECUSADO e não descartado: a URL é editável
// à mão, e um tipo inventado no conjunto filtraria tudo fora — a tela leria
// "Nenhuma criatura casa com os filtros" sem explicar por quê. É diferente do
// `knownTypes`, que descarta de propósito porque lá o conjunto inteiro vem
// da URL e uma vírgula sobrando não deve esvaziar a tela.
func ToggleType(tipos []string, tipo string) ([]string, error) {
	if !slices.Contains(book.CreatureTypes, tipo) {
		return nil, fmt.Errorf("tipo de criatura desconhecido: %s", tipo)
	}
	if i := slices.Index(tipos, tipo); i >= 0 {
		return slices.Delete(slices.Clone(tipos), i, i+1), nil
	}
	return append(slices.Clone(tipos), tipo), nil
}

// openTheEntry marca o pedido que deve ABRIR a ficha ao terminar.
//
// A marca vai na URL e não num sinal porque a MESMA rota serve a busca e os
// filtros de tipo, e os dois mandam os sinais todos — inclusive o `criatura`.
// Um sinal não distinguiria "escolhi esta criatura" de "digitei uma letra na
// busca com uma criatura já escolhida", e a busca passaria a abrir a ficha
// sozinha a cada tecla.
func openTheEntry(base string) string {
	if strings.Contains(base, "?") {
		return base + "&abrir=1"
	}
	return base + "?abrir=1"
}

// ── campos alcançáveis pela SETA (ALE-264) ───────────────────────────────────

// navigableField são os atributos que põem um `<input>` na navegação por setas.
//
// O driver NÃO considera campo um item: o seletor dele é `a[href], button,
// [tabindex], [data-nav-item]` — `input` está fora de propósito, porque com o
// foco dentro de um campo a seta EDITA (move o cursor, muda o número) e quem
// entra não sai. `data-nav-item` é o ponto de extensão declarado para dizer
// "este é um item mesmo assim".
//
// O que torna isso seguro é a SAÍDA. O driver se recolhe em alvo de digitação
// (`isTypingTarget`) e — a parte que importa — se recolhe SEM consumir a tecla:
// não há `stop(e)`, então o evento continua até o elemento. Um `keydown` no
// próprio campo alcança o Esc, e é assim que a porta de saída existe sem tocar
// no driver, que é compartilhado com a SPA.
//
// Esc sobe para o TRILHO e não apenas tira o foco, porque é o que a gramática da
// casa faz em toda parte: o `handleBack` do driver leva para
// `[data-nav-region="rail"]` antes de sair da cena. Duas saídas diferentes para
// a mesma tecla seria a pessoa aprendendo duas regras.
//
// É `keydown` por cena? Não: é UM helper, usado por todo campo que entra na
// navegação. A filosofia proíbe "hand-roll per-scene keydown handlers", e o que
// ela protege é justamente isto — a regra num lugar só.
func navigableField() templ.Attributes {
	return templ.Attributes{
		"data-nav-item": "",
		"data-on:keydown": "evt.key === 'Escape' && " +
			"(evt.preventDefault(), document.querySelector('[data-nav-region=\"rail\"] a')?.focus())",
	}
}
