package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/infra/db/sqlcgen"
	"testing"
)

func TestTheItemCardOffersTheReachablePlaces(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "adaga", "Adaga", "")
	itemSemeia(t, f, id, "montante", "Montante", "")
	itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	screen := bagScreen(t, f, id)
	// UMA MÃO: a adaga é de uma mão e não é versátil, então ocupar as duas não
	// ganharia nada (p150) e a opção não existe.
	if !strings.Contains(screen, "Empunhar (1 mão)") {
		t.Error("a adaga não oferece empunhar com uma mão")
	}
	// DUAS MÃOS, obrigatórias: o montante não cabe numa mão só.
	if !strings.Contains(screen, "Empunhar (2 mãos)") {
		t.Error("o montante não oferece as duas mãos")
	}
	// O CONSUMÍVEL não se equipa em lugar nenhum, e por isso ele mostra o USAR
	// no lugar do bloco de equipar.
	if !strings.Contains(screen, "Usar Bálsamo restaurador") {
		t.Error("o consumível não oferece o Usar")
	}
	// E O ESTADO ATUAL não é oferecido: um item guardado com um botão "Guardar"
	// é um controle que não faz nada.
	if strings.Contains(itemScreenSheet(screen, "Adaga"), ">Guardar<") {
		t.Error("um item já guardado oferece Guardar")
	}
}

// UM ITEM INVENTADO não entra pelo catálogo.
func TestAnInventedItemDoesNotEnterThroughTheCatalog(t *testing.T) {
	f, id := fighterFixture(t)

	if refusal := bagCommand(t, f, id, "itens/adiciona/espada-de-luz"); refusal == "" {
		t.Error("um item que não existe no livro foi aceito")
	}
	items, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("a recusa gravou assim mesmo: %d itens", len(items))
	}
}

// O DIÁLOGO só oferece o que cabe na família do item.
func TestTheImprovementsDialogOnlyOffersWhatFits(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	screen := bagScreen(t, f, id)
	fromSword := improvementScreenDialog(screen, "Espada longa")

	if fromSword == "" {
		t.Fatal("a espada não tem diálogo de melhorias: nada abaixo mediria coisa alguma")
	}
	if !strings.Contains(fromSword, "Certeira") {
		t.Error("a melhoria de arma não é oferecida à espada")
	}
	// A Reforçada é de armadura e escudo, e não de arma.
	if strings.Contains(fromSword, "Reforçada") {
		t.Error("uma melhoria de armadura foi oferecida a uma arma")
	}
}

func sheetNameItem(t *testing.T, f sceneFixture, id int64, name string) sqlcgen.ListItemsByCharacterRow {
	t.Helper()
	items, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar os itens: %v", err)
	}
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("o item %q não está na ficha", name)
	return sqlcgen.ListItemsByCharacterRow{}
}

// ADICIONAR DO CATÁLOGO usa o nome e os espaços DO LIVRO.
//
// Deixar o cliente mandá-los abriria a porta para uma "Espada longa" de zero
// espaços, que é carga de graça na mochila.
func TestAddingFromTheCatalogUsesTheBookNumbers(t *testing.T) {
	f, id := fighterFixture(t)

	if refusal := bagCommand(t, f, id, "itens/adiciona/espada-longa"); refusal != "" {
		t.Fatalf("adicionar foi recusado: %q", refusal)
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

	if refusal := customItem(t, f, id, `{"item_name":"  ","item_qty":1,"item_slots":1}`); refusal == "" {
		t.Error("um item sem nome foi aceito")
	}
	if refusal := customItem(t, f, id, `{"item_name":"Pena","item_qty":1,"item_slots":0.3}`); refusal == "" {
		t.Error("0,3 espaço foi aceito, e o livro conta de meio em meio")
	}
	if refusal := customItem(t, f, id, `{"item_name":"Pena","item_qty":1,"item_slots":0.5}`); refusal != "" {
		t.Fatalf("meio espaço foi recusado: %q", refusal)
	}
	item := sheetNameItem(t, f, id, "Pena")
	if item.Slots != 0.5 || item.Catalogid.Valid {
		t.Errorf("o item custom entrou errado: espaços %v, catálogo %q", item.Slots, item.Catalogid.String)
	}
}

func customItem(t *testing.T, f sceneFixture, id int64, body string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/itens/custom?tab=bag", id)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, body).Body.String())
}

// EDITAR muda os três campos, e REMOVER tira da ficha.
func TestEditingAndRemovingAnItem(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, "", "Lembrança", "")

	target := fmt.Sprintf("/personagens/%d/itens/%d/edita?tab=bag", id, item)
	body := `{"item_name":"Lembrança da Ana","item_qty":3,"item_slots":0.5}`
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, body).Body.String()); refusal != "" {
		t.Fatalf("editar foi recusado: %q", refusal)
	}
	edited := sheetNameItem(t, f, id, "Lembrança da Ana")
	if edited.Quantity != 3 || edited.Slots != 0.5 {
		t.Errorf("a edição gravou %d × %v", edited.Quantity, edited.Slots)
	}

	if refusal := bagCommand(t, f, id, fmt.Sprintf("itens/%d/remover", item)); refusal != "" {
		t.Fatalf("remover foi recusado: %q", refusal)
	}
	items, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("o item sobreviveu ao remover: %d na ficha", len(items))
	}
}

