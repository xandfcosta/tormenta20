package api

import (
	"net/http"
	"strings"
	"testing"
)

// Os guardas do CONSTRUTOR DE ENCONTROS (ALE-259).
//
// A conta em si é do `engine` e tem os testes dela lá, contra o livro. O que se
// prende aqui é a ÁLGEBRA DO RASCUNHO e a tradução do gesto — que é onde um
// erro apaga o encontro do mestre sem avisar.

func encontroDe(t *testing.T, linhas []linhaDoEncontro) encontrosView {
	t.Helper()
	return carregaEncontros(nivelPadrao, grupoPadrao, linhas, "")
}

// TestAMesmaCriaturaSOBEAContagem, e não vira uma segunda linha.
//
// Duas linhas do mesmo verbete calculariam cada uma o próprio ND de grupo, e a
// regra da dobra (p282) só significa alguma coisa sobre UM grupo: dois grupos
// de dois ogros valeriam MENOS que um grupo de quatro, que é o oposto da regra.
// É por isso que este guarda mede o ND e não só a contagem de linhas.
func TestAMesmaCriaturaSobeAContagem(t *testing.T) {
	linhas := []linhaDoEncontro{}
	for i := 0; i < 4; i++ {
		linhas = acrescenta(linhas, "ogro")
	}
	if len(linhas) != 1 {
		t.Fatalf("%d linhas, quero 1 com quantidade 4", len(linhas))
	}
	if linhas[0].Qtd != 4 {
		t.Fatalf("quantidade %d, quero 4", linhas[0].Qtd)
	}

	umGrupoDeQuatro := encontroDe(t, linhas).ND()
	doisGruposDeDois := encontroDe(t, []linhaDoEncontro{
		{ID: "ogro", Qtd: 2}, {ID: "ogro", Qtd: 2},
	}).ND()
	if umGrupoDeQuatro >= doisGruposDeDois {
		t.Errorf("um grupo de quatro deu ND %v e dois de dois deram %v — a regra da dobra "+
			"exige que juntar seja MAIS perigoso", umGrupoDeQuatro, doisGruposDeDois)
	}
}

// TestOUltimoTiradoLEVAALinha: um grupo de zero criaturas não é um grupo, e
// deixar a linha com 0 mostraria "ND 0" numa linha que ainda parece parte do
// encontro.
func TestOUltimoTiradoLevaALinha(t *testing.T) {
	linhas := []linhaDoEncontro{{ID: "ogro", Qtd: 2}}
	linhas = diminui(linhas, "ogro")
	if len(linhas) != 1 || linhas[0].Qtd != 1 {
		t.Fatalf("depois de um passo: %+v", linhas)
	}
	linhas = diminui(linhas, "ogro")
	if len(linhas) != 0 {
		t.Errorf("a linha sobreviveu ao último passo: %+v", linhas)
	}
}

// TestOVerbeteQueSumiuNaoVIRALinhaVazia. Um id velho colado numa URL
// renderizaria uma linha sem nome com quantidade viva.
func TestOVerbeteQueSumiuNaoViraLinhaVazia(t *testing.T) {
	v := encontroDe(t, []linhaDoEncontro{
		{ID: "ogro", Qtd: 1},
		{ID: "dragao-de-papel-machê", Qtd: 3},
	})
	if len(v.Linhas) != 1 {
		t.Fatalf("%d linhas, quero só a que existe", len(v.Linhas))
	}
	if v.Linhas[0].Verbete.Name == "" {
		t.Error("linha sem nome sobreviveu")
	}
}

