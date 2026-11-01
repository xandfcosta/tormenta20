package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"testing"
)

func seedPericia(t *testing.T, s *Server, id int64, name, attribute string, trained bool) {
	t.Helper()
	training := int64(0)
	if trained {
		training = 1
	}
	_, err := s.sceneCore().Queries().CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: id, Name: name, Attribute: attribute, Trained: training, Custom: 0,
	})
	if err != nil {
		t.Fatalf("semear a perícia %q: %v", name, err)
	}
}

// seedEfeitoCondicional põe um efeito de cena que soma no ataque só quando o
// jogador o liga.
func seedEfeitoCondicional(t *testing.T, s *Server, id int64, howMuch int) {
	t.Helper()
	mods := fmt.Sprintf(
		`[{"target":{"k":"attack","scope":"all"},"amount":%d,"bonusType":"untyped",`+
			`"condition":{"c":"context","note":"enquanto estiver em Fúria"}}]`, howMuch)
	_, err := s.sceneCore().Queries().CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: id, Catalogid: "furia", Scope: "scene",
		Modifiers: mods, Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o efeito condicional: %v", err)
	}
}
func fighterFixture(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id, err := f.s.sceneCore().Queries().CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.player, Name: "Combatente", Origin: "soldado", Level: 3,
		Strength: 4, Dexterity: 2, Constitution: 3, Intelligence: 0, Wisdom: 1, Charisma: 0,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassChoices: "{}", CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o combatente: %v", err)
	}
	seedClasse(t, f.s, id, "Guerreiro", 3)
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent, pools.MpCurrent = pools.HpMax, pools.MpMax
		return pools, nil
	})
	seedPericia(t, f.s, id, "Luta", "strength", true)
	seedPericia(t, f.s, id, "Pontaria", "dexterity", false)
	seedPericia(t, f.s, id, "Fortitude", "constitution", false)
	seedPericia(t, f.s, id, "Reflexos", "dexterity", false)
	seedPericia(t, f.s, id, "Vontade", "wisdom", false)
	seedMember(t, f.s, f.campaignID, id)
	return f, id
}

func combatScreen(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	return f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=combat", id), "").Body.String()
}

// O PAINEL CHEGA NA TELA com os números do motor.
//
// Os esperados são escritos À MÃO a partir do que o livro compõe, e não lidos de
// uma segunda chamada ao motor: derivar o esperado da produção faria o teste
// concordar com o defeito. Para o combatente de nível 3 acima:
//
//	Defesa    = 10 + Destreza 2                = 12
//	Atq CaC   = ½ nível 1 + Força 4 + treino 2 = +7
//	Atq Dist  = ½ nível 1 + Destreza 2         = +3
//	Fortitude = ½ nível 1 + Constituição 3     = +4
//	Reflexos  = ½ nível 1 + Destreza 2         = +3
//	Vontade   = ½ nível 1 + Sabedoria 1        = +2
//
// As resistências moram no topo das Perícias e não no Combate (decisão do
// dono, ALE-473), e é de lá que elas são lidas.
func TestTheCombatPanelSaysTheEngineNumbers(t *testing.T) {
	f, id := fighterFixture(t)
	screen := combatScreen(t, f, id)

	// O número da linha FECHADA, achada pelo título: procurar só o número
	// acharia o "12" de qualquer outro lugar da página.
	for name, want := range map[string]string{
		"Defesa": "12", "Ataque corpo a corpo": "+7", "Ataque à distância": "+3",
	} {
		if got := expertiseRowTotal(t, screen, name); got != want {
			t.Errorf("a linha %q do Combate diz %s, e o livro compõe %s", name, got, want)
		}
	}
	expertises := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=expertises", id), "").Body.String()
	for name, want := range map[string]string{"Fortitude": "+4", "Reflexos": "+3", "Vontade": "+2"} {
		if got := expertiseRowTotal(t, expertises, name); got != want {
			t.Errorf("a resistência %q diz %s, e o livro compõe %s", name, got, want)
		}
	}
	// Os SEIS atributos, e o "+0" é tão informativo quanto os outros: uma caixa
	// que some é uma pergunta sem resposta. Chave e valor no MESMO par, ou o
	// "+4" da Força casaria com o de qualquer outra caixa.
	for key, want := range map[string]string{
		"FOR": "+4", "DES": "+2", "CON": "+3", "INT": "+0", "SAB": "+1", "CAR": "+0",
	} {
		pair := `<span class="ds-stat-key">` + key + `</span> <span class="ds-stat-value">` + want + `</span>`
		if !strings.Contains(screen, pair) {
			t.Errorf("a tela não tem o atributo %s %s", key, want)
		}
	}
}

