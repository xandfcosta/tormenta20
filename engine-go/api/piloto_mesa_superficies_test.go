package api

import (
	"net/http"
	"strings"
	"testing"
)

// Os guardas da FORMA DO JOGADOR (ALE-129, portada na ALE-269).
//
// A cena do jogador deixou de ser uma coluna com tudo empilhado e virou DUAS
// superfícies que ocupam a tela, com um seletor ancorado no topo. O que se prende
// aqui é o que a mudança de forma pode quebrar em silêncio.

// TestOJogadorTemTODASasRegioesUmaVezSo — o espelho do guarda do palco do mestre,
// e ele importa mais depois desta fatia.
//
// As superfícies são `data-show`, e `data-show` é `display:none` e NÃO remoção:
// as regiões continuam todas no documento, escondidas ou não. É isso que faz o
// stream continuar remendando a fila enquanto o jogador olha o mapa — e é
// exatamente o que se perderia ao trocar o `data-show` por um `if` do servidor.
//
// "Uma vez só" é a metade que pega o erro provável: mover uma região para uma
// superfície e esquecer de tirá-la da outra deixa DUAS raízes com o mesmo id, o
// remendo acerta a primeira e a segunda envelhece na tela.
func TestOJogadorTemTodasAsRegioesUmaVezSo(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)

	html := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	for _, id := range asRegioesDaMesa {
		marca := `id="` + id + `"`
		if n := strings.Count(html, marca); n != 1 {
			t.Errorf("a região %q aparece %d vezes na cena do jogador, e o remendo precisa de exatamente 1", id, n)
		}
	}
}

// TestOSeletorTemAsTresSuperficies.
//
// Por duas fatias foram DUAS, e a asserção aqui era a negativa: a ficha era a
// última tela da migração e a aba dela nasceria junto com ela (decisão do dono).
// A ficha nasceu na fatia 8 e ganhou link na 10a; a aba entra na 10b, antes de a
// SPA ser apagada, para a migração não tirar da mesa o que ela tinha.
//
// A asserção é sobre o RÓTULO que o usuário leria, e não sobre um id interno: é
// o rótulo que promete.
func TestOSeletorTemAsTresSuperficies(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)

	html := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(html, "O que ver na sessão") {
		t.Fatal("o jogador não recebeu o seletor de superfícies")
	}
	for _, rotulo := range []string{"Ficha", "Mesa", "Tabuleiro"} {
		if !strings.Contains(html, ">"+rotulo+"<") {
			t.Errorf("o seletor não oferece %q", rotulo)
		}
	}
	// E a ficha chega DESENHADA, não prometida: a aba sem conteúdo atrás é
	// exatamente o que a decisão do dono evitava enquanto ela não existia.
	if !strings.Contains(html, `id="cena-ficha"`) {
		t.Error("a aba Ficha está na tela e a ficha não veio junto")
	}
}

// TestAFichaNaSessaoNaoNavegaParaForaDela.
//
// Dentro da sessão as abas da ficha são COMANDO e não link. Um `<a href>` ali
// tiraria o jogador da mesa no meio do combate — e o modo de errar é silencioso,
// porque o link funciona: ele leva para uma tela legítima, só que a errada.
func TestAFichaNaSessaoNaoNavegaParaForaDela(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)

	html := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	dentroDaFicha := html[strings.Index(html, `id="cena-ficha"`):]
	if i := strings.Index(dentroDaFicha, "Seções da ficha"); i >= 0 {
		nav := dentroDaFicha[i : i+3000]
		if strings.Contains(nav, `href="/piloto/personagens/`) {
			t.Error("a fileira de abas da ficha embutida ainda tem link para fora da sessão")
		}
		if !strings.Contains(nav, "embutida=1") {
			t.Error("a aba embutida não carrega a marca que a mantém dentro da sessão")
		}
	} else {
		t.Fatal("não achei a fileira de abas dentro da ficha embutida")
	}
}

// TestOMestreNaoRecebeOSeletor.
//
// Ele tem o PALCO — faixa, trilhos e tabuleiro ao mesmo tempo —, e é essa a
// diferença entre as duas formas. Um seletor na tela dele esconderia atrás de uma
// aba o que a forma do mestre existe para mostrar junto.
func TestOMestreNaoRecebeOSeletor(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)

	html := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	// O CONTROLE: a cena do mestre chegou inteira. Sem ele, "não achei o seletor"
	// seria verdade também num 403 ou numa página vazia.
	if !strings.Contains(html, `id="mesa-trilho-fila"`) {
		t.Fatal("o mestre não recebeu o palco — a página não é o que este teste pensa que é")
	}
	if strings.Contains(html, "O que ver na sessão") {
		t.Error("o mestre recebeu o seletor de superfícies do jogador")
	}
}

// TestASuperficieQueAbreEDERIVADAenaoDIGITADA.
//
// O padrão é a MESA (decisão do dono), e ele é escrito num lugar só: a página
// semeia o sinal a partir da mesma constante que a lista de superfícies usa.
// Digitar 'mesa' no `data-signals` seria a segunda cópia, e a que fica para trás
// no dia em que o padrão mudar — a cena nasceria com um sinal e o botão marcando
// outro.
func TestASuperficieQueAbreEDerivadaENaoDigitada(t *testing.T) {
	f := novoPiloto(t)
	html := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(html, `superficie: &#39;`+superficieQueAbrePadrao+`&#39;`) {
		t.Errorf("a página não semeia a superfície padrão (%q)", superficieQueAbrePadrao)
	}
	// E a superfície padrão precisa EXISTIR na lista: um padrão que não é
	// oferecido abriria a sessão com as duas abas apagadas e a tela vazia.
	oferecida := false
	for _, s := range asSuperficiesDoJogador {
		oferecida = oferecida || s.ID == superficieQueAbrePadrao
	}
	if !oferecida {
		t.Errorf("a superfície padrão %q não está entre as oferecidas", superficieQueAbrePadrao)
	}
}
