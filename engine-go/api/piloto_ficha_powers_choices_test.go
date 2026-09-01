package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// Os guardas do DIÁLOGO DE ESCOLHER PODERES (ALE-272, fatia 8).
//
// A parte que interessa é a FRONTEIRA: quantos poderes cabem no nível, quantos
// benefícios a origem dá e quais caminhos a classe aceita eram regra só da tela
// — o `handleUpdateAbilities` gravava os cinco blobs sem conferir nada.

func oEscolhido(t *testing.T, f pilotoFixture, id int64) string {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	return row.Classpowers
}

// AS VAGAS DE PODER são uma por nível a partir do 2º (p33).
func TestAsVagasDePoderSaoUmaPorNivelAPartirDoSegundo(t *testing.T) {
	f, id := oBarbaro(t, 3)

	// Duas vagas no 3º nível: a do 2º e a do 3º.
	if recusa := oComandoDoPoder(t, f, id, "escolhe/class.barbaro.golpe-poderoso", ""); recusa != "" {
		t.Fatalf("a primeira escolha foi recusada: %q", recusa)
	}
	if recusa := oComandoDoPoder(t, f, id, "escolhe/class.barbaro.brado-assustador", ""); recusa != "" {
		t.Fatalf("a segunda escolha foi recusada: %q", recusa)
	}
	// A TERCEIRA não cabe.
	recusa := oComandoDoPoder(t, f, id, "escolhe/class.barbaro.frenesi", "")
	if !strings.Contains(recusa, "2 vagas") {
		t.Errorf("a recusa não diz quantas vagas existem: %q", recusa)
	}
	if guardados := oEscolhido(t, f, id); strings.Contains(guardados, "frenesi") {
		t.Errorf("a recusa gravou assim mesmo: %s", guardados)
	}
}

// O NÍVEL 1 NÃO TEM VAGA: a primeira abre no 2º.
func TestNoPrimeiroNivelNaoHaVagaDePoder(t *testing.T) {
	f, id := oBarbaro(t, 1)

	recusa := oComandoDoPoder(t, f, id, "escolhe/class.barbaro.golpe-poderoso", "")
	if !strings.Contains(recusa, "0 vagas") {
		t.Errorf("o nível 1 aceitou um poder: %q", recusa)
	}
}

// UM PODER DE OUTRA CLASSE não entra.
func TestUmPoderDeOutraClasseNaoEntra(t *testing.T) {
	f, id := oBarbaro(t, 5)

	recusa := oComandoDoPoder(t, f, id, "escolhe/class.bardo.lendas-e-historias", "")
	if !strings.Contains(recusa, "Bardo") {
		t.Errorf("a recusa não diz de qual classe é o poder: %q", recusa)
	}
	// O PODER GERAL, ao contrário, entra em qualquer classe: "você sempre pode
	// substituir um poder de classe por um poder geral" (p33).
	if recusa := oComandoDoPoder(t, f, id, "escolhe/ataque-poderoso", ""); recusa != "" {
		t.Errorf("um poder geral foi recusado: %q", recusa)
	}
}

// UM PODER AUTOMÁTICO não ocupa vaga — nem se alguém tentar escolhê-lo.
func TestUmPoderAutomaticoNaoOcupaVaga(t *testing.T) {
	f, id := oBarbaro(t, 5)

	recusa := oComandoDoPoder(t, f, id, "escolhe/class.barbaro.furia", "")
	if !strings.Contains(recusa, "automático") {
		t.Errorf("a Fúria, que o nível concede, foi aceita como escolha: %q", recusa)
	}
}

// A ORIGEM DÁ DOIS BENEFÍCIOS, e o terceiro é recusado.
func TestAOrigemDaDoisBeneficios(t *testing.T) {
	f, id := oBarbaro(t, 3)

	for _, b := range []string{"pericia-Furtividade", "pericia-Percepção"} {
		if recusa := oComandoDoPoder(t, f, id, "origem/origin-batedor-"+b, ""); recusa != "" {
			t.Fatalf("o benefício %q foi recusado: %q", b, recusa)
		}
	}
	recusa := oComandoDoPoder(t, f, id, "origem/origin-batedor-pericia-Sobrevivência", "")
	if !strings.Contains(recusa, "2 benefícios") {
		t.Errorf("o terceiro benefício foi aceito: %q", recusa)
	}
}

