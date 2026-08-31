package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
)

// Os guardas do painel de PERÍCIAS (ALE-272, fatia 4).
//
// O que eles prendem é a REGRA e a DECISÃO: a ordem que a mesa lê, o filtro que
// ignora acento, os quatro gestos de escrita e a fronteira de cada um. Os
// NÚMEROS são do motor e já têm o oráculo de paridade.

// oPericioso é uma ficha com perícias de verdade e um ofício inventado.
func oPericioso(t *testing.T) (pilotoFixture, int64) {
	t.Helper()
	f, id := oCombatente(t)
	seedPericia(t, f.s, id, "Acrobacia", "dexterity", false)
	seedPericia(t, f.s, id, "Atuação", "charisma", false)
	seedOficio(t, f.s, id, "Ferreiro", "intelligence")
	return f, id
}

func seedOficio(t *testing.T, s *Server, id int64, nome, atributo string) {
	t.Helper()
	_, err := s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: id, Name: nome, Attribute: atributo, Trained: 1, Custom: 1,
	})
	if err != nil {
		t.Fatalf("semear o ofício %q: %v", nome, err)
	}
}

func aTelaDasPericias(t *testing.T, f pilotoFixture, id int64, busca string) string {
	t.Helper()
	alvo := fmt.Sprintf("/piloto/personagens/%d?tab=expertises", id)
	if busca != "" {
		alvo += "&busca=" + url.QueryEscape(busca)
	}
	return f.pede(t, f.jogador, http.MethodGet, alvo, "").Body.String()
}

// aPericia manda um dos gestos e devolve a tela redesenhada.
func aPericia(t *testing.T, f pilotoFixture, id int64, caminho string) string {
	t.Helper()
	alvo := fmt.Sprintf("/piloto/personagens/%d/pericias/%s?tab=expertises", id, caminho)
	rec := f.pede(t, f.jogador, http.MethodPost, alvo, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o comando %q respondeu %d: %s", caminho, rec.Code, rec.Body.String())
	}
	return aTelaDasPericias(t, f, id, "")
}

// oTreinoDe lê o que o BANCO guarda, que é a única fonte da verdade do gesto.
func oTreinoDe(t *testing.T, f pilotoFixture, id int64, nome string) (treinada bool, atributo string) {
	t.Helper()
	todas, err := f.s.queries.ListExpertisesByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler as perícias: %v", err)
	}
	for _, e := range todas {
		if e.Name == nome {
			return e.Trained != 0, e.Attribute
		}
	}
	t.Fatalf("a perícia %q não está na ficha", nome)
	return false, ""
}

// O PAINEL CHEGA NA TELA com os números do motor.
//
// Os esperados são escritos À MÃO, do combatente de nível 3: ½ nível 1, Luta
// treinada com Força 4 e treino 2, e as demais sem treino.
//
//	Luta      = 1 + 4 + 2 = +7
//	Fortitude = 1 + 3     = +4
//	Acrobacia = 1 + 2     = +3
func TestOPainelDePericiasDizOsNumerosDoMotor(t *testing.T) {
	f, id := oPericioso(t)
	tela := aTelaDasPericias(t, f, id, "")

	if strings.Contains(tela, "ainda vive na ficha antiga") {
		t.Fatal("a aba de Perícias ainda manda para a ficha velha")
	}
	for _, esperado := range []string{"Detalhar Luta +7", "Detalhar Fortitude +4", "Detalhar Acrobacia +3"} {
		if !strings.Contains(tela, `aria-label="`+esperado+`"`) {
			t.Errorf("a tela não tem %q", esperado)
		}
	}
	// O CABEÇALHO diz as duas parcelas que valem para todas as linhas.
	if !strings.Contains(tela, "treino +2 • ½ nível 1") {
		t.Error("o cabeçalho não diz o treino e o ½ nível do personagem")
	}
}

// O TREINO POR NÍVEL tem três degraus, e o cabeçalho os diz.
func TestOCabecalhoDizOTreinoDoNivel(t *testing.T) {
	casos := []struct {
		nivel int64
		quer  int
	}{{1, 2}, {6, 2}, {7, 4}, {14, 4}, {15, 6}, {20, 6}}
	for _, caso := range casos {
		if got := trainingBonusFor(caso.nivel); got != caso.quer {
			t.Errorf("no nível %d o treino é %d, quer %d", caso.nivel, got, caso.quer)
		}
	}
}

