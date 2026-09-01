package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
)

// Os guardas do painel de COMBATE (ALE-272, fatia 3).
//
// O que eles prendem é a REGRA e a DECISÃO: que os condicionais ligados entram
// na conta, que as linhas de um diálogo somam o total que ele mostra, e quem vê
// cada bloco. Os NÚMEROS em si são do motor e já têm o oráculo de paridade —
// repetir aqui a ficha inteira seria a mesma tabela escrita duas vezes.

// oCombatente é uma ficha com atributos e perícias de VERDADE.
//
// O `seedCharacterAtLevel` deixa todo atributo em zero, e sobre zeros um painel
// de combate inteiro fica indistinguível de um painel vazio: "+0" em toda caixa
// passaria igual se a conta não acontecesse. Os valores abaixo são escolhidos
// para que cada número da tela seja DIFERENTE dos outros.
//
// Nível 3, e daí saem à mão: ½ nível = 1, e o treino de uma perícia treinada
// vale +2 até o 6º (`trainingBonusForLevel`).
func oCombatente(t *testing.T) (pilotoFixture, int64) {
	t.Helper()
	f := novoPiloto(t)
	id, err := f.s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.jogador, Name: "Combatente", Origin: "Soldado", Level: 3,
		HpMax: 30, HpCurrent: 30, MpMax: 0, MpCurrent: 0,
		Strength: 4, Dexterity: 2, Constitution: 3, Intelligence: 0, Wisdom: 1, Charisma: 0,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: plataforma.NowISO(), UpdatedAt: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o combatente: %v", err)
	}
	seedClasse(t, f.s, id, "Guerreiro", 3)
	seedPericia(t, f.s, id, "Luta", "strength", true)
	seedPericia(t, f.s, id, "Pontaria", "dexterity", false)
	seedPericia(t, f.s, id, "Fortitude", "constitution", false)
	seedPericia(t, f.s, id, "Reflexos", "dexterity", false)
	seedPericia(t, f.s, id, "Vontade", "wisdom", false)
	return f, id
}

func seedPericia(t *testing.T, s *Server, id int64, nome, atributo string, treinada bool) {
	t.Helper()
	treino := int64(0)
	if treinada {
		treino = 1
	}
	_, err := s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: id, Name: nome, Attribute: atributo, Trained: treino, Custom: 0,
	})
	if err != nil {
		t.Fatalf("semear a perícia %q: %v", nome, err)
	}
}

func aTelaDoCombate(t *testing.T, f pilotoFixture, id int64) string {
	t.Helper()
	return f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/piloto/personagens/%d?tab=combat", id), "").Body.String()
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
func TestOPainelDeCombateDizOsNumerosDoMotor(t *testing.T) {
	f, id := oCombatente(t)
	tela := aTelaDoCombate(t, f, id)

	// Rótulo E valor no mesmo `aria-label`, que é como a caixa se nomeia: procurar
	// só o número acharia o "12" de qualquer outro lugar da página.
	for _, esperado := range []string{
		"Defesa 12", "Atq CaC +7", "Atq Dist +3", "Fort +4", "Refl +3", "Vont +2",
	} {
		if !strings.Contains(tela, `aria-label="`+esperado+`"`) {
			t.Errorf("a tela não tem a caixa %q", esperado)
		}
	}
	// Os SEIS atributos, e o "+0" é tão informativo quanto os outros: uma caixa
	// que some é uma pergunta sem resposta.
	for _, esperado := range []string{"FOR", "+4", "DES", "+2", "CON", "+3", "INT", "SAB", "CAR"} {
		if !strings.Contains(tela, ">"+esperado+"</p>") {
			t.Errorf("a tela não tem o atributo %q", esperado)
		}
	}
}

// AS LINHAS DE UM DIÁLOGO SOMAM O TOTAL QUE ELE MOSTRA.
//
// É a promessa que um diálogo de decomposição faz, e a mais fácil de quebrar sem
// perceber: o `Defense.Base` do motor é 10 + Destreza num campo só, enquanto a
// tela mostra o 10 numa linha e a Destreza na seguinte. Usar o campo cru
// contaria a Destreza DUAS vezes — a Defesa da caixa continuaria certa e só o
// diálogo mentiria, que é o defeito que ninguém vê até alguém somar.
//
// Os dois ramos importam: com Destreza aplicada e com ela bloqueada.
func TestAsLinhasDaDefesaSomamOTotal(t *testing.T) {
	casos := []struct {
		nome     string
		aplicada bool
	}{
		{"com a Destreza aplicada", true},
		{"com a Destreza bloqueada por armadura pesada", false},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			sheet := engine.ComputedSheetV2{
				Defense: engine.DefenseBreakdown{
					Base: 10, Total: 15, VsMelee: 15, VsRanged: 15, DexApplied: caso.aplicada,
					Contributions: []engine.BreakdownContribution{{Source: "Armadura", Amount: 5}},
				},
				Attributes: map[string]engine.AttributeBreakdown{"dexterity": {Total: 3}},
			}
			if caso.aplicada {
				// O motor embute a Destreza no `Base`; o total sobe junto.
				sheet.Defense.Base = 13
				sheet.Defense.Total = 18
				sheet.Defense.VsMelee, sheet.Defense.VsRanged = 18, 18
			}

			soma := 0
			for _, linha := range defenseRows(sheet) {
				soma += valorDaLinha(t, linha.Value)
			}
			if soma != sheet.Defense.Total {
				t.Errorf("as linhas somam %d e a caixa mostra %d: o diálogo mente sobre de onde vem a Defesa",
					soma, sheet.Defense.Total)
			}
		})
	}
}