// USAR gasta a dose e aplica o que a MESA rolou, preso no máximo.
func TestUsingSpendsTheDoseAndAppliesTheTableRoll(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Ferido", "Guerreiro", 3, 20, 0)
	item := itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	if refusal := use(t, f, id, item, `{"item_roll_hp":7}`); refusal != "" {
		t.Fatalf("usar foi recusado: %q", refusal)
	}
	if pool := poolsOf(t, f.s, id); pool.HpCurrent != 17 {
		t.Errorf("o PV ficou %d, quer 17 (10 + os 7 que a mesa rolou)", pool.HpCurrent)
	}
	// A DOSE FOI GASTA: era uma só, então a linha sai da ficha.
	items, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(items) != 0 {
		t.Error("a dose foi usada e o item continua na mochila")
	}
}

// A CURA NÃO PASSA DO MÁXIMO, e é o motor que prende.
func TestUsingDoesNotGoPastMaximumHp(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Quase cheio", "Guerreiro", 3, 2, 0)
	item := itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	if refusal := use(t, f, id, item, `{"item_roll_hp":8}`); refusal != "" {
		t.Fatalf("usar foi recusado: %q", refusal)
	}
	if pool := poolsOf(t, f.s, id); pool.HpCurrent != 30 {
		t.Errorf("o PV ficou %d, quer 30 — a cura passou do máximo", pool.HpCurrent)
	}
}

// O QUE NÃO É CONSUMÍVEL não se usa.
func TestWhatIsNotConsumableCannotBeUsed(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")

	if refusal := use(t, f, id, item, "{}"); !strings.Contains(refusal, "consumível") {
		t.Errorf("a recusa não diz o motivo: %q", refusal)
	}
	if _, err := f.s.sceneCore().Queries().GetItem(context.Background(), item); err != nil {
		t.Error("a espada foi consumida assim mesmo")
	}
}

func use(t *testing.T, f sceneFixture, id, item int64, body string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/itens/%d/usa?tab=bag", id, item)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, body).Body.String())
}