// OS CONDICIONAIS LIGADOS ENTRAM NA CONTA.
//
// Computar a ficha BASE (`sheet.Compute` com `map[string]bool{}`) faz um bárbaro
// em Fúria ver o ataque de quem não está em Fúria — e a ficha discorda da Mesa,
// que lê o estado ligado. O defeito não tem sintoma nenhum numa ficha SEM
// condicional, que é a maioria delas.
//
// O efeito abaixo soma +3 em TODO ataque, e só quando ligado.
func TestActiveConditionalsEnterTheAttack(t *testing.T) {
	f, id := fighterFixture(t)
	seedEfeitoCondicional(t, f.s, id, 3)

	// DESLIGADO: o ataque é o mesmo de sempre — ½ nível 1 + Força 4 + treino 2.
	if got := expertiseRowTotal(t, combatScreen(t, f, id), "Ataque corpo a corpo"); got != "+7" {
		t.Fatalf("com o condicional desligado o ataque é %s e não +7: o painel mudou de base e o resto do caso não mede nada", got)
	}

	ligaOCondicional(t, f, id)

	// LIGADO: +3, e o número é escrito à mão — 7 + 3.
	melee := sumLine(t, combatScreen(t, f, id), "Ataque corpo a corpo")
	if !strings.Contains(melee, `<span class="ds-row-value">+10</span>`) {
		t.Errorf("com o condicional LIGADO o ataque continua sem os +3: o painel computa a ficha base "+
			"e mostra o número de quem não está com o efeito ativo:\n%s", melee)
	}
	// A DECOMPOSIÇÃO tem de contar a mesma história que a linha: um total que
	// sobe sem parcela que o explique é pior que o total errado, porque parece
	// certo. Na MESMA linha, e não em qualquer lugar da página.
	if !strings.Contains(melee, "(cond.)") {
		t.Errorf("a conta do ataque não mostra a parcela do condicional que somou os +3:\n%s", melee)
	}
}

// ligaOCondicional marca o opt-in do jogador.
//
// A identidade do condicional é PERGUNTADA ao motor em vez de remontada aqui:
// ela é um encadeado de fonte, alvo, nota, valor e tipo, e reescrevê-lo à mão
// faria este teste falhar no dia em que a chave mudasse de forma — dizendo
// "o condicional não entrou na conta" quando o que mudou foi o formato do id.
// Perguntar é o que a própria aba de Efeitos faz para desenhar a lista.
//
// O CONTROLE está aqui: o motor tem de oferecer EXATAMENTE um condicional. Sem
// ele, um efeito semeado errado não ofereceria nenhum, o teste ligaria nada, e o
// ataque continuaria +7 — falhando com a mensagem de que o painel ignora
// condicionais, que é a conclusão errada.
func ligaOCondicional(t *testing.T, f sceneFixture, id int64) {
	t.Helper()
	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	dto, err := f.s.sheetRules().LoadCharacter(context.Background(), row)
	if err != nil {
		t.Fatalf("carregar o personagem: %v", err)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		t.Fatalf("converter para o motor: %v", err)
	}
	offered := engine.ComputeItemEffects(dto.Ruleset.ActiveItemsFor(ec)).Conditional
	if len(offered) != 1 {
		t.Fatalf("o motor ofereceu %d condicionais e o caso precisa de exatamente 1: "+
			"o efeito semeado não virou um opt-in, e ligar nada mediria o vazio", len(offered))
	}
	err = f.s.sceneCore().Queries().AddCharacterConditional(context.Background(), sqlcgen.AddCharacterConditionalParams{
		Characterid: id, Conditionalid: offered[0].Term,
	})
	if err != nil {
		t.Fatalf("ligar o condicional: %v", err)
	}
}

// A REDUÇÃO POR TIPO CHEGA NA TELA, e com o tipo escrito (ALE-410, p229).
//
//	"Armadura Óssea. Você recebe redução de corte, frio e perfuração 5." (p20)
//
// O osteon é a prova de que a RD por tipo não é a geral de outro jeito: ele NÃO
// tem RD geral nenhuma, e antes desta fatia a caixa da Defesa não dizia nada —
// a habilidade existia só como texto na aba de Poderes.
//
// A BARRA não aparece na linha curta de propósito: "RD 10/mágico" é a notação
// do livro para a redução que NÃO vale contra aqueles tipos (p229), o oposto
// desta. Escrever "RD 5/3 tipos" diria ao leitor de T20 exatamente o contrário.
func TestTheTypedDamageReductionReachesTheDefenseTile(t *testing.T) {
	f, id := fighterFixture(t)
	seedRaca(t, f.s, id, "osteon")

	screen := combatScreen(t, f, id)
	if !strings.Contains(screen, "RD por tipo") {
		t.Errorf("a caixa da Defesa não anuncia a redução por tipo do osteon:\n%s",
			recorte(screen))
	}
	for _, tipo := range []string{"corte", "frio", "perfuração"} {
		if !strings.Contains(screen, "Raça: Osteon · "+tipo) {
			t.Errorf("a decomposição não diz que a redução de %s vem da raça — "+
				"um \"+5\" sem o tipo não responde contra o quê ele vale", tipo)
		}
	}
	// E O CONTROLE: o guerreiro sem raça nenhuma não ganha promessa.
	f2, id2 := fighterFixture(t)
	if screen2 := combatScreen(t, f2, id2); strings.Contains(screen2, "RD por tipo") {
		t.Error("o guerreiro sem raça de redução lê uma RD por tipo que ele não tem")
	}
}

