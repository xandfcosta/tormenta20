package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
	"t20engine/sheet"
	"testing"
)

func seedPericia(t *testing.T, s *Server, id int64, nome, atributo string, treinada bool) {
	t.Helper()
	treino := int64(0)
	if treinada {
		treino = 1
	}
	_, err := s.Queries().CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: id, Name: nome, Attribute: atributo, Trained: treino, Custom: 0,
	})
	if err != nil {
		t.Fatalf("semear a perícia %q: %v", nome, err)
	}
}

// seedEfeitoCondicional põe um efeito de cena que soma no ataque só quando o
// jogador o liga.
func seedEfeitoCondicional(t *testing.T, s *Server, id int64, quanto int) {
	t.Helper()
	mods := fmt.Sprintf(
		`[{"target":{"k":"attack","scope":"all"},"amount":%d,"bonusType":"untyped",`+
			`"condition":{"c":"context","note":"enquanto estiver em Fúria"}}]`, quanto)
	_, err := s.Queries().CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: id, Catalogid: "furia", Scope: "scene",
		Modifiers: mods, Createdat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o efeito condicional: %v", err)
	}
}
func fighterFixture(t *testing.T) (pilotoFixture, int64) {
	t.Helper()
	f := novoPiloto(t)
	id, err := f.s.Queries().CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
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

func combatScreen(t *testing.T, f pilotoFixture, id int64) string {
	t.Helper()
	return f.pede(t, f.jogador, http.MethodGet,
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
func TestTheCombatPanelSaysTheEngineNumbers(t *testing.T) {
	f, id := fighterFixture(t)
	tela := combatScreen(t, f, id)

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

// OS CONDICIONAIS LIGADOS ENTRAM NA CONTA, e esta é a garantia nova da fatia.
//
// Toda cena do piloto até aqui computou a ficha BASE (`sheetFromDTO`, com
// `map[string]bool{}`). Se o Combate fizesse o mesmo, um bárbaro em Fúria veria
// o ataque de quem não está em Fúria — e a ficha discordaria da Mesa, que já lê
// o estado ligado. O defeito não teria sintoma nenhum numa ficha sem
// condicional, que é a maioria delas.
//
// O efeito abaixo soma +3 em TODO ataque, e só quando ligado.
func TestActiveConditionalsEnterTheAttack(t *testing.T) {
	f, id := fighterFixture(t)
	seedEfeitoCondicional(t, f.s, id, 3)

	// DESLIGADO: o ataque é o mesmo de sempre — ½ nível 1 + Força 4 + treino 2.
	if tela := combatScreen(t, f, id); !strings.Contains(tela, `aria-label="Atq CaC +7"`) {
		t.Fatal("com o condicional desligado o ataque não é +7: o painel mudou de base e o resto do caso não mede nada")
	}

	ligaOCondicional(t, f, id)

	// LIGADO: +3, e o número é escrito à mão — 7 + 3.
	tela := combatScreen(t, f, id)
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
	row, err := f.s.Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	dto, err := f.s.LoadCharacter(context.Background(), row)
	if err != nil {
		t.Fatalf("carregar o personagem: %v", err)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		t.Fatalf("converter para o motor: %v", err)
	}
	oferecidos := engine.ComputeItemEffects(f.s.Catalogs().ActiveItemsFor(ec)).Conditional
	if len(oferecidos) != 1 {
		t.Fatalf("o motor ofereceu %d condicionais e o caso precisa de exatamente 1: "+
			"o efeito semeado não virou um opt-in, e ligar nada mediria o vazio", len(oferecidos))
	}
	err = f.s.Queries().AddCharacterConditional(context.Background(), sqlcgen.AddCharacterConditionalParams{
		Characterid: id, Conditionalid: engine.ConditionalID(oferecidos[0]),
	})
	if err != nil {
		t.Fatalf("ligar o condicional: %v", err)
	}
}