// A MELHORIA QUE NÃO CABE É RECUSADA PELO SERVIDOR.
//
// Filtro de tela não recusa nada: com a compatibilidade só no diálogo, um
// pedido montado à mão põe corda de arco num escudo e o servidor grava.
func TestAnImprovementThatDoesNotFitIsRefusedByTheServer(t *testing.T) {
	f, id := fighterFixture(t)
	shield := itemSemeia(t, f, id, "escudo-leve", "Escudo leve", "")

	// A Certeira é de ARMA (`appliesTo: ["weapon"]`).
	refusal := improvements(t, f, id, shield, `{"item_improvements":["melhoria-certeira"]}`)
	if !strings.Contains(refusal, "Certeira") || !strings.Contains(refusal, "Escudo leve") {
		t.Errorf("a recusa não nomeia os dois lados: %q", refusal)
	}
	if saved := itemImprovements(t, f, shield); saved != "[]" {
		t.Errorf("a recusa gravou assim mesmo: %q", saved)
	}

	// E a que CABE passa: o guarda não pode estar recusando tudo.
	if refused := improvements(t, f, id, shield, `{"item_improvements":["melhoria-reforcada"]}`); refused != "" {
		t.Fatalf("uma melhoria de escudo foi recusada: %q", refused)
	}
	if saved := itemImprovements(t, f, shield); !strings.Contains(saved, "melhoria-reforcada") {
		t.Errorf("a melhoria que cabe não foi gravada: %q", saved)
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
	balm := itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	refusal := improvements(t, f, id, balm, `{"item_material":"material-aco-rubi"}`)
	if refusal == "" {
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
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")

	if refusal := improvements(t, f, id, sword, `{"item_improvements":["melhoria-lendaria"]}`); refusal == "" {
		t.Error("uma melhoria que não existe foi aceita")
	}
	if saved := itemImprovements(t, f, sword); saved != "[]" {
		t.Errorf("a recusa gravou assim mesmo: %q", saved)
	}
}

func improvements(t *testing.T, f sceneFixture, id, item int64, body string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/itens/%d/melhorias?tab=bag", id, item)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, body).Body.String())
}

func itemImprovements(t *testing.T, f sceneFixture, item int64) string {
	t.Helper()
	row, err := f.s.sceneCore().Queries().GetItem(context.Background(), item)
	if err != nil {
		t.Fatalf("ler o item: %v", err)
	}
	return row.Improvements
}

// OS TRÊS ENCANTOS DA ARMA, e as duas regras que só eles têm (ALE-416).
//
// Batem no `EnchantItem` e não numa rota, porque não HÁ rota: quem encanta é o
// mestre, e a tela dele é outra fatia. O que este caso prende é a regra viajando
// com a ESCRITA — quando a tela do mestre chegar, ela herda a recusa em vez de
// reescrevê-la.
//
// O teto é do livro: "um item mágico menor possui um encanto, um médio possui
// dois e um item mágico maior possui três encantos" (p334), e a mesma página
// chama três de "o máximo possível". Ele conta por PESO e não por linha: três
// encantos da Tabela 8-8 contam como dois, e contar linhas deixaria entrar uma
// espada com três Magníficas — que pela contagem do livro é uma arma de seis.
func TestAWeaponTakesNoMoreThanThreeEnchants(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	encanta := func(ids ...string) string {
		err := f.s.characterPlays().EnchantItem(context.Background(), sword, ids)
		if err == nil {
			return ""
		}
		return err.Error()
	}

	// TRÊS DE PESO UM CABEM — este é o controle, e sem ele "nada foi gravado"
	// se explicaria igualmente bem por "a recusa recusa tudo".
	if refusal := encanta("encanto-flamejante", "encanto-congelante", "encanto-eletrica"); refusal != "" {
		t.Fatalf("o controle já estava errado: três encantos de peso um foram recusados: %q", refusal)
	}
	if saved := itemEnchants(t, f, sword); len(saved) != 3 {
		t.Fatalf("os três encantos que cabem viraram %v", saved)
	}

	// O QUARTO NÃO.
	quatro := encanta("encanto-flamejante", "encanto-congelante", "encanto-eletrica", "encanto-tumular")
	if !strings.Contains(quatro, "334") {
		t.Errorf("o quarto encanto entrou, ou a recusa não cita a página: %q", quatro)
	}

	// E DOIS QUE CONTAM COMO DOIS JÁ ESTOURAM, apesar de serem só duas linhas.
	// Sem o peso este pedido passaria: `len` é 2, e o teto é 3.
	pesados := encanta("encanto-dilacerante", "encanto-lancinante", "encanto-formidavel", "encanto-magnifica")
	if !strings.Contains(pesados, "peso") {
		t.Errorf("duas linhas de peso dois entraram numa arma de três encantos: %q", pesados)
	}
	// E a arma continua com os três primeiros — recusa não grava pela metade.
	if saved := itemEnchants(t, f, sword); len(saved) != 3 {
		t.Errorf("a recusa mexeu no que já estava gravado: %v", saved)
	}
}

// O PRÉ-REQUISITO SE CONFERE CONTRA O CONJUNTO ESCOLHIDO.
//
// "Pré-requisito: formidável" (p336) para a Magnífica e a Energética,
// "Pré-requisito: dilacerante" (p335) para a Lancinante. Quem grava manda o
// conjunto INTEIRO, então conferir contra o que a arma já tinha deixaria tirar o
// pré-requisito e manter o dependente no mesmo pedido.
func TestAnEnchantWithoutItsPrerequisiteIsRefused(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	encanta := func(ids ...string) string {
		err := f.s.characterPlays().EnchantItem(context.Background(), sword, ids)
		if err == nil {
			return ""
		}
		return err.Error()
	}

	refusal := encanta("encanto-magnifica")
	if !strings.Contains(refusal, "Magnífica") || !strings.Contains(refusal, "Formidável") {
		t.Errorf("a recusa não nomeia os dois lados: %q", refusal)
	}
	if saved := itemEnchants(t, f, sword); len(saved) != 0 {
		t.Errorf("a Magnífica entrou sem a Formidável: %v", saved)
	}

	// COM A FORMIDÁVEL JUNTA, no MESMO pedido, ela entra.
	if refused := encanta("encanto-formidavel", "encanto-magnifica"); refused != "" {
		t.Fatalf("o par legítimo foi recusado: %q", refused)
	}
	if saved := itemEnchants(t, f, sword); len(saved) != 2 {
		t.Errorf("o par legítimo não foi gravado: %v", saved)
	}

	// E TIRAR A FORMIDÁVEL NO MESMO PEDIDO não deixa a Magnífica órfã: é aqui
	// que conferir contra o ESTADO ANTERIOR deixaria passar.
	if refused := encanta("encanto-magnifica"); refused == "" {
		t.Error("a Formidável saiu e a Magnífica ficou sozinha na arma")
	}
}

// E SÓ ARMA RECEBE ENCANTO: o livro põe o encanto no lugar da melhoria de um
// item superior, e itens superiores são "armas, armaduras e escudos, ferramentas,
// vestuário e esotéricos" (p164) — mas os 28 da Tabela 8-8 declaram todos
// `appliesTo: ["weapon"]`.
func TestOnlyAWeaponTakesAnEnchant(t *testing.T) {
	f, id := fighterFixture(t)
	shield := itemSemeia(t, f, id, "escudo-leve", "Escudo leve", "")

	err := f.s.characterPlays().EnchantItem(context.Background(), shield, []string{"encanto-flamejante"})
	if err == nil || !strings.Contains(err.Error(), "arma") {
		t.Errorf("um escudo foi encantado, ou a recusa não diz por quê: %v", err)
	}
	if saved := itemEnchants(t, f, shield); len(saved) != 0 {
		t.Errorf("o escudo ficou com %v", saved)
	}
}

// O JOGADOR NÃO ENCANTA, E A FRONTEIRA É A AUSÊNCIA DO SINAL.
//
// O diálogo de melhorias é do jogador e grava melhoria e material. Um pedido
// montado à mão com `item_enchants` não tem por onde entrar — o campo não existe
// nos `Signals`, e o JSON que não casa é descartado em silêncio pelo
// `encoding/json`.
//
// Este caso existe para o dia em que alguém RECOLOCAR o campo achando que
// completa a simetria com a melhoria. Ele falha dizendo o que a simetria custa.
func TestThePlayerCannotEnchantThroughTheImprovementsDialog(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")

	// O CONTROLE: o mesmo pedido COM uma melhoria é aceito, então "nada foi
	// gravado" não se explica por a rota estar quebrada.
	body := `{"item_improvements":["melhoria-certeira"],"item_enchants":["encanto-flamejante"]}`
	if refusal := improvements(t, f, id, sword, body); refusal != "" {
		t.Fatalf("o controle já estava errado: a melhoria foi recusada: %q", refusal)
	}
	if saved := itemImprovements(t, f, sword); !strings.Contains(saved, "melhoria-certeira") {
		t.Fatalf("o controle já estava errado: a melhoria não foi gravada: %q", saved)
	}
	if saved := itemEnchants(t, f, sword); len(saved) != 0 {
		t.Errorf("o jogador encantou a arma pelo diálogo de melhorias: %v", saved)
	}
}

// E A CARTA DA MÃO NOMEIA O ENCANTO, junto das melhorias (ALE-416).
//
// A carta é o que o jogador OLHA em combate — a ficha do item é dois toques
// depois. Uma espada Flamejante cujo cartão diz só "Certeira · Mitral" mente
// por omissão no lugar mais caro de mentir.
//
// Foi o que o olho pegou: o dado estava certo, a ficha do item já mostrava os
// dois encantos, e o cartão não. Nenhum limiar responde isso.
func TestTheHandCardNamesTheEnchantsOfTheWeapon(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().SaveItemOverlays(
		context.Background(), sword, []string{"melhoria-certeira"}, "",
	); err != nil {
		t.Fatalf("melhorar a espada: %v", err)
	}
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), sword, []string{"encanto-flamejante"},
	); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}

	// O CARTÃO É O DA MÃO, e o recorte vai até o cartão vizinho: a ficha do
	// item desenha os mesmos nomes mais abaixo na tela, e procurar na tela
	// inteira passaria verde sobre um cartão vazio.
	tela := bagScreen(t, f, id)
	inicio := strings.Index(tela, "MÃO PRINCIPAL")
	if inicio < 0 {
		inicio = strings.Index(tela, "Mão principal")
	}
	if inicio < 0 {
		t.Fatal("a tira das mãos não saiu na tela")
	}
	cartao := tela[inicio:]
	if fim := strings.Index(cartao, "Mão secundária"); fim > 0 {
		cartao = cartao[:fim]
	}
	// O CONTROLE é a Certeira: mesma forma, mesmo cartão, e ela já funcionava.
	if !strings.Contains(cartao, "Certeira") {
		t.Fatalf("o controle já estava errado: o cartão não traz a melhoria:\n%s", cartao)
	}
	if !strings.Contains(cartao, "Flamejante") {
		t.Errorf("o cartão da mão não diz que a espada é Flamejante:\n%s", cartao)
	}
}

