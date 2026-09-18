package master

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"t20engine/domain/book"
	"t20engine/serve/web/bookui"

	"github.com/a-h/templ"
)

// O BESTIÁRIO como dado.
//
// O catálogo fica em memória no servidor por `go:embed`, e o que atravessa a
// rede é a lista JÁ FILTRADA.
//
// Sem LISTA VIRTUALIZADA, de propósito: virtualização é maquinaria para lista
// que não cabe, e o bestiário tem 80 criaturas — a 80 ela cobra o custo sem
// cobrir problema nenhum, inclusive o de precisar de e2e porque jsdom mede
// zero.

// BestiaryView é o que a cena precisa para se desenhar inteira, numa resposta.
//
// `Total` viaja junto do resultado filtrado porque a linha "12 de 80" precisa
// dos dois números: sem ele a tela não sabe se o filtro apertou muito ou se o
// bestiário é pequeno.
type BestiaryView struct {
	// Base é o prefixo das rotas que ESTA cena chama, e existe porque o mesmo
	// desenho serve dois lugares: a cena do mestre em `/mestre/bestiario` e o
	// painel da sessão em `/campanhas/{c}/sessoes/{s}/bestiario`. O que muda é o
	// ENDEREÇO, não a lista nem o bloco.
	//
	// Sem valor não há rota: o `BestiaryBase` recusa a string vazia em vez de
	// deixar o botão apontar para a página atual, que é o defeito silencioso
	// desta forma — o clique "funciona" e recarrega a cena.
	Base string
	// Book é o endereço do PDF do livro, e o ZERO VALOR é o caso normal: sem
	// `LIVRO_PDF` configurado não há livro para abrir e o bloco não desenha o
	// botão. Vem pedido no construtor, ao lado da Base, pela mesma razão que
	// ela — são os dois endereços de que a cena depende, e um deles esquecido
	// some em silêncio.
	Book    bookui.BookAddress
	Entries []book.Entry
	Total   int
	Chosen  *book.Entry
	Term    string
	Types   []string
	CRMin   float64
	CRMax   float64
	// Open diz que ESTE pedido veio de um clique numa linha, e por isso a ficha
	// tem de nascer aberta. Vem da URL e não de um sinal — a razão está no
	// `openTheEntry`.
	Open bool
}

// chosenOrFirst: a cena SEMPRE mostra um bloco quando há lista.
//
// Quando o filtro muda e a criatura escolhida sai da lista, cai na primeira em
// vez de esvaziar — painel vazio ao lado de uma lista cheia parece defeito.
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

// LoadBestiaryFrom monta a cena a partir do que veio na URL ou nos sinais.
//
// A BASE é o primeiro parâmetro e é obrigatória: construtor que consegue
// produzir valor inválido é o próprio defeito — pedir aqui torna o
// esquecimento impossível em vez de detectável.
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
	// `sheet_open` sai DAQUI e não de um evento de sinal separado: este
	// `data-signals` mora no `#bestiary`, que É o elemento remendado, então ele
	// REDECLARA os sinais a cada remendo. Um evento de sinal mandado depois do
	// conteúdo é desfeito por esta linha — o fio leva `{"sheet_open":true}` e o
	// diálogo continua `display:none`.
	//
	// A saída não é mover os sinais para um elemento que ninguém remenda: é o
	// servidor redeclarar com o valor CERTO, e aí o conteúdo e o estado de
	// aberto chegam no MESMO remendo — atômicos, sem janela em que um esteja
	// aplicado e o outro não.
	return fmt.Sprintf(`{search: %s, ndMin: %s, ndMax: %s, tipos: %s, creature: %s, sheet_open: %t}`,
		busca, crInBox(v.CRMin), crInBox(v.CRMax), tipos, criatura, v.Open)
}

// BestiaryBase é o prefixo de rota da cena, e ele NÃO tem padrão.
//
// Uma base vazia produziria `@get(”)`, que o navegador resolve para a página
// ATUAL: o filtro pareceria funcionar (a página recarrega) e não filtraria nada.
// O pânico é barulhento e acontece na primeira vez que alguém monta a cena sem
// dizer de onde ela fala.
func (v BestiaryView) BestiaryBase() string {
	if v.Base == "" {
		panic("bestiarioView sem Base: a cena não sabe para que rota falar")
	}
	return v.Base
}

// ToggleType liga ou desliga UM crachá de tipo no conjunto.
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

// ── campos alcançáveis pela SETA ─────────────────────────────────────────────

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
// no driver.
//
// Esc sobe para o TRILHO e não apenas tira o foco, porque é o que a gramática da
// casa faz em toda parte: o `handleBack` do driver leva para
// `[data-nav-region="rail"]` antes de sair da cena. Duas saídas diferentes para
// a mesma tecla seria a pessoa aprendendo duas regras.
//
// É um helper e não um `keydown` escrito por cena, para a regra ficar num lugar
// só.
func navigableField() templ.Attributes {
	return templ.Attributes{
		"data-nav-item": "",
		"data-on:keydown": "evt.key === 'Escape' && " +
			"(evt.preventDefault(), document.querySelector('[data-nav-region=\"rail\"] a')?.focus())",
	}
}
