package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// Os guardas da MOCHILA (ALE-272, fatia 7).
//
// O que eles prendem é a REGRA e a DECISÃO de tela: os dois tetos da p141, o
// eixo de equipar, a conta do dinheiro, e o que a tira faz com um item que já
// está fora do teto. Os números da carga não são reafirmados aqui — eles vêm do
// motor e têm guarda de regra própria em `engine/carga_rules_test.go`.

// aTelaDaMochila é a aba desenhada.
func aTelaDaMochila(t *testing.T, f pilotoFixture, id int64) string {
	t.Helper()
	return f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=bag", id), "").Body.String()
}

// oComandoDaMochila dispara um gesto e devolve a recusa, ou "".
func oComandoDaMochila(t *testing.T, f pilotoFixture, id int64, caminho string) string {
	t.Helper()
	alvo := fmt.Sprintf("/personagens/%d/%s?tab=bag", id, caminho)
	return aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, "").Body.String())
}

// semeiaItem põe um item na ficha e devolve o id.
func semeiaItem(t *testing.T, f pilotoFixture, id int64, catalogo, nome string, equipado string) int64 {
	t.Helper()
	item, err := f.s.queries.CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: id, Catalogid: sql.NullString{String: catalogo, Valid: catalogo != ""},
		Name: nome, Quantity: 1, Slots: 1,
		Equipped:     sql.NullString{String: equipado, Valid: equipado != ""},
		Improvements: "[]", Createdat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o item %q: %v", nome, err)
	}
	return item.ID
}

// oEquipadoDe lê a coluna direto do banco — o teste não pergunta à tela o que
// aconteceu no banco.
func oEquipadoDe(t *testing.T, f pilotoFixture, itemID int64) string {
	t.Helper()
	item, err := f.s.queries.GetItem(context.Background(), itemID)
	if err != nil {
		t.Fatalf("ler o item %d: %v", itemID, err)
	}
	return item.Equipped.String
}

// A TIRA desenha os dois tetos do livro, e as posições vazias junto.
func TestATiraMostraOsDoisTetosDaP141(t *testing.T) {
	f, id := oCombatente(t)
	semeiaItem(t, f, id, "espada-longa", "Espada longa", "wielded")
	semeiaItem(t, f, id, "armadura-couro", "Armadura de couro", "vested")
	semeiaItem(t, f, id, "corda", "Corda", "")

	tela := aTelaDaMochila(t, f, id)
	for _, esperado := range []string{"Mãos", "1/2", "Vestidos", "1/4", "Mão principal", "Mão secundária"} {
		if !strings.Contains(tela, esperado) {
			t.Errorf("a tira não tem %q", esperado)
		}
	}
	// AS POSIÇÕES VAZIAS SÃO DESENHADAS, e são contadas: uma mão livre e três
	// vestidos livres. Só afirmar que a palavra "vazio" aparece deixaria passar
	// uma tira que desenha um lugar livre e esconde os outros três — e é
	// justamente a soma que mostra quanto ainda cabe.
	if livres := strings.Count(tela, ">vazio</p>"); livres != 4 {
		t.Errorf("a tira desenhou %d posições livres, quer 4 (1 mão + 3 vestidos)", livres)
	}
	// O que o item CONCEDE aparece no cartão: sem isso a tira vira uma lista de
	// nomes, e a pergunta que ela existe para responder é "o que isto me dá?".
	for _, cracha := range []string{"Dano 1d8", "Defesa +2"} {
		if !strings.Contains(tela, cracha) {
			t.Errorf("o cartão não mostra %q", cracha)
		}
	}
}

// O ITEM ACIMA DO TETO NÃO SOME — e este é o defeito que o porte conserta.
//
// A ficha antiga desenha `wielded[0]` e `wielded[1]` e para por aí: um terceiro
// item empunhado — que o banco tem, porque a semente escreveu direto — não
// aparecia em lugar nenhum. Nem na tira, nem na grade do guardado, porque ele
// está equipado. Medido na bancada contra a tela antiga: o Escudo leve do Thal
// existia no banco e não existia na tela.
func TestOItemAcimaDoTetoNaoSomeDaTela(t *testing.T) {
	f, id := oCombatente(t)
	semeiaItem(t, f, id, "espada-longa", "Espada longa", "wielded")
	semeiaItem(t, f, id, "adaga", "Adaga", "wielded")
	semeiaItem(t, f, id, "escudo-leve", "Escudo leve", "wielded")

	tela := aTelaDaMochila(t, f, id)
	if !strings.Contains(tela, "Escudo leve") {
		t.Error("o terceiro item empunhado sumiu da tela")
	}
	if !strings.Contains(tela, "Acima do limite") {
		t.Error("o item fora do teto não é nomeado como tal")
	}
	// O CONTADOR denuncia junto: 3/2 é o que faz a pessoa entender por que há
	// um cartão a mais.
	if !strings.Contains(tela, "3/2") {
		t.Error("o contador das mãos não mostra que passou do teto")
	}
}