// AS RESISTÊNCIAS VÊM PRIMEIRO, e os ofícios por último.
//
// "Teste de Reflexos CD 20" é a consulta mais quente da mesa. Prender a ordem é
// prender uma decisão de produto que uma reordenação alfabética inocente
// desfaria sem ninguém notar.
func TestAOrdemPoeAsResistenciasNaFrenteEOsOficiosNoFim(t *testing.T) {
	f, id := oPericioso(t)
	tela := aTelaDasPericias(t, f, id, "")

	posicao := func(nome string) int { return strings.Index(tela, `aria-label="`+nome+` treinada"`) }
	fort, refl, acro, oficio := posicao("Fortitude"), posicao("Reflexos"), posicao("Acrobacia"), posicao("Ferreiro")
	if fort < 0 || refl < 0 || acro < 0 || oficio < 0 {
		t.Fatalf("alguma linha não saiu: fort=%d refl=%d acro=%d oficio=%d", fort, refl, acro, oficio)
	}
	if !(fort < refl && refl < acro) {
		t.Error("as resistências não vieram antes do resto da lista")
	}
	if oficio < acro {
		t.Error("o ofício do jogador veio antes das perícias do livro")
	}
}

// A BUSCA IGNORA ACENTO, porque ninguém digita "Atuação" com o til.
func TestABuscaAchaSemAcentoESemCaixa(t *testing.T) {
	f, id := oPericioso(t)
	for _, termo := range []string{"atuacao", "ATUACAO", "Atuação", "tuaç"} {
		tela := aTelaDasPericias(t, f, id, termo)
		if !strings.Contains(tela, `aria-label="Atuação treinada"`) {
			t.Errorf("buscar %q não achou Atuação", termo)
		}
		if strings.Contains(tela, `aria-label="Fortitude treinada"`) {
			t.Errorf("buscar %q trouxe Fortitude junto: o filtro não filtra", termo)
		}
	}
	// SEM ACHADO a lista diz isso, em vez de ficar vazia parecendo defeito.
	if vazia := aTelaDasPericias(t, f, id, "zzz"); !strings.Contains(vazia, "Nenhuma perícia para") {
		t.Error("uma busca sem achado deixou a lista muda")
	}
}

// O `@get` DA BUSCA CARREGA A ABA.
//
// A varredura `TestNenhumComandoDaFichaPerdeAAba` olha só os `@post`, então este
// caminho precisa de guarda próprio: sem o `?tab=`, digitar na busca devolveria
// a ficha desenhada na PRIMEIRA aba — e a primeira aba É Perícias, o que faria o
// defeito parecer funcionar até alguém buscar de outra seção.
func TestOGetDaBuscaCarregaAAba(t *testing.T) {
	f, id := oPericioso(t)
	tela := aTelaDasPericias(t, f, id, "")
	if !strings.Contains(tela, "?tab=expertises&#39;)") {
		t.Error("o `@get` da busca não carrega o `?tab=`: a resposta viria noutra aba")
	}
}

// O TREINO ALTERNA, e o comando manda a perícia e não o estado.
func TestOTreinoAlternaNosDoisSentidos(t *testing.T) {
	f, id := oPericioso(t)
	if treinada, _ := oTreinoDe(t, f, id, "Acrobacia"); treinada {
		t.Fatal("a Acrobacia começou treinada: o caso não mede a ida")
	}

	aPericia(t, f, id, "treino/"+url.PathEscape("Acrobacia"))
	if treinada, _ := oTreinoDe(t, f, id, "Acrobacia"); !treinada {
		t.Error("o primeiro toque não treinou a Acrobacia")
	}
	aPericia(t, f, id, "treino/"+url.PathEscape("Acrobacia"))
	if treinada, _ := oTreinoDe(t, f, id, "Acrobacia"); treinada {
		t.Error("o segundo toque não destreinou a Acrobacia: o comando manda o ESTADO em vez da perícia")
	}
}