// A DESTREZA BLOQUEADA APARECE ZERADA, e não some.
//
// Sumir seria a resposta errada para a pergunta que o diálogo existe para
// responder: quem veste armadura pesada quer ver POR QUE a Defesa não subiu com
// a Destreza dele, e uma linha ausente não diz nada.
func TestADestrezaBloqueadaSaiComoLinhaApagada(t *testing.T) {
	sheet := engine.ComputedSheetV2{
		Defense:    engine.DefenseBreakdown{Base: 10, Total: 10, VsMelee: 10, VsRanged: 10, DexApplied: false},
		Attributes: map[string]engine.AttributeBreakdown{"dexterity": {Total: 3}},
	}
	linhas := defenseRows(sheet)
	if len(linhas) < 2 {
		t.Fatalf("a Defesa saiu com %d linhas: nem a base e a Destreza estão lá", len(linhas))
	}
	dex := linhas[1]
	if !strings.Contains(dex.Label, "bloqueada por armadura pesada") {
		t.Errorf("a linha da Destreza é %q e não diz que ela está bloqueada", dex.Label)
	}
	if dex.Value != "+0" {
		t.Errorf("a Destreza bloqueada vale %q e devia valer +0: ela não entra na Defesa", dex.Value)
	}
	if !dex.Muted {
		t.Error("a linha da Destreza bloqueada não está apagada: ela se lê como um bônus que existe")
	}
}

// QUEM VÊ O BLOCO DE ARMA, e a regra não é "quem tem arma".
//
// É DECISÃO com três ramos, e o do meio é o que se perde num porte: o marcial de
// mãos livres vê o texto de vazio, para a caixa não parecer quebrada, enquanto o
// conjurador de mãos livres não vê o bloco — para ele o assunto é a tripla
// mágica, e um "nenhuma arma empunhada" seria ruído sobre o que ele nunca teve.
func TestOBlocoDeArmaSegueQuemEmpunhaEQuemConjura(t *testing.T) {
	umaArma := []engine.WeaponCard{{Name: "Machado", Skill: "Luta", Damage: "1d12", CritRange: 20, CritMult: 3}}
	casos := []struct {
		nome        string
		cards       []engine.WeaponCard
		caster      bool
		querBloco   bool
		querCartoes int
	}{
		{"o marcial que empunha vê o cartão", umaArma, false, true, 1},
		{"o marcial de mãos livres vê o texto de vazio", nil, false, true, 0},
		{"o conjurador que empunha vê o cartão", umaArma, true, true, 1},
		{"o conjurador de mãos livres não vê o bloco", nil, true, false, 0},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			painel := combatPanelFor(engine.ComputedSheetV2{}, caso.cards, caso.caster)
			if painel.ShowWeapons != caso.querBloco {
				t.Errorf("ShowWeapons = %v, quer %v", painel.ShowWeapons, caso.querBloco)
			}
			if len(painel.Weapons) != caso.querCartoes {
				t.Errorf("saíram %d cartões, quer %d", len(painel.Weapons), caso.querCartoes)
			}
		})
	}
}