// GUARDAR tira da mão e devolve à grade.
func TestGuardarTiraDaMaoEPoeNaGrade(t *testing.T) {
	f, id := oCombatente(t)
	item := semeiaItem(t, f, id, "espada-longa", "Espada longa", "wielded")

	if recusa := oComandoDaMochila(t, f, id, fmt.Sprintf("itens/%d/guarda", item)); recusa != "" {
		t.Fatalf("guardar foi recusado: %q", recusa)
	}
	if got := oEquipadoDe(t, f, item); got != "" {
		t.Errorf("o item continua equipado como %q", got)
	}
	if !strings.Contains(aTelaDaMochila(t, f, id), "0/2") {
		t.Error("a mão não ficou livre na tela")
	}
}

// EQUIPAR respeita o EIXO do item: um escudo não se veste.
func TestEquiparRespeitaOEixoDoItem(t *testing.T) {
	f, id := oCombatente(t)
	item := semeiaItem(t, f, id, "escudo-leve", "Escudo leve", "")

	recusa := oComandoDaMochila(t, f, id, fmt.Sprintf("itens/%d/equipa/vested", item))
	if !strings.Contains(recusa, "Escudo leve") {
		t.Errorf("a recusa não nomeia o item: %q", recusa)
	}
	if got := oEquipadoDe(t, f, item); got != "" {
		t.Errorf("a recusa equipou assim mesmo: %q", got)
	}
	// E o caminho que o livro permite continua aberto.
	if recusa := oComandoDaMochila(t, f, id, fmt.Sprintf("itens/%d/equipa/wielded", item)); recusa != "" {
		t.Errorf("empunhar um escudo foi recusado: %q", recusa)
	}
	if got := oEquipadoDe(t, f, item); got != "wielded" {
		t.Errorf("o escudo não foi empunhado: %q", got)
	}
}

// EQUIPAR respeita o TETO de duas mãos (p141), e a recusa chega à TELA.
func TestEquiparRespeitaOTetoDeDuasMaos(t *testing.T) {
	f, id := oCombatente(t)
	semeiaItem(t, f, id, "espada-longa", "Espada longa", "wielded")
	semeiaItem(t, f, id, "adaga", "Adaga", "wielded")
	terceiro := semeiaItem(t, f, id, "clava", "Clava", "")

	recusa := oComandoDaMochila(t, f, id, fmt.Sprintf("itens/%d/equipa/wielded", terceiro))
	if recusa == "" {
		t.Fatal("uma terceira mão foi aceita")
	}
	if !strings.Contains(recusa, "2") {
		t.Errorf("a recusa não diz qual é o teto: %q", recusa)
	}
	if got := oEquipadoDe(t, f, terceiro); got != "" {
		t.Errorf("a recusa equipou assim mesmo: %q", got)
	}
}

// UM ITEM DE OUTRA FICHA não se mexe por esta.
//
// A posse do PERSONAGEM já é do `comandoDaFicha`; a do ITEM é o que falta. Sem
// ela o id de outra ficha passaria — o `UPDATE` é por id, e afetar zero linhas
// não é erro que alguém veja.
func TestOItemDeOutraFichaNaoSeMexe(t *testing.T) {
	f, minha := oCombatente(t)
	outra := seedCharacterAtLevel(t, f.s, f.jogador, "Vizinho", 1, 10, 10, 0, 0)
	alheio := semeiaItem(t, f, outra, "espada-longa", "Espada longa", "wielded")

	if recusa := oComandoDaMochila(t, f, minha, fmt.Sprintf("itens/%d/guarda", alheio)); recusa == "" {
		t.Error("guardei o item de outra ficha pela minha")
	}
	if got := oEquipadoDe(t, f, alheio); got != "wielded" {
		t.Errorf("o item alheio foi mexido assim mesmo: %q", got)
	}
}