// TestOLinkCopiadoREABREOEncontro é o ciclo inteiro: montar, copiar, colar.
//
// Ele existe porque o formato do link é escrito num lugar e lido em outro, e é
// exatamente aí que um `:` vira `-` e o encontro chega vazio do outro lado sem
// nenhum erro.
func TestOLinkCopiadoReabreOEncontro(t *testing.T) {
	original := carregaEncontros(5, 3, []linhaDoEncontro{
		{ID: "ogro", Qtd: 2}, {ID: "goblin-salteador", Qtd: 4},
	}, "")

	link := enderecoDoEncontro(original)
	_, query, achou := strings.Cut(link, "?")
	if !achou {
		t.Fatalf("o link não tem query: %q", link)
	}

	// A volta: o que o `?c=` carrega tem de reconstruir a mesma composição.
	var c string
	for _, par := range strings.Split(query, "&") {
		if chave, valor, _ := strings.Cut(par, "="); chave == "c" {
			c = strings.ReplaceAll(valor, "%3A", ":")
			c = strings.ReplaceAll(c, "%2C", ",")
		}
	}
	devolta := carregaEncontros(5, 3, linhasDaURL(c), "")

	if len(devolta.Linhas) != len(original.Linhas) {
		t.Fatalf("voltaram %d linhas de %d", len(devolta.Linhas), len(original.Linhas))
	}
	if devolta.ND() != original.ND() {
		t.Errorf("o ND mudou na volta: %v virou %v", original.ND(), devolta.ND())
	}
	for i := range original.Linhas {
		if devolta.Linhas[i].Verbete.ID != original.Linhas[i].Verbete.ID ||
			devolta.Linhas[i].Qtd != original.Linhas[i].Qtd {
			t.Errorf("linha %d voltou diferente: %+v", i, devolta.Linhas[i])
		}
	}
}

// TestOLinkTortoNaoCustaOEncontroInteiro: ele chega por chat, e um caractere a
// mais não pode zerar o que veio junto.
func TestOLinkTortoNaoCustaOEncontroInteiro(t *testing.T) {
	linhas := linhasDaURL("ogro:2,lixo,goblin-salteador:x,,cascavel:3")
	if len(linhas) != 2 {
		t.Fatalf("%d linhas de duas boas: %+v", len(linhas), linhas)
	}
	if linhas[0].ID != "ogro" || linhas[1].ID != "cascavel" {
		t.Errorf("as linhas boas não sobreviveram: %+v", linhas)
	}
}

// TestONivelEOTamanhoAbsurdosCaemNoPadrao. Os dois vêm da URL, que qualquer um
// edita à mão, e um nível 999 mudaria a dificuldade sem mudar o encontro.
func TestONivelEOTamanhoAbsurdosCaemNoPadrao(t *testing.T) {
	v := carregaEncontros(999, -3, nil, "")
	if v.Nivel != nivelPadrao || v.Grupo != grupoPadrao {
		t.Errorf("nível %d e grupo %d, quero os padrões %d e %d",
			v.Nivel, v.Grupo, nivelPadrao, grupoPadrao)
	}
}

// TestOPainelDeBuscaSoAparaceComTermo: mostrar as 80 criaturas abaixo da
// composição empurraria o VEREDITO para fora da tela, e o veredito é o assunto
// da ferramenta (ALE-170).
func TestOPainelDeBuscaSoApareceComTermo(t *testing.T) {
	if got := carregaEncontros(1, 4, nil, "").Achados; len(got) != 0 {
		t.Errorf("sem termo vieram %d criaturas", len(got))
	}
	if got := carregaEncontros(1, 4, nil, "ogro").Achados; len(got) == 0 {
		t.Error("com termo não veio criatura nenhuma")
	}
}

// ── pelo fio ─────────────────────────────────────────────────────────────────

// TestOGestoDeAcrescentarDevolveRemendo, e não uma página: recarregar no meio
// da montagem perderia o rascunho, que só vive nos sinais.
func TestOGestoDeAcrescentarDevolveRemendo(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/piloto/mestre/encontros/adicionar/ogro", `{"encontro":[]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type %q — o gesto navegou em vez de remendar", ct)
	}
	if !strings.Contains(rec.Body.String(), "Ogro") {
		t.Error("o remendo não trouxe a criatura acrescentada")
	}
}

// TestOEncontroDaURLValeNaCargaFria — é o link colado no chat abrindo montado.
func TestOEncontroDaURLValeNaCargaFria(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET",
		"/piloto/mestre/encontros?nivel=3&grupo=4&c=ogro:2", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	corpo := rec.Body.String()
	if !strings.Contains(corpo, "Ogro") {
		t.Error("o encontro do link não foi desenhado")
	}
	esperado := carregaEncontros(3, 4, []linhaDoEncontro{{ID: "ogro", Qtd: 2}}, "")
	if !strings.Contains(corpo, esperado.Dificuldade().Rotulo) {
		t.Errorf("a dificuldade %q não está na página", esperado.Dificuldade().Rotulo)
	}
}
