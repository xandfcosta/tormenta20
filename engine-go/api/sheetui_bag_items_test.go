package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/db/sqlcgen"
	"testing"
)

func TestTheItemCardOffersTheReachablePlaces(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "adaga", "Adaga", "")
	itemSemeia(t, f, id, "montante", "Montante", "")
	itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	tela := bagScreen(t, f, id)
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
	if strings.Contains(itemScreenSheet(tela, "Adaga"), ">Guardar<") {
		t.Error("um item já guardado oferece Guardar")
	}
}

// UM ITEM INVENTADO não entra pelo catálogo.
func TestAnInventedItemDoesNotEnterThroughTheCatalog(t *testing.T) {
	f, id := fighterFixture(t)

	if recusa := bagCommand(t, f, id, "itens/adiciona/espada-de-luz"); recusa == "" {
		t.Error("um item que não existe no livro foi aceito")
	}
	itens, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(itens) != 0 {
		t.Errorf("a recusa gravou assim mesmo: %d itens", len(itens))
	}
}

// O DIÁLOGO só oferece o que cabe na família do item.
func TestTheImprovementsDialogOnlyOffersWhatFits(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	tela := bagScreen(t, f, id)
	daEspada := improvementScreenDialog(tela, "Espada longa")

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

func sheetNameItem(t *testing.T, f pilotoFixture, id int64, nome string) sqlcgen.ListItemsByCharacterRow {
	t.Helper()
	itens, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
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

// ADICIONAR DO CATÁLOGO usa o nome e os espaços DO LIVRO.
//
// Deixar o cliente mandá-los abriria a porta para uma "Espada longa" de zero
// espaços, que é carga de graça na mochila.
func TestAddingFromTheCatalogUsesTheBookNumbers(t *testing.T) {
	f, id := fighterFixture(t)

	if recusa := bagCommand(t, f, id, "itens/adiciona/espada-longa"); recusa != "" {
		t.Fatalf("adicionar foi recusado: %q", recusa)
	}
	item := sheetNameItem(t, f, id, "Espada longa")
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

// O ITEM CUSTOM exige nome, e os espaços são múltiplos de meio (p141).
func TestACustomItemRequiresANameAndHalfStepSlots(t *testing.T) {
	f, id := fighterFixture(t)

	if recusa := customItem(t, f, id, `{"itemnome":"  ","itemqtd":1,"itemespacos":1}`); recusa == "" {
		t.Error("um item sem nome foi aceito")
	}
	if recusa := customItem(t, f, id, `{"itemnome":"Pena","itemqtd":1,"itemespacos":0.3}`); recusa == "" {
		t.Error("0,3 espaço foi aceito, e o livro conta de meio em meio")
	}
	if recusa := customItem(t, f, id, `{"itemnome":"Pena","itemqtd":1,"itemespacos":0.5}`); recusa != "" {
		t.Fatalf("meio espaço foi recusado: %q", recusa)
	}
	item := sheetNameItem(t, f, id, "Pena")
	if item.Slots != 0.5 || item.Catalogid.Valid {
		t.Errorf("o item custom entrou errado: espaços %v, catálogo %q", item.Slots, item.Catalogid.String)
	}
}

func customItem(t *testing.T, f pilotoFixture, id int64, corpo string) string {
	t.Helper()
	alvo := fmt.Sprintf("/personagens/%d/itens/custom?tab=bag", id)
	return sceneRefusal(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String())
}

// EDITAR muda os três campos, e REMOVER tira da ficha.
func TestEditingAndRemovingAnItem(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, "", "Lembrança", "")

	alvo := fmt.Sprintf("/personagens/%d/itens/%d/edita?tab=bag", id, item)
	corpo := `{"itemnome":"Lembrança da Ana","itemqtd":3,"itemespacos":0.5}`
	if recusa := sceneRefusal(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String()); recusa != "" {
		t.Fatalf("editar foi recusado: %q", recusa)
	}
	editado := sheetNameItem(t, f, id, "Lembrança da Ana")
	if editado.Quantity != 3 || editado.Slots != 0.5 {
		t.Errorf("a edição gravou %d × %v", editado.Quantity, editado.Slots)
	}

	if recusa := bagCommand(t, f, id, fmt.Sprintf("itens/%d/remove", item)); recusa != "" {
		t.Fatalf("remover foi recusado: %q", recusa)
	}
	itens, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(itens) != 0 {
		t.Errorf("o item sobreviveu ao remover: %d na ficha", len(itens))
	}
}

// USAR gasta a dose e aplica o que a MESA rolou, preso no máximo.
func TestUsingSpendsTheDoseAndAppliesTheTableRoll(t *testing.T) {
	f := novoPiloto(t)
	id := seedCharacterAtLevel(t, f.s, f.jogador, "Ferido", 3, 10, 30, 0, 0)
	item := itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	if recusa := use(t, f, id, item, `{"itemrolagempv":7}`); recusa != "" {
		t.Fatalf("usar foi recusado: %q", recusa)
	}
	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	if row.Hpcurrent != 17 {
		t.Errorf("o PV ficou %d, quer 17 (10 + os 7 que a mesa rolou)", row.Hpcurrent)
	}
	// A DOSE FOI GASTA: era uma só, então a linha sai da ficha.
	itens, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(itens) != 0 {
		t.Error("a dose foi usada e o item continua na mochila")
	}
}

// A CURA NÃO PASSA DO MÁXIMO, e é o motor que prende.
func TestUsingDoesNotGoPastMaximumHp(t *testing.T) {
	f := novoPiloto(t)
	id := seedCharacterAtLevel(t, f.s, f.jogador, "Quase cheio", 3, 28, 30, 0, 0)
	item := itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	if recusa := use(t, f, id, item, `{"itemrolagempv":8}`); recusa != "" {
		t.Fatalf("usar foi recusado: %q", recusa)
	}
	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	if row.Hpcurrent != 30 {
		t.Errorf("o PV ficou %d, quer 30 — a cura passou do máximo", row.Hpcurrent)
	}
}

// O QUE NÃO É CONSUMÍVEL não se usa.
func TestWhatIsNotConsumableCannotBeUsed(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")

	if recusa := use(t, f, id, item, "{}"); !strings.Contains(recusa, "consumível") {
		t.Errorf("a recusa não diz o motivo: %q", recusa)
	}
	if _, err := f.s.sceneCore().Queries().GetItem(context.Background(), item); err != nil {
		t.Error("a espada foi consumida assim mesmo")
	}
}

func use(t *testing.T, f pilotoFixture, id, item int64, corpo string) string {
	t.Helper()
	alvo := fmt.Sprintf("/personagens/%d/itens/%d/usa?tab=bag", id, item)
	return sceneRefusal(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String())
}

// A MELHORIA QUE NÃO CABE É RECUSADA PELO SERVIDOR — e esta é a fronteira que
// a fatia fecha.
//
// A compatibilidade vivia só no filtro do diálogo da SPA: o Go tinha o campo
// `appliesTo` e não o lia, e o próprio `handleAddItem` registra a dívida em
// comentário. Filtro de tela não recusa nada — um pedido montado à mão punha
// corda de arco num escudo, e o servidor gravava.
func TestAnImprovementThatDoesNotFitIsRefusedByTheServer(t *testing.T) {
	f, id := fighterFixture(t)
	escudo := itemSemeia(t, f, id, "escudo-leve", "Escudo leve", "")

	// A Certeira é de ARMA (`appliesTo: ["weapon"]`).
	recusa := improvements(t, f, id, escudo, `{"itemmelhorias":["melhoria-certeira"]}`)
	if !strings.Contains(recusa, "Certeira") || !strings.Contains(recusa, "Escudo leve") {
		t.Errorf("a recusa não nomeia os dois lados: %q", recusa)
	}
	if guardadas := itemImprovements(t, f, escudo); guardadas != "[]" {
		t.Errorf("a recusa gravou assim mesmo: %q", guardadas)
	}

	// E a que CABE passa: o guarda não pode estar recusando tudo.
	if recusa := improvements(t, f, id, escudo, `{"itemmelhorias":["melhoria-reforcada"]}`); recusa != "" {
		t.Fatalf("uma melhoria de escudo foi recusada: %q", recusa)
	}
	if guardadas := itemImprovements(t, f, escudo); !strings.Contains(guardadas, "melhoria-reforcada") {
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
func TestAConsumableTakesNoImprovement(t *testing.T) {
	f, id := fighterFixture(t)
	balsamo := itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	recusa := improvements(t, f, id, balsamo, `{"itemmaterial":"material-aco-rubi"}`)
	if recusa == "" {
		t.Error("um consumível recebeu material")
	}
	// E o DIÁLOGO nem oferece o botão — a tela e o servidor concordam.
	if strings.Contains(bagScreen(t, f, id), "Melhorias de Bálsamo restaurador") {
		t.Error("a tela oferece melhorias para um consumível")
	}
}

// UMA MELHORIA INVENTADA não entra.
func TestAnInventedImprovementDoesNotEnter(t *testing.T) {
	f, id := fighterFixture(t)
	espada := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")

	if recusa := improvements(t, f, id, espada, `{"itemmelhorias":["melhoria-lendaria"]}`); recusa == "" {
		t.Error("uma melhoria que não existe foi aceita")
	}
	if guardadas := itemImprovements(t, f, espada); guardadas != "[]" {
		t.Errorf("a recusa gravou assim mesmo: %q", guardadas)
	}
}

func improvements(t *testing.T, f pilotoFixture, id, item int64, corpo string) string {
	t.Helper()
	alvo := fmt.Sprintf("/personagens/%d/itens/%d/melhorias?tab=bag", id, item)
	return sceneRefusal(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String())
}

func itemImprovements(t *testing.T, f pilotoFixture, item int64) string {
	t.Helper()
	row, err := f.s.sceneCore().Queries().GetItem(context.Background(), item)
	if err != nil {
		t.Fatalf("ler o item: %v", err)
	}
	return row.Improvements
}