// O ATRIBUTO TROCA, e um atributo inventado é recusado.
func TestOAtributoTrocaESoAceitaOsSeis(t *testing.T) {
	f, id := oPericioso(t)
	aPericia(t, f, id, "atributo/"+url.PathEscape("Acrobacia")+"/strength")
	if _, atributo := oTreinoDe(t, f, id, "Acrobacia"); atributo != "strength" {
		t.Errorf("a Acrobacia ficou em %q, quer strength", atributo)
	}

	alvo := fmt.Sprintf("/piloto/personagens/%d/pericias/atributo/Acrobacia/sorte?tab=expertises", id)
	if recusa := aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, "").Body.String()); recusa == "" {
		t.Error("um atributo inventado foi aceito sem uma palavra na tela")
	}
	if _, atributo := oTreinoDe(t, f, id, "Acrobacia"); atributo != "strength" {
		t.Error("a recusa mexeu no banco assim mesmo")
	}
}

// O OFÍCIO ACEITA TREINO E ATRIBUTO, e este guarda é o conserto de um defeito
// que a ficha ANTIGA carrega até hoje (ALE-272).
//
// O `handleUpdateExpertise` exigia que o nome fosse uma das 29 do livro, e a SPA
// desenhava o botão de treino e o seletor em toda linha — nos ofícios os dois
// davam 400. Promessa de tela que o servidor não cumpria, e ninguém tinha
// notado porque nenhum teste mexia num ofício depois de criá-lo.
func TestOOficioAceitaTreinoEAtributo(t *testing.T) {
	f, id := oPericioso(t)

	aPericia(t, f, id, "treino/Ferreiro")
	if treinada, _ := oTreinoDe(t, f, id, "Ferreiro"); treinada {
		t.Error("o ofício não destreinou: o servidor recusa editar o que não é do livro")
	}
	aPericia(t, f, id, "atributo/Ferreiro/dexterity")
	if _, atributo := oTreinoDe(t, f, id, "Ferreiro"); atributo != "dexterity" {
		t.Errorf("o ofício ficou em %q, quer dexterity: o servidor recusou a troca", atributo)
	}
}

// O OFÍCIO NASCE TREINADO E SE REMOVE; a perícia do livro NÃO se remove.
//
// A recusa é do SERVIDOR e não da tela: a ficha não desenha lixeira numa perícia
// do livro, mas travar só na UI deixaria a regra sem fronteira — quem montar o
// `@post` à mão apagaria a Fortitude.
func TestOOficioNasceTreinadoESoEleSeRemove(t *testing.T) {
	f, id := oPericioso(t)
	if treinada, _ := oTreinoDe(t, f, id, "Ferreiro"); !treinada {
		t.Error("o ofício não nasceu treinado")
	}

	aPericia(t, f, id, "remove/Ferreiro")
	todas, _ := f.s.queries.ListExpertisesByCharacter(context.Background(), id)
	for _, e := range todas {
		if e.Name == "Ferreiro" {
			t.Fatal("o ofício sobreviveu ao remover")
		}
	}

	alvo := fmt.Sprintf("/piloto/personagens/%d/pericias/remove/Fortitude?tab=expertises", id)
	if recusa := aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, "").Body.String()); recusa == "" {
		t.Error("uma perícia do LIVRO foi removida da ficha")
	}
	if _, _ = oTreinoDe(t, f, id, "Fortitude"); false {
		t.Error("inalcançável")
	}
}

// A REGRA DO NOME É UMA SÓ para a API JSON e para a ficha (ALE-272).
//
// Um ofício não pode ROUBAR o nome de uma das 29: a ficha passaria a ter duas
// linhas com o mesmo nome, e a decomposição de uma cairia sobre a outra.
func TestOOficioNaoRoubaONomeDeUmaPericiaDoLivro(t *testing.T) {
	f, id := oPericioso(t)
	casos := []struct {
		nome string
		erro string
	}{
		{"", "dê um nome"},
		{"Fortitude", "é uma perícia do livro"},
		{"Ferreiro", "já tem"},
	}
	for _, caso := range casos {
		err := f.s.guardaOOficioNovo(context.Background(), id, caso.nome)
		if err == nil {
			t.Errorf("o nome %q foi aceito", caso.nome)
			continue
		}
		if !strings.Contains(err.Error(), caso.erro) {
			t.Errorf("o nome %q deu %q, e a mensagem devia falar de %q", caso.nome, err, caso.erro)
		}
	}
	if err := f.s.guardaOOficioNovo(context.Background(), id, "Marinheiro"); err != nil {
		t.Errorf("um nome legítimo foi recusado: %v", err)
	}
}
