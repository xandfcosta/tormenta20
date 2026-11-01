package api

import (
	"fmt"
	"strings"
	"testing"

	"t20engine/domain/book"
)

// Os dez catálogos vestem a moldura da Mesa do Mestre, cada um com o próprio nome
// no cabeçalho e o verbete em cartão. Os nomes estão escritos à mão de propósito:
// o cabeçalho é o que diz à pessoa onde ela está.
func TestEveryCatalogWearsTheMasterFrame(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")

	for slug, title := range map[string]string{
		"classes": "Classes", "condicoes": "Condições", "deuses": "Deuses",
		"efeitos": "Efeitos", "escolas": "Escolas", "itens": "Itens",
		"magias": "Magias", "pericias": "Perícias", "poderes": "Poderes", "racas": "Raças",
	} {
		t.Run(slug, func(t *testing.T) {
			body := pedeNoMestre(t, s, gm, "GET", "/mestre/"+slug, "").Body.String()
			if !strings.Contains(body, "scene-ds") {
				t.Fatal("o catálogo não veste o Tempestade")
			}
			if !strings.Contains(body, `<h1 class="ds-header-title">`+title+`</h1>`) {
				t.Errorf("o cabeçalho não diz %q", title)
			}
			menu := sliceBetween(t, body, `class="ds-master-menu"`, "</nav>")
			current := sliceBetween(t, menu, `href="/mestre/`+slug+`"`, ">")
			if !strings.Contains(current, `aria-current="page"`) {
				t.Errorf("o menu não marca %s como a tela atual: %s", slug, current)
			}
			if !strings.Contains(body, `class="ds-catalog-card"`) {
				t.Error("o catálogo não desenha o verbete em cartão")
			}
		})
	}
}

// Sem nenhum crachá aceso o cabeçalho diz o tamanho do catálogo, e com um aceso
// ele diz quanto do catálogo sobrou.
func TestTheCatalogHeadSaysHowMuchIsLeft(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")

	whole := pedeNoMestre(t, s, gm, "GET", "/mestre/condicoes", "").Body.String()
	if !strings.Contains(whole, "<span>Condições</span><span>35</span>") {
		t.Error("o catálogo inteiro não diz que tem 35 verbetes")
	}
	filtered := pedeNoMestre(t, s, gm, "GET", "/mestre/condicoes?efeito=medo", "").Body.String()
	if !strings.Contains(filtered, "<span>Condições</span><span>2 de 35</span>") {
		t.Error("o catálogo filtrado não diz quanto sobrou de 35")
	}
}

// A caixa do elo desenha TODO catálogo que um elo pode citar. O nome da escola e o
// da perícia são elos nos cartões de magia e de classe: sem o cartão delas a caixa
// abria vazia, e o elo parecia quebrado.
func TestTheEntryBoxDrawsEveryCatalogALinkCanCite(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")

	for _, tc := range []struct{ tab, id, name string }{
		{"escolas", "evocacao", "Evocação"},
		{"pericias", "acrobacia", "Acrobacia"},
		{"condicoes", "abalado", "Abalado"},
		{"efeitos", "medo", "Medo"},
	} {
		body := pedeNoMestre(t, s, gm, "GET", fmt.Sprintf("/verbete?aba=%s&entrada=%s", tc.tab, tc.id), "").Body.String()
		if !strings.Contains(body, tc.name) {
			t.Errorf("a caixa de %s/%s não desenhou %q", tc.tab, tc.id, tc.name)
		}
	}
}

// "31 poderes de classe" leva aos poderes daquela classe, e o número tem de ser o
// que o filtro mostra: um elo que promete 31 e entrega outra conta faz o mestre
// desconfiar dos dois.
func TestTheClassPowerCountLeadsToThatManyPowers(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")

	_, classes, _ := book.CharacterCatalogs()
	page := pedeNoMestre(t, s, gm, "GET", "/mestre/classes", "").Body.String()
	for _, c := range classes {
		address := "/mestre/poderes?fonte=" + c.Name
		if !strings.Contains(page, `href="`+address+`"`) {
			t.Errorf("o cartão de %s não leva a %s", c.Name, address)
			continue
		}
		filtered := pedeNoMestre(t, s, gm, "GET", address, "").Body.String()
		want := fmt.Sprintf("<span>%d de ", c.Powers)
		if !strings.Contains(filtered, want) {
			t.Errorf("%s promete %d poderes e o filtro não mostra essa conta", c.Name, c.Powers)
		}
	}
}

// A duração da magia é escrita como o livro escreve, e não como a chave que o motor
// lê. "definida" não sai: é a magia cuja duração o próprio texto diz.
func TestTheSpellDurationComesOutWrittenAndNotAsAKey(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")

	body := pedeNoMestre(t, s, gm, "GET", "/mestre/magias", "").Body.String()
	for _, written := range []string{"Duração: instantânea", "Duração: 1 dia", "Duração: cena"} {
		if !strings.Contains(body, written) {
			t.Errorf("nenhuma magia diz %q", written)
		}
	}
	for _, key := range []string{"Duração: instantanea", "Duração: dia", "Duração: definida"} {
		if strings.Contains(body, key) {
			t.Errorf("a tela escreveu a chave: %q", key)
		}
	}
}

// O título da caixa diz o que ela mostra: o verbete pelo nome, e os aprimoramentos
// como aprimoramentos. O título vem no mesmo remendo do miolo.
func TestTheEntryBoxTitleSaysWhatItShows(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")

	for address, want := range map[string]string{
		"/verbete?aba=condicoes&entrada=abalado":                        ">Abalado</h2>",
		"/verbete?aba=escolas&entrada=evocacao":                         ">Evocação</h2>",
		"/verbete?aba=magias&parte=aprimoramentos&entrada=bola-de-fogo": ">Aprimoramentos de Bola de Fogo</h2>",
		"/verbete?aba=efeitos&entrada=nao-existe":                       ">Verbete</h2>",
	} {
		body := pedeNoMestre(t, s, gm, "GET", address, "").Body.String()
		if !strings.Contains(body, `id="entry-in-dialog-title"`) {
			t.Errorf("%s: o remendo não traz o título da caixa", address)
		}
		if !strings.Contains(body, want) {
			t.Errorf("%s: o título não diz %q", address, want)
		}
	}
}
