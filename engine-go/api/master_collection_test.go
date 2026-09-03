package api

import (
	"fmt"
	"net/http"
	"strings"
	"t20engine/book"
	"testing"
)

// Os guardas dos CATÁLOGOS (ALE-258).

// TestACatalogSceneDrawsTheWholeCatalog.
//
// O endereço mudou na ALE-264: cada catálogo virou uma parada do trilho e ganhou
// cena própria (`/mestre/condicoes`). Este guarda passou a pedir a cena
// direto — quem cobra o endereço VELHO é o `TestTheOldCollectionAddressRedirects`.
func TestACatalogSceneDrawsTheWholeCatalog(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	corpo := rec.Body.String()
	a := book.Catalogs()
	if !strings.Contains(corpo, fmt.Sprintf("%d entradas", len(a.Condicoes))) {
		t.Errorf("a contagem não é a das %d condições", len(a.Condicoes))
	}
	// Uma condição de verdade, e o texto dela: sem isso o teste passaria com a
	// cena desenhando só o cabeçalho.
	if !strings.Contains(corpo, "Abalado") {
		t.Error("a condição Abalado não está na página")
	}
	if strings.Contains(corpo, "Bola de Fogo") {
		t.Error("a aba Condições desenhou magia — a aba não é um catálogo só")
	}
}

// TestTheWholeCollectionComesOutInThePowersTab: a decisão do dono foi mandar TUDO, sem
// teto nem paginação. Se alguém puser um `[:60]` aqui um dia, este guarda cai.
func TestTheWholeCollectionComesOutInThePowersTab(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/mestre/poderes", "")
	poderes := book.Catalogs().Poderes
	if !strings.Contains(rec.Body.String(), fmt.Sprintf("%d entradas", len(poderes))) {
		t.Fatalf("a contagem não é a dos %d poderes", len(poderes))
	}
	// O ÚLTIMO da lista, e não o primeiro: um teto cortaria pelo fim.
	ultimo := poderes[len(poderes)-1]
	if !strings.Contains(rec.Body.String(), ultimo.Name) {
		t.Errorf("o último poder (%q) não saiu — a lista foi cortada", ultimo.Name)
	}
}

// TestSearchingSweepsTheEightCatalogsFromAnyScene.
//
// Este guarda cobrava a FILEIRA DE ABAS, que sumiu na ALE-264 — cada catálogo
// virou uma parada do trilho, e ter as duas coisas seria o mesmo estado
// desenhado em dois lugares. O que ele protegia CONTINUA valendo e é o que ele
// cobra agora: com termo digitado a busca varre os OITO catálogos, não só o da
// cena. É a ALE-22 — "bola de fogo" digitado em Condições dizia "nada
// encontrado" com a magia existindo.
func TestSearchingSweepsTheEightCatalogsFromAnyScene(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	// Da cena das CONDIÇÕES, buscando uma MAGIA.
	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes?busca=bola+de+fogo", "").Body.String()
	if !strings.Contains(corpo, "Bola de Fogo") {
		t.Error("a busca da cena de condições não achou a magia — voltou a filtrar só a aba")
	}
	// O CONTROLE: sem termo, a cena mostra só o catálogo dela.
	so := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes", "").Body.String()
	if strings.Contains(so, "Bola de Fogo") {
		t.Error("sem busca a cena das condições trouxe magia")
	}
}

// TestTheOldCollectionAddressRedirects: `?aba=` foi o único endereço por duas
// fatias desta issue, e pode estar colado no chat de alguma mesa.
func TestTheOldCollectionAddressRedirects(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/mestre/catalogos?aba=magias&busca=fogo", "")
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("o endereço velho respondeu %d", rec.Code)
	}
	destino := rec.Header().Get("Location")
	if !strings.HasPrefix(destino, "/mestre/magias") {
		t.Errorf("levou para %q", destino)
	}
	// A CONSULTA sobrevive: um redirecionamento que perde a busca devolve a
	// pessoa a uma tela que não é a que ela pediu.
	if !strings.Contains(destino, "busca=fogo") {
		t.Errorf("o redirecionamento perdeu a busca: %q", destino)
	}
}