// O DINHEIRO: os três modos, o piso e o arredondamento.
func TestODinheiroRecebeGastaECorrige(t *testing.T) {
	f, id := oCombatente(t)
	if err := f.s.queries.SetCharacterTibar(context.Background(), sqlcgen.SetCharacterTibarParams{
		Tibar: 35.7, UpdatedAt: plataforma.NowISO(), ID: id,
	}); err != nil {
		t.Fatalf("semear o dinheiro: %v", err)
	}

	// GASTAR 0,3 sobre 35,7. Este par é escolhido, e não qualquer um: em ponto
	// flutuante ele dá 35,400000000000006, e é ESSE número que iria para o
	// banco e para a tela sem o arredondamento de duas casas. Um par ingênuo
	// não serve — medido, 1200,3 − 80,1 fecha exato em IEEE, e um teste escrito
	// com ele passa verde com o arredondamento removido.
	if recusa := oDinheiro(t, f, id, "gastar", 0.3); recusa != "" {
		t.Fatalf("gastar foi recusado: %q", recusa)
	}
	if got := oTibarDe(t, f, id); got != 35.4 {
		t.Errorf("o saldo ficou %v, quer 35.4 — a soma binária foi para o banco", got)
	}

	// E RECEBER tem a mesma armadilha do outro lado: 0,1 + 0,2 dá
	// 0,30000000000000004.
	if err := f.s.queries.SetCharacterTibar(context.Background(), sqlcgen.SetCharacterTibarParams{
		Tibar: 0.1, UpdatedAt: plataforma.NowISO(), ID: id,
	}); err != nil {
		t.Fatalf("semear o dinheiro: %v", err)
	}
	if recusa := oDinheiro(t, f, id, "receber", 0.2); recusa != "" {
		t.Fatalf("receber foi recusado: %q", recusa)
	}
	if got := oTibarDe(t, f, id); got != 0.3 {
		t.Errorf("receber deu %v, quer 0.3", got)
	}

	if recusa := oDinheiro(t, f, id, "corrigir", 42); recusa != "" {
		t.Fatalf("corrigir foi recusado: %q", recusa)
	}
	if got := oTibarDe(t, f, id); got != 42 {
		t.Errorf("corrigir deu %v, quer 42", got)
	}
}

// DÍVIDA NÃO EXISTE na ficha, e a razão é a carga.
//
// Saldo negativo viraria carga de moeda NEGATIVA, que COMPRARIA espaço na
// mochila em vez de ocupar (ALE-215). Por isso o piso é zero, e não um aviso.
func TestODinheiroNaoFicaNegativo(t *testing.T) {
	f, id := oCombatente(t)
	if err := f.s.queries.SetCharacterTibar(context.Background(), sqlcgen.SetCharacterTibarParams{
		Tibar: 50, UpdatedAt: plataforma.NowISO(), ID: id,
	}); err != nil {
		t.Fatalf("semear o dinheiro: %v", err)
	}

	recusa := oDinheiro(t, f, id, "gastar", 80)
	if !strings.Contains(recusa, "50") {
		t.Errorf("a recusa não diz quanto a pessoa tem: %q", recusa)
	}
	if got := oTibarDe(t, f, id); got != 50 {
		t.Errorf("a recusa gastou assim mesmo: %v", got)
	}
}

func oDinheiro(t *testing.T, f pilotoFixture, id int64, modo string, valor float64) string {
	t.Helper()
	corpo := fmt.Sprintf(`{"tibarmodo":%q,"tibarvalor":%v}`, modo, valor)
	alvo := fmt.Sprintf("/personagens/%d/dinheiro?tab=bag", id)
	return aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String())
}

func oTibarDe(t *testing.T, f pilotoFixture, id int64) float64 {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	return row.Tibar
}

// A BUSCA da grade ignora acento, e o CHIP filtra por categoria.
func TestAGradeFiltraPorBuscaEPorCategoria(t *testing.T) {
	f, id := oCombatente(t)
	semeiaItem(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")
	semeiaItem(t, f, id, "espada-longa", "Espada longa", "")

	comBusca := f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=bag&itembusca=balsamo", id), "").Body.String()
	if !strings.Contains(comBusca, "Bálsamo restaurador") {
		t.Error("a busca sem acento não achou o Bálsamo")
	}
	if strings.Contains(oGuardadoDaTela(comBusca), "Espada longa") {
		t.Error("a busca deixou passar o que não casa")
	}

	comChip := f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=bag&itemcategoria=weapons", id), "").Body.String()
	if !strings.Contains(oGuardadoDaTela(comChip), "Espada longa") {
		t.Error("o chip de armas escondeu a espada")
	}
	if strings.Contains(oGuardadoDaTela(comChip), "Bálsamo restaurador") {
		t.Error("o chip de armas deixou passar um consumível")
	}
}

// oGuardadoDaTela recorta só a GRADE, e o recorte não é preciosismo: o nome de
// um item aparece também nos crachás da tira e em CADA diálogo de ficha de
// item, que são desenhados logo depois do painel. Procurar na tela inteira
// acharia justamente o que o filtro escondeu, e o teste passaria dizendo o
// contrário do que mede.
//
// O corte vai da grade até o fim da SEÇÃO, que é onde o painel acaba e os
// diálogos começam.
func oGuardadoDaTela(tela string) string {
	inicio := strings.Index(tela, "grid-cols-3")
	if inicio < 0 {
		return ""
	}
	fim := strings.Index(tela[inicio:], "</section>")
	if fim < 0 {
		return tela[inicio:]
	}
	return tela[inicio : inicio+fim]
}
