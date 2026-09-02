package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"t20engine/search"
	"testing"
)

// O guarda do BUSCADOR DO LIVRO (ALE-264).
//
// O que ele protege não é "achou alguma coisa" — é a ORDEM e o CORTE. A caixa
// mostra seis por grupo de 1.072 entradas, então um ranqueamento ruim não
// aparece como lista errada: aparece como o verbete certo faltando, que é
// indistinguível de "não existe no livro".
//
// Três dos casos abaixo nasceram VERMELHOS contra o código escrito, e estão
// anotados um a um.

// TestOVerbeteCertoVemPrimeiro: a escada de pontuação, do nome inteiro ao typo.
func TestOVerbeteCertoVemPrimeiro(t *testing.T) {
	escada := []struct {
		nome, busca string
		esperado    int
	}{
		{"Abalado", "abalado", 100},
		{"Abalado", "abal", 80},
		{"Bola de Fogo", "fogo", 60},
		{"Bola de Fogo", "ogo", 40},
		{"Necromante", "ncromante", 20},
		{"Naja", "abal", 0},
	}
	for _, c := range escada {
		if ponto := search.Score(c.nome, c.busca); ponto != c.esperado {
			t.Errorf("pontuaBusca(%q, %q) = %d, esperado %d", c.nome, c.busca, ponto, c.esperado)
		}
	}
}

// TestDoisTermosAchamONomeInteiro.
//
// PROVADO VERMELHO com a versão de um termo só: "bola fogo" devolvia ZERO —
// o nome não começa com a frase, não a contém, e pular o "de " estoura a folga
// do quase-igual. Digitar duas palavras do que se lembra é o gesto normal.
func TestDoisTermosAchamONomeInteiro(t *testing.T) {
	v := buscaNoLivro("bola fogo")
	if !temOAchado(v, "Bola de Fogo") {
		t.Errorf("“bola fogo” não achou “Bola de Fogo” — %d achados", v.Achados)
	}
	if v.PeloTexto {
		t.Error("o acerto foi anunciado como menção no texto, e é casamento de NOME")
	}
}

// TestOCorpoDaRegraSoEntraQuandoNomeNenhumCasa.
//
// PROVADO VERMELHO com o corpo valendo sempre: "abal" devolvia 282 entradas —
// 139 poderes cujo texto diz "Abalado" — e a condição "Abalado", que era o que
// se procurava, saía num grupo de seis ao lado de "Naja" e "Jiboia".
//
// O controle é o segundo caso: quem NÃO sabe o nome ("chance de falha") continua
// achando, e a tela diz que aquilo é menção e não nome.
func TestOCorpoDaRegraSoEntraQuandoNomeNenhumCasa(t *testing.T) {
	porNome := buscaNoLivro("abal")
	if porNome.PeloTexto {
		t.Fatal("houve casamento de nome e a busca caiu na segunda passada mesmo assim")
	}
	if porNome.Grupos[0].Achados[0].Nome != "Abalado" {
		t.Errorf("o primeiro achado de “abal” é %q", porNome.Grupos[0].Achados[0].Nome)
	}
	if porNome.Achados > 20 {
		t.Errorf("“abal” trouxe %d achados: o corpo das regras vazou para a primeira passada", porNome.Achados)
	}

	porTexto := buscaNoLivro("chance de falha")
	if porTexto.Achados == 0 {
		t.Fatal("quem não sabe o nome ficou sem nada — a segunda passada não roda")
	}
	if !porTexto.PeloTexto {
		t.Error("a tela não vai avisar que estes achados são menção, e a lista vai parecer errada")
	}
}

// TestOCorteDizQuantoSobrouETemSaida: corte silencioso ensina que não existe.
func TestOCorteDizQuantoSobrouETemSaida(t *testing.T) {
	v := buscaNoLivro("arma")
	poderes := grupoChamado(v, "Poderes")
	if poderes == nil {
		t.Fatal("“arma” não achou poder nenhum — o guarda mediria outra coisa")
	}
	if len(poderes.Achados) != achadosPorGrupo {
		t.Errorf("o grupo veio com %d linhas, e o corte é %d", len(poderes.Achados), achadosPorGrupo)
	}
	if poderes.Cortados() <= 0 {
		t.Fatalf("“arma” achou %d poderes: escolha outro termo para medir o corte", poderes.Total)
	}
	if !strings.Contains(poderes.Mais, url.QueryEscape("arma")) {
		t.Errorf("o “+%d” leva para %q, que não é a cena com a mesma busca", poderes.Cortados(), poderes.Mais)
	}
}

// TestOAchadoSabeParaOndeLevar: cada linha é um endereço, e eles diferem por
// ferramenta — criatura vai ao bestiário, o resto ao acervo.
func TestOAchadoSabeParaOndeLevar(t *testing.T) {
	criatura := primeiroDoGrupo(t, buscaNoLivro("lobo"), "Criaturas")
	if !strings.HasPrefix(criatura.Destino, rotaDoBestiarioDoMestre+"?criatura=") {
		t.Errorf("a criatura leva para %q", criatura.Destino)
	}
	if criatura.Pagina == 0 {
		t.Error("a criatura veio sem página do livro, e o bestiário é o único catálogo que sabe a dele")
	}
	condicao := primeiroDoGrupo(t, buscaNoLivro("abalado"), "Condições")
	if !strings.Contains(condicao.Destino, "/mestre/condicoes") {
		t.Errorf("a condição leva para %q", condicao.Destino)
	}
}