// A TRIPLA MÁGICA SÓ SAI PARA QUEM CONJURA POR CLASSE.
func TestATriplaMagicaSoSaiParaQuemConjura(t *testing.T) {
	if tiles := combatPanelFor(engine.ComputedSheetV2{}, nil, false).MagicTiles; len(tiles) != 0 {
		t.Errorf("quem não conjura recebeu %d caixas mágicas", len(tiles))
	}
	tiles := combatPanelFor(engine.ComputedSheetV2{}, nil, true).MagicTiles
	if len(tiles) != 3 {
		t.Fatalf("o conjurador recebeu %d caixas mágicas, quer 3", len(tiles))
	}
	for _, tile := range tiles {
		if !tile.Magic {
			t.Errorf("a caixa %q saiu com a paleta de combate em vez da arcana", tile.Label)
		}
	}
}

// OS CONDICIONAIS LIGADOS ENTRAM NA CONTA, e esta é a garantia nova da fatia.
//
// Toda cena do piloto até aqui computou a ficha BASE (`sheetFromDTO`, com
// `map[string]bool{}`). Se o Combate fizesse o mesmo, um bárbaro em Fúria veria
// o ataque de quem não está em Fúria — e a ficha discordaria da Mesa, que já lê
// o estado ligado. O defeito não teria sintoma nenhum numa ficha sem
// condicional, que é a maioria delas.
//
// O efeito abaixo soma +3 em TODO ataque, e só quando ligado.
func TestOsCondicionaisLigadosEntramNoAtaque(t *testing.T) {
	f, id := oCombatente(t)
	seedEfeitoCondicional(t, f.s, id, 3)

	// DESLIGADO: o ataque é o mesmo de sempre — ½ nível 1 + Força 4 + treino 2.
	if tela := aTelaDoCombate(t, f, id); !strings.Contains(tela, `aria-label="Atq CaC +7"`) {
		t.Fatal("com o condicional desligado o ataque não é +7: o painel mudou de base e o resto do caso não mede nada")
	}

	ligaOCondicional(t, f, id)

	// LIGADO: +3, e o número é escrito à mão — 7 + 3.
	tela := aTelaDoCombate(t, f, id)
	if !strings.Contains(tela, `aria-label="Atq CaC +10"`) {
		t.Error("com o condicional LIGADO o ataque continua sem os +3: o painel computa a ficha base " +
			"e mostra o número de quem não está com o efeito ativo")
	}
	// A DECOMPOSIÇÃO tem de contar a mesma história que a caixa: um total que
	// sobe sem linha que o explique é pior que o total errado, porque parece
	// certo.
	if !strings.Contains(tela, "(cond.)") {
		t.Error("o diálogo do ataque não mostra a linha do condicional que somou os +3")
	}
}

// seedEfeitoCondicional põe um efeito de cena que soma no ataque só quando o
// jogador o liga.
func seedEfeitoCondicional(t *testing.T, s *Server, id int64, quanto int) {
	t.Helper()
	mods := fmt.Sprintf(
		`[{"target":{"k":"attack","scope":"all"},"amount":%d,"bonusType":"untyped",`+
			`"condition":{"c":"context","note":"enquanto estiver em Fúria"}}]`, quanto)
	_, err := s.queries.CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: id, Catalogid: "furia", Scope: "scene",
		Modifiers: mods, Createdat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o efeito condicional: %v", err)
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
func ligaOCondicional(t *testing.T, f pilotoFixture, id int64) {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	dto, err := f.s.loadCharacter(context.Background(), row)
	if err != nil {
		t.Fatalf("carregar o personagem: %v", err)
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		t.Fatalf("converter para o motor: %v", err)
	}
	oferecidos := engine.ComputeItemEffects(f.s.catalogs.ActiveItemsFor(ec)).Conditional
	if len(oferecidos) != 1 {
		t.Fatalf("o motor ofereceu %d condicionais e o caso precisa de exatamente 1: "+
			"o efeito semeado não virou um opt-in, e ligar nada mediria o vazio", len(oferecidos))
	}
	err = f.s.queries.AddCharacterConditional(context.Background(), sqlcgen.AddCharacterConditionalParams{
		Characterid: id, Conditionalid: engine.ConditionalID(oferecidos[0]),
	})
	if err != nil {
		t.Fatalf("ligar o condicional: %v", err)
	}
}

// valorDaLinha lê o "+5" de uma linha de volta para inteiro.
func valorDaLinha(t *testing.T, texto string) int {
	t.Helper()
	var n int
	if _, err := fmt.Sscanf(strings.Replace(texto, "+", "", 1), "%d", &n); err != nil {
		t.Fatalf("a linha tem o valor %q, que não é um número com sinal", texto)
	}
	return n
}
