package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
)

// Os guardas dos DIÁLOGOS da Mochila (ALE-272, fatia 7).
//
// A ficha de item, o catálogo, o item custom, a dose do consumível e — a parte
// que era só de tela até aqui — a compatibilidade de melhoria e material.

// oItemDaFichaPorNome acha um item pelo nome, para o teste não guardar ids.
func oItemDaFichaPorNome(t *testing.T, f pilotoFixture, id int64, nome string) sqlcgen.ListItemsByCharacterRow {
	t.Helper()
	itens, err := f.s.queries.ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar os itens: %v", err)
	}
	for _, item := range itens {
		if item.Name == nome {
			return item
		}
	}
	t.Fatalf("o item %q não está na ficha", nome)
	return sqlcgen.ListItemsByCharacterRow{}
}

// A FICHA DO ITEM oferece os lugares ALCANÇÁVEIS, e só eles.
func TestAFichaDoItemOfereceOsLugaresAlcancaveis(t *testing.T) {
	f, id := oCombatente(t)
	semeiaItem(t, f, id, "adaga", "Adaga", "")
	semeiaItem(t, f, id, "montante", "Montante", "")
	semeiaItem(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	tela := aTelaDaMochila(t, f, id)
	// UMA MÃO: a adaga é de uma mão e não é versátil, então ocupar as duas não
	// ganharia nada (p150) e a opção não existe.
	if !strings.Contains(tela, "Empunhar (1 mão)") {
		t.Error("a adaga não oferece empunhar com uma mão")
	}
	// DUAS MÃOS, obrigatórias: o montante não cabe numa mão só.
	if !strings.Contains(tela, "Empunhar (2 mãos)") {
		t.Error("o montante não oferece as duas mãos")
	}
	// O CONSUMÍVEL não se equipa em lugar nenhum, e por isso ele mostra o USAR
	// no lugar do bloco de equipar.
	if !strings.Contains(tela, "Usar Bálsamo restaurador") {
		t.Error("o consumível não oferece o Usar")
	}
	// E O ESTADO ATUAL não é oferecido: um item guardado com um botão "Guardar"
	// é um controle que não faz nada.
	if strings.Contains(oFichaDoItemNaTela(tela, "Adaga"), ">Guardar<") {
		t.Error("um item já guardado oferece Guardar")
	}
}

// oFichaDoItemNaTela recorta o diálogo de UM item pelo rótulo dele.
func oFichaDoItemNaTela(tela, nome string) string {
	inicio := strings.Index(tela, `aria-label="`+nome+`"`)
	if inicio < 0 {
		return ""
	}
	fim := strings.Index(tela[inicio:], "</div></div>")
	if fim < 0 {
		return tela[inicio:]
	}
	return tela[inicio : inicio+fim]
}

// ADICIONAR DO CATÁLOGO usa o nome e os espaços DO LIVRO.
//
// Deixar o cliente mandá-los abriria a porta para uma "Espada longa" de zero
// espaços, que é carga de graça na mochila.
func TestAdicionarDoCatalogoUsaOsNumerosDoLivro(t *testing.T) {
	f, id := oCombatente(t)

	if recusa := oComandoDaMochila(t, f, id, "itens/adiciona/espada-longa"); recusa != "" {
		t.Fatalf("adicionar foi recusado: %q", recusa)
	}
	item := oItemDaFichaPorNome(t, f, id, "Espada longa")
	if item.Slots != 1 {
		t.Errorf("a espada entrou com %v espaços, e o livro diz 1", item.Slots)
	}
	if item.Catalogid.String != "espada-longa" {
		t.Errorf("o item não guardou o id do catálogo: %q", item.Catalogid.String)
	}
	if item.Quantity != 1 {
		t.Errorf("a quantidade padrão virou %d", item.Quantity)
	}
}

// UM ITEM INVENTADO não entra pelo catálogo.
func TestUmItemInventadoNaoEntraPeloCatalogo(t *testing.T) {
	f, id := oCombatente(t)

	if recusa := oComandoDaMochila(t, f, id, "itens/adiciona/espada-de-luz"); recusa == "" {
		t.Error("um item que não existe no livro foi aceito")
	}
	itens, err := f.s.queries.ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(itens) != 0 {
		t.Errorf("a recusa gravou assim mesmo: %d itens", len(itens))
	}
}

// O ITEM CUSTOM exige nome, e os espaços são múltiplos de meio (p141).
func TestOItemCustomExigeNomeEEspacosDeMeioEmMeio(t *testing.T) {
	f, id := oCombatente(t)

	if recusa := oItemCustom(t, f, id, `{"itemnome":"  ","itemqtd":1,"itemespacos":1}`); recusa == "" {
		t.Error("um item sem nome foi aceito")
	}
	if recusa := oItemCustom(t, f, id, `{"itemnome":"Pena","itemqtd":1,"itemespacos":0.3}`); recusa == "" {
		t.Error("0,3 espaço foi aceito, e o livro conta de meio em meio")
	}
	if recusa := oItemCustom(t, f, id, `{"itemnome":"Pena","itemqtd":1,"itemespacos":0.5}`); recusa != "" {
		t.Fatalf("meio espaço foi recusado: %q", recusa)
	}
	item := oItemDaFichaPorNome(t, f, id, "Pena")
	if item.Slots != 0.5 || item.Catalogid.Valid {
		t.Errorf("o item custom entrou errado: espaços %v, catálogo %q", item.Slots, item.Catalogid.String)
	}
}

func oItemCustom(t *testing.T, f pilotoFixture, id int64, corpo string) string {
	t.Helper()
	alvo := fmt.Sprintf("/piloto/personagens/%d/itens/custom?tab=bag", id)
	return aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String())
}