func itemEnchants(t *testing.T, f sceneFixture, item int64) []string {
	t.Helper()
	ids, err := f.s.sceneCore().Queries().EnchantsOfItem(context.Background(), item)
	if err != nil {
		t.Fatalf("ler os encantos do item: %v", err)
	}
	return ids
}

// O JOGADOR VÊ OS ENCANTOS DA ARMA DELE, na ficha do item (ALE-416).
//
// Ver é a metade que sobra para o jogador depois de a escrita ir para o mestre,
// e sem ela a arma encantada só existiria no número da carta: +2 no ataque sem
// nada na tela dizendo de onde ele vem. A procedência de um termo é o que
// impede a ficha de virar um punhado de números.
//
// O bloco é irmão do de melhorias e NÃO tem botão — é essa ausência que diz que
// o jogador não mexe. O teste afirma as duas coisas.
func TestThePlayerSeesTheEnchantsOfTheirWeapon(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), sword, []string{"encanto-flamejante", "encanto-formidavel"},
	); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}

	ficha := itemScreenSheet(bagScreen(t, f, id), "Espada longa")
	if ficha == "" {
		t.Fatal("a ficha da espada longa não saiu na tela")
	}
	for _, escrito := range []string{"Encantos", "Flamejante", "Formidável", "+1d6 de dano de fogo"} {
		if !strings.Contains(ficha, escrito) {
			t.Errorf("a ficha da espada não escreve %q:\n%s", escrito, ficha)
		}
	}
	// E O QUE ELA NÃO PODE TER: um interruptor. O diálogo de melhorias usa
	// `role="switch"` em cada linha; se o bloco do encanto ganhar um, o jogador
	// passou a escolher.
	bloco := ficha[strings.Index(ficha, "Encantos"):]
	if fim := strings.Index(bloco, "</div>"); fim > 0 {
		bloco = bloco[:fim]
	}
	if strings.Contains(bloco, `role="switch"`) || strings.Contains(bloco, "<button") {
		t.Errorf("o bloco de encantos oferece um gesto, e o jogador não encanta:\n%s", bloco)
	}
}