// O PODER ÚNICO DA ORIGEM é escolhível — ele não está na lista de benefícios.
//
// O catálogo guarda o `poderUnico` num campo à parte, e a ficha o trata como um
// dos dois que a pessoa leva (p85). Medido na bancada: sem juntá-lo, o poder da
// origem não aparecia em lugar nenhum e o servidor recusava quem tentasse
// escolhê-lo.
func TestOPoderUnicoDaOrigemEEscolhivel(t *testing.T) {
	f, id := oBarbaro(t, 3)

	if recusa := oComandoDoPoder(t, f, id, "origem/origin-batedor-unique", ""); recusa != "" {
		// O id do poder único do Batedor sai do catálogo; se ele mudar, o teste
		// falha dizendo o que procurar.
		t.Fatalf("o poder único da origem foi recusado: %q", recusa)
	}
	if !strings.Contains(aTelaDosPoderes(t, f, id), "Estilo de Disparo") {
		t.Error("o poder único escolhido não aparece na lista")
	}
}

// UMA ORIGEM SEM BENEFÍCIOS não cobra o que não oferece.
//
// O Amnésico é a única assim: "em vez de dois benefícios, recebe uma perícia e
// um poder escolhidos pelo mestre" (p88). Cobrar dois dele daria uma pendência
// que a pessoa não tem como resolver, para sempre.
func TestUmaOrigemSemBeneficiosNaoCobraDois(t *testing.T) {
	f, id := oBarbaro(t, 1)
	var set setBuilder
	set.Add("origin = ?", "Amnésico")
	if err := set.exec(context.Background(), f.s.db, "UPDATE characters", id); err != nil {
		t.Fatalf("trocar a origem: %v", err)
	}

	tela := aTelaDosPoderes(t, f, id)
	if strings.Contains(tela, "Origem: 2 benefícios") {
		t.Error("o Amnésico cobra dois benefícios de uma lista que tem um")
	}
	if !strings.Contains(tela, "Origem: 1 benefício") {
		t.Error("o Amnésico não cobra o poder único que ele oferece")
	}
}

// UM BENEFÍCIO DE OUTRA ORIGEM não entra.
func TestUmBeneficioDeOutraOrigemNaoEntra(t *testing.T) {
	f, id := oBarbaro(t, 3)

	recusa := oComandoDoPoder(t, f, id, "origem/origin-acolito-pericia-Cura", "")
	if !strings.Contains(recusa, "Batedor") {
		t.Errorf("a recusa não diz qual é a origem da ficha: %q", recusa)
	}
}

// O CAMINHO e o DEVOTO só aceitam o que a classe oferece.
func TestOCaminhoEODevotoSoAceitamOQueAClasseOferece(t *testing.T) {
	f, id := oArcanista(t)

	if recusa := oComandoDoPoder(t, f, id, "classe/Arcanista/caminho/mago", ""); recusa != "" {
		t.Fatalf("um caminho do arcanista foi recusado: %q", recusa)
	}
	if recusa := oComandoDoPoder(t, f, id, "classe/Arcanista/caminho/bastiao", ""); recusa == "" {
		t.Error("o caminho do cavaleiro foi aceito num arcanista")
	}
	// E O ARCANISTA NÃO ESCOLHE DEVOTO: são três classes que escolhem (p57,
	// p61, p82), e ele não é nenhuma delas.
	if recusa := oComandoDoPoder(t, f, id, "classe/Arcanista/devoto/khalmyr", ""); recusa == "" {
		t.Error("um arcanista escolheu devoto")
	}
}

// AS PENDÊNCIAS dizem o que ainda falta, e a contagem sai no crachá.
func TestAsPendenciasDizemOQueFalta(t *testing.T) {
	f, id := oBarbaro(t, 3)

	tela := aTelaDosPoderes(t, f, id)
	for _, esperado := range []string{
		"2 poderes por escolher", // duas vagas, nenhuma usada
		"Origem: 2 benefícios por escolher",
		"escolhas pendentes",
	} {
		if !strings.Contains(tela, esperado) {
			t.Errorf("a tela não anuncia %q", esperado)
		}
	}

	// E ELAS SOMEM quando a escolha é feita.
	oComandoDoPoder(t, f, id, "escolhe/class.barbaro.golpe-poderoso", "")
	if tela := aTelaDosPoderes(t, f, id); !strings.Contains(tela, "1 poder por escolher") {
		t.Error("a pendência não desceu para uma vaga depois da escolha")
	}
}