// EDITAR muda os três campos, e REMOVER tira da ficha.
func TestEditarERemoverUmItem(t *testing.T) {
	f, id := oCombatente(t)
	item := semeiaItem(t, f, id, "", "Lembrança", "")

	alvo := fmt.Sprintf("/piloto/personagens/%d/itens/%d/edita?tab=bag", id, item)
	corpo := `{"itemnome":"Lembrança da Ana","itemqtd":3,"itemespacos":0.5}`
	if recusa := aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String()); recusa != "" {
		t.Fatalf("editar foi recusado: %q", recusa)
	}
	editado := oItemDaFichaPorNome(t, f, id, "Lembrança da Ana")
	if editado.Quantity != 3 || editado.Slots != 0.5 {
		t.Errorf("a edição gravou %d × %v", editado.Quantity, editado.Slots)
	}

	if recusa := oComandoDaMochila(t, f, id, fmt.Sprintf("itens/%d/remove", item)); recusa != "" {
		t.Fatalf("remover foi recusado: %q", recusa)
	}
	itens, err := f.s.queries.ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(itens) != 0 {
		t.Errorf("o item sobreviveu ao remover: %d na ficha", len(itens))
	}
}

// USAR gasta a dose e aplica o que a MESA rolou, preso no máximo.
func TestUsarGastaADoseEAplicaARolagemDaMesa(t *testing.T) {
	f := novoPiloto(t)
	id := seedCharacterAtLevel(t, f.s, f.jogador, "Ferido", 3, 10, 30, 0, 0)
	item := semeiaItem(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	if recusa := oUso(t, f, id, item, `{"itemrolagempv":7}`); recusa != "" {
		t.Fatalf("usar foi recusado: %q", recusa)
	}
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	if row.Hpcurrent != 17 {
		t.Errorf("o PV ficou %d, quer 17 (10 + os 7 que a mesa rolou)", row.Hpcurrent)
	}
	// A DOSE FOI GASTA: era uma só, então a linha sai da ficha.
	itens, err := f.s.queries.ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(itens) != 0 {
		t.Error("a dose foi usada e o item continua na mochila")
	}
}

// A CURA NÃO PASSA DO MÁXIMO, e é o motor que prende.
func TestUsarNaoPassaDoPvMaximo(t *testing.T) {
	f := novoPiloto(t)
	id := seedCharacterAtLevel(t, f.s, f.jogador, "Quase cheio", 3, 28, 30, 0, 0)
	item := semeiaItem(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	if recusa := oUso(t, f, id, item, `{"itemrolagempv":8}`); recusa != "" {
		t.Fatalf("usar foi recusado: %q", recusa)
	}
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	if row.Hpcurrent != 30 {
		t.Errorf("o PV ficou %d, quer 30 — a cura passou do máximo", row.Hpcurrent)
	}
}

// O QUE NÃO É CONSUMÍVEL não se usa.
func TestOQueNaoEConsumivelNaoSeUsa(t *testing.T) {
	f, id := oCombatente(t)
	item := semeiaItem(t, f, id, "espada-longa", "Espada longa", "")

	if recusa := oUso(t, f, id, item, "{}"); !strings.Contains(recusa, "consumível") {
		t.Errorf("a recusa não diz o motivo: %q", recusa)
	}
	if _, err := f.s.queries.GetItem(context.Background(), item); err != nil {
		t.Error("a espada foi consumida assim mesmo")
	}
}

func oUso(t *testing.T, f pilotoFixture, id, item int64, corpo string) string {
	t.Helper()
	alvo := fmt.Sprintf("/piloto/personagens/%d/itens/%d/usa?tab=bag", id, item)
	return aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String())
}

// A MELHORIA QUE NÃO CABE É RECUSADA PELO SERVIDOR — e esta é a fronteira que
// a fatia fecha.
//
// A compatibilidade vivia só no filtro do diálogo da SPA: o Go tinha o campo
// `appliesTo` e não o lia, e o próprio `handleAddItem` registra a dívida em
// comentário. Filtro de tela não recusa nada — um pedido montado à mão punha
// corda de arco num escudo, e o servidor gravava.
func TestAMelhoriaQueNaoCabeERecusadaPeloServidor(t *testing.T) {
	f, id := oCombatente(t)
	escudo := semeiaItem(t, f, id, "escudo-leve", "Escudo leve", "")

	// A Certeira é de ARMA (`appliesTo: ["weapon"]`).
	recusa := asMelhorias(t, f, id, escudo, `{"itemmelhorias":["melhoria-certeira"]}`)
	if !strings.Contains(recusa, "Certeira") || !strings.Contains(recusa, "Escudo leve") {
		t.Errorf("a recusa não nomeia os dois lados: %q", recusa)
	}
	if guardadas := asMelhoriasDoItem(t, f, escudo); guardadas != "[]" {
		t.Errorf("a recusa gravou assim mesmo: %q", guardadas)
	}

	// E a que CABE passa: o guarda não pode estar recusando tudo.
	if recusa := asMelhorias(t, f, id, escudo, `{"itemmelhorias":["melhoria-reforcada"]}`); recusa != "" {
		t.Fatalf("uma melhoria de escudo foi recusada: %q", recusa)
	}
	if guardadas := asMelhoriasDoItem(t, f, escudo); !strings.Contains(guardadas, "melhoria-reforcada") {
		t.Errorf("a melhoria que cabe não foi gravada: %q", guardadas)
	}
}

// UMA POÇÃO NÃO RECEBE MELHORIA: não se forja um bálsamo em aço-rubi.
//
// Medido ao sabotar: tirar SÓ o portão das categorias fechadas
// (`aceitaMelhoria`) deixa este caso verde, porque a regra de FAMÍLIA pega o
// mesmo pedido — hoje toda melhoria e todo material do catálogo declaram
// `appliesTo`. O portão continua valendo por duas razões, e nenhuma é esta
// asserção: ele é quem esconde o BOTÃO na tela, e é o que segura uma
// sobreposição futura que chegue sem `appliesTo` — que, pela regra de "sem
// restrição serve a todos", entraria numa poção.
func TestUmConsumivelNaoRecebeMelhoria(t *testing.T) {
	f, id := oCombatente(t)
	balsamo := semeiaItem(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	recusa := asMelhorias(t, f, id, balsamo, `{"itemmaterial":"material-aco-rubi"}`)
	if recusa == "" {
		t.Error("um consumível recebeu material")
	}
	// E o DIÁLOGO nem oferece o botão — a tela e o servidor concordam.
	if strings.Contains(aTelaDaMochila(t, f, id), "Melhorias de Bálsamo restaurador") {
		t.Error("a tela oferece melhorias para um consumível")
	}
}

// UMA MELHORIA INVENTADA não entra.
func TestUmaMelhoriaInventadaNaoEntra(t *testing.T) {
	f, id := oCombatente(t)
	espada := semeiaItem(t, f, id, "espada-longa", "Espada longa", "")

	if recusa := asMelhorias(t, f, id, espada, `{"itemmelhorias":["melhoria-lendaria"]}`); recusa == "" {
		t.Error("uma melhoria que não existe foi aceita")
	}
	if guardadas := asMelhoriasDoItem(t, f, espada); guardadas != "[]" {
		t.Errorf("a recusa gravou assim mesmo: %q", guardadas)
	}
}

// O DIÁLOGO só oferece o que cabe na família do item.
func TestODialogoDeMelhoriasSoOfereceOQueCabe(t *testing.T) {
	f, id := oCombatente(t)
	semeiaItem(t, f, id, "espada-longa", "Espada longa", "")
	tela := aTelaDaMochila(t, f, id)
	daEspada := oDialogoDeMelhoriasNaTela(tela, "Espada longa")

	if daEspada == "" {
		t.Fatal("a espada não tem diálogo de melhorias: nada abaixo mediria coisa alguma")
	}
	if !strings.Contains(daEspada, "Certeira") {
		t.Error("a melhoria de arma não é oferecida à espada")
	}
	// A Reforçada é de armadura e escudo, e não de arma.
	if strings.Contains(daEspada, "Reforçada") {
		t.Error("uma melhoria de armadura foi oferecida a uma arma")
	}
}

func oDialogoDeMelhoriasNaTela(tela, nome string) string {
	inicio := strings.Index(tela, `aria-label="Melhorias de `+nome+`"`)
	if inicio < 0 {
		return ""
	}
	fim := strings.Index(tela[inicio:], "Aplicar")
	if fim < 0 {
		return tela[inicio:]
	}
	return tela[inicio : inicio+fim]
}

func asMelhorias(t *testing.T, f pilotoFixture, id, item int64, corpo string) string {
	t.Helper()
	alvo := fmt.Sprintf("/piloto/personagens/%d/itens/%d/melhorias?tab=bag", id, item)
	return aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String())
}

func asMelhoriasDoItem(t *testing.T, f pilotoFixture, item int64) string {
	t.Helper()
	row, err := f.s.queries.GetItem(context.Background(), item)
	if err != nil {
		t.Fatalf("ler o item: %v", err)
	}
	return row.Improvements
}