// A RD DO MATERIAL RAMIFICA PELA PEÇA, e a caixa da Defesa diz o número certo
// (p166, ALE-438).
//
// "Fornece redução de dano: armaduras leves e escudos, RD 2; armaduras pesadas,
// RD 5" — o adamante escreve dois números, e até aqui a ficha não mostrava
// nenhum: o verbete dizia, por escrito no `unmodeled`, que um modificador de
// sobreposição não sabia ramificar pelo hospedeiro.
//
// O caso mede as DUAS pontas no mesmo material, porque uma só passaria verde
// com um filtro que aceita qualquer peça.
func TestTheMaterialDamageReductionFollowsTheHostPiece(t *testing.T) {
	rdOf := func(t *testing.T, item, nome, slot string) string {
		t.Helper()
		f, id := fighterFixture(t)
		linha := itemSemeia(t, f, id, item, nome, slot)
		if err := f.s.characterPlays().SaveItemOverlays(
			context.Background(), linha, []string{}, "material-adamante",
		); err != nil {
			t.Fatalf("forjar %s em adamante: %v", nome, err)
		}
		return combatScreen(t, f, id)
	}

	if tela := rdOf(t, "armadura-completa", "Armadura completa", "vested"); !strings.Contains(tela, "RD 5") {
		t.Errorf("a armadura pesada de adamante não anuncia RD 5:\n%s", recorte(tela))
	}
	if tela := rdOf(t, "escudo-pesado", "Escudo pesado", "wielded"); !strings.Contains(tela, "RD 2") {
		t.Errorf("o escudo de adamante não anuncia RD 2:\n%s", recorte(tela))
	}
	// E O CONTROLE: sem o material, a mesma armadura não promete RD nenhuma.
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "armadura-completa", "Armadura completa", "vested")
	if tela := combatScreen(t, f, id); strings.Contains(tela, "RD ") {
		t.Errorf("a armadura comum anuncia uma RD que ela não tem:\n%s", recorte(tela))
	}
}

// A CHANCE DE IGNORAR O EXTRA DO CRÍTICO CHEGA À CAIXA DA DEFESA (p338, ALE-436).
//
// O app INFORMA e a mesa rola, então quem tem a chance precisa VER que ela
// existe: um benefício que só mora dentro de um diálogo é um benefício que
// ninguém lembra de usar na hora.
func TestTheChanceToIgnoreTheCriticalExtraReachesTheDefenseTile(t *testing.T) {
	f, id := fighterFixture(t)
	armadura := itemSemeia(t, f, id, "armadura-completa", "Armadura completa", "vested")
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), armadura, []string{"encanto-fortificado"},
	); err != nil {
		t.Fatalf("encantar a armadura: %v", err)
	}

	screen := combatScreen(t, f, id)
	// NA FACE DA CAIXA, ao lado do que a RD já dizia.
	if !strings.Contains(screen, "50% vs crítico") {
		t.Errorf("a caixa da Defesa não anuncia a chance do Fortificado:\n%s", recorte(screen))
	}
	// E NA DECOMPOSIÇÃO, com a fonte e o que ela faz.
	if !strings.Contains(screen, "ignora o extra de crítico") {
		t.Error("a decomposição não diz o que a chance alcança")
	}
	// O CONTROLE: a mesma armadura sem encanto não promete chance nenhuma.
	f2, id2 := fighterFixture(t)
	itemSemeia(t, f2, id2, "armadura-completa", "Armadura completa", "vested")
	if screen2 := combatScreen(t, f2, id2); strings.Contains(screen2, "50% vs crítico") {
		t.Error("a armadura comum anuncia uma chance que ela não tem")
	}
}

// E A CHANCE SEGUE A PEÇA: o mesmo encanto num ESCUDO dá 25% (p338).
func TestTheFortifiedChanceFollowsThePieceItIsOn(t *testing.T) {
	f, id := fighterFixture(t)
	escudo := itemSemeia(t, f, id, "escudo-pesado", "Escudo pesado", "wielded")
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), escudo, []string{"encanto-fortificado"},
	); err != nil {
		t.Fatalf("encantar o escudo: %v", err)
	}
	if screen := combatScreen(t, f, id); !strings.Contains(screen, "25% vs crítico") {
		t.Errorf("o escudo Fortificado não anuncia os 25%% da p338:\n%s", recorte(screen))
	}
}