// A PENDÊNCIA DO ATRIBUTO DE RAÇA é a promessa da forja ("dá para criar assim e
// terminar na ficha", ALE-169).
func TestAPendenciaDoAtributoDeRacaApareceEFecha(t *testing.T) {
	f, id := oBarbaro(t, 1)
	seedRaca(t, f.s, id, "Humano")

	if tela := aTelaDosPoderes(t, f, id); !strings.Contains(tela, "distribuir o bônus de atributo") {
		t.Fatal("o humano sem distribuição não mostra a pendência")
	}

	corpo := `{"racaatributos":["strength","dexterity","constitution"]}`
	if recusa := oComandoDoPoder(t, f, id, "atributos", corpo); recusa != "" {
		t.Fatalf("a distribuição foi recusada: %q", recusa)
	}
	if tela := aTelaDosPoderes(t, f, id); strings.Contains(tela, "distribuir o bônus de atributo") {
		t.Error("a pendência ficou depois de a distribuição fechar")
	}
}

// A DISTRIBUIÇÃO REPETIDA é recusada — o livro pede atributos DIFERENTES.
func TestADistribuicaoRepetidaERecusada(t *testing.T) {
	f, id := oBarbaro(t, 1)
	seedRaca(t, f.s, id, "Humano")

	corpo := `{"racaatributos":["strength","strength","strength"]}`
	if recusa := oComandoDoPoder(t, f, id, "atributos", corpo); !strings.Contains(recusa, "distintos") {
		t.Errorf("três vezes o mesmo atributo foi aceito: %q", recusa)
	}
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	if strings.Contains(row.Raceattributechoices, "strength") {
		t.Errorf("a recusa gravou assim mesmo: %s", row.Raceattributechoices)
	}
}

// O DIÁLOGO oferece o que cabe em cada aba.
func TestODialogoOfereceOQueCabeEmCadaAba(t *testing.T) {
	f, id := oArcanista(t)
	tela := aTelaDosPoderes(t, f, id)

	if !strings.Contains(tela, "Escolher poderes") {
		t.Fatal("a aba não tem o diálogo de escolher")
	}
	// O CAMINHO do arcanista, que ele escolhe desde o 1º nível.
	for _, esperado := range []string{"Bruxo", "Feiticeiro", "Mago"} {
		if !strings.Contains(tela, esperado) {
			t.Errorf("o diálogo não oferece o caminho %q", esperado)
		}
	}
	// E NÃO oferece o que é de outra classe.
	if strings.Contains(tela, "Égide Sagrada") {
		t.Error("o diálogo ofereceu um caminho de paladino a um arcanista")
	}
}

// A API JSON tem a MESMA fronteira, e ela era a porta que estava aberta.
//
// O pedido vai pelo roteador da API — `authed`, e não o `pede` do piloto: o
// `pede` monta o `WebRouter`, onde `/characters/...` não existe, e um 404
// passaria por "recusou" sem que a regra tivesse rodado. Medido: escrito assim,
// o teste continuava verde com a validação REMOVIDA.
func TestAApiJsonRecusaAEscolhaForaDaRegra(t *testing.T) {
	f, id := oBarbaro(t, 3)

	corpo := `{"classPowers":["class.barbaro.golpe-poderoso","class.barbaro.frenesi","class.barbaro.brado-assustador"]}`
	rec := authed(t, f.s, f.jogador, http.MethodPatch, abilitiesPath(id), corpo)
	if rec.Code == http.StatusOK {
		t.Fatalf("a API JSON aceitou três poderes em duas vagas: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "2 vagas") {
		t.Errorf("a recusa não diz quantas vagas existem: %s", rec.Body.String())
	}
	// CONTROLE: a mesma escrita com DOIS poderes passa — sem ele, um 404 ou um
	// 403 de rota errada leria como "a regra funcionou".
	cabe := `{"classPowers":["class.barbaro.golpe-poderoso","class.barbaro.frenesi"]}`
	if rec := authed(t, f.s, f.jogador, http.MethodPatch, abilitiesPath(id), cabe); rec.Code != http.StatusOK {
		t.Fatalf("dois poderes em duas vagas foram recusados: %d %s", rec.Code, rec.Body.String())
	}
}