// TestARotaDoBuscadorLeOSinal: o caminho que o navegador usa de verdade.
//
// Pelo SINAL e não por `?busca=`: é assim que o `@get` do Datastar manda o que
// foi digitado, e a URL é só o caminho de quem abre o endereço à mão. Um guarda
// que só medisse a URL passaria verde com o sinal quebrado.
func TestARotaDoBuscadorLeOSinal(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeAoBuscador(t, s, eu, `{"buscador":"abalado"}`)
	if !strings.Contains(corpo, "datastar-patch-elements") {
		t.Fatal("a rota não devolveu remendo nenhum — o resto do guarda mediria a resposta errada")
	}
	if !strings.Contains(corpo, "Abalado") {
		t.Error("o remendo não traz a condição buscada")
	}
	if !strings.Contains(corpo, `id="buscador-achados"`) {
		t.Error("o remendo não traz o id que ele substitui — o Datastar não teria onde aplicá-lo")
	}
}

// TestAPortaNaoDesenhaOBuscador.
//
// A caixa liga um sinal, e sinal é estado de cliente que viaja em TODA
// requisição seguinte — na porta, junto com a senha. O
// `TestPortaNaoPoeNadaEmSinalDoDatastar` cobra a regra geral; este cobra que
// esta caixa em particular ficou de fora.
//
// O controle é o segundo caso: a MESMA casca, numa tela com sessão, desenha.
func TestAPortaNaoDesenhaOBuscador(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	porta := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(porta, httptest.NewRequest(http.MethodGet, "/entrar", nil))
	if strings.Contains(porta.Body.String(), `id="buscador"`) {
		t.Error("a porta desenhou a caixa do buscador, e com ela um sinal que viaja com a senha")
	}

	dentro := pedeNoMestre(t, s, eu, "GET", "/mestre/bestiario", "")
	if !strings.Contains(dentro.Body.String(), `id="buscador"`) {
		t.Error("a caixa sumiu da cena com sessão — o guarda acima passaria por ausência de tudo")
	}
}

func pedeAoBuscador(t *testing.T, s *Server, userID int64, sinais string) string {
	t.Helper()
	u, err := s.queries.GetUserByID(t.Context(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.signToken(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, rotaDoBuscador+"?datastar="+url.QueryEscape(sinais), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("datastar-request", "true")
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("o buscador respondeu %d", rec.Code)
	}
	return rec.Body.String()
}

func grupoChamado(v buscadorView, rotulo string) *grupoDoBuscador {
	for i := range v.Grupos {
		if v.Grupos[i].Rotulo == rotulo {
			return &v.Grupos[i]
		}
	}
	return nil
}

func primeiroDoGrupo(t *testing.T, v buscadorView, rotulo string) achadoDoBuscador {
	t.Helper()
	g := grupoChamado(v, rotulo)
	if g == nil || len(g.Achados) == 0 {
		t.Fatalf("nenhum achado em %q para %q", rotulo, v.Busca)
	}
	return g.Achados[0]
}

func temOAchado(v buscadorView, nome string) bool {
	for _, g := range v.Grupos {
		for _, a := range g.Achados {
			if a.Nome == nome {
				return true
			}
		}
	}
	return false
}

// TestOMelhorAchadoVemNoPrimeiroGrupo (ALE-264).
//
// PROVADO VERMELHO contra a ordem fixa: os grupos saíam na ordem da FILEIRA DE
// ABAS, e o dono viu o efeito — digitando "medo", o verbete "Medo" (nome
// inteiro, nota máxima) aparecia no sexto grupo, abaixo de criaturas que só têm
// a palavra no nome. A ordem da fileira é a certa para NAVEGAR e a errada para
// BUSCAR.
func TestOMelhorAchadoVemNoPrimeiroGrupo(t *testing.T) {
	casos := []struct{ termo, grupo, achado string }{
		{"medo", "Efeitos", "Medo"},
		{"abal", "Condições", "Abalado"},
		{"lobo", "Criaturas", "Lobo"},
	}
	for _, caso := range casos {
		v := buscaNoLivro(caso.termo)
		if len(v.Grupos) == 0 {
			t.Errorf("%q não achou nada", caso.termo)
			continue
		}
		primeiro := v.Grupos[0]
		if primeiro.Rotulo != caso.grupo || primeiro.Achados[0].Nome != caso.achado {
			t.Errorf("%q → %s(%s); esperado %s(%s)",
				caso.termo, primeiro.Rotulo, primeiro.Achados[0].Nome, caso.grupo, caso.achado)
		}
	}
}

// TestOEmpateMantemAOrdemDaFileira: a ordenação é ESTÁVEL.
//
// A ordem das abas tem razão registrada — condição primeiro porque é a consulta
// do combate — e ela continua valendo quando dois grupos têm achados igualmente
// bons. Sem estabilidade, a mesma busca poderia sair em ordens diferentes.
func TestOEmpateMantemAOrdemDaFileira(t *testing.T) {
	grupos := []grupoDoBuscador{
		{Rotulo: "Condições", Achados: []achadoDoBuscador{{Nome: "a", ponto: 40}}},
		{Rotulo: "Magias", Achados: []achadoDoBuscador{{Nome: "b", ponto: 40}}},
		{Rotulo: "Itens", Achados: []achadoDoBuscador{{Nome: "c", ponto: 100}}},
	}
	ordenaPorRelevancia(grupos)
	if grupos[0].Rotulo != "Itens" {
		t.Errorf("o grupo com o melhor achado não veio na frente: %q", grupos[0].Rotulo)
	}
	if grupos[1].Rotulo != "Condições" || grupos[2].Rotulo != "Magias" {
		t.Errorf("o empate não manteve a ordem da fileira: %q, %q", grupos[1].Rotulo, grupos[2].Rotulo)
	}
}
