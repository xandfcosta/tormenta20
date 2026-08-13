package engine

import (
	"path/filepath"
	"strings"
	"testing"
)

// Redução de dano do Bárbaro — livro p42.
//
//	"REDUÇÃO DE DANO. A partir do 5º nível […] Você recebe redução de dano 2
//	 (todo dano que sofre é reduzido em 2). A cada três níveis, sua RD aumenta
//	 em 2, até um máximo de RD 10 no 17º nível."
//
// A progressão vinha coberta só pelos oráculos de paridade. O que se fixa aqui é
// a regra do livro: os patamares, o passo de três níveis e o teto (ALE-105).
func TestBarbaroRdProgression(t *testing.T) {
	// Um caso por FRONTEIRA, não um por nível: o que quebra numa tabela de
	// patamares é o limite, não o meio da faixa.
	tests := []struct {
		level int
		want  int
		nota  string
	}{
		{1, 0, "antes do 5º não há RD"},
		{4, 0, "o nível 4 ainda é zero — a habilidade começa no 5º"},
		{5, 2, "a habilidade entra com RD 2"},
		{7, 2, "dentro da faixa, não muda"},
		{8, 4, "+2 três níveis depois"},
		{11, 6, ""},
		{14, 8, ""},
		{17, 10, "o teto do livro, no 17º"},
		{20, 10, "acima do 17º continua 10 — é máximo, não progressão"},
	}
	for _, tt := range tests {
		if got := barbaroRdForLevel(tt.level); got != tt.want {
			t.Errorf("nível %d: RD = %d, want %d %s", tt.level, got, tt.want, tt.nota)
		}
	}
}

// O passo é de TRÊS níveis, não de dois nem de quatro — a forma mais provável de
// alguém "corrigir" a tabela errado. Derivado da regra em vez de repetir a
// tabela acima: se o passo mudar, isto quebra mesmo que os patamares batam.
func TestBarbaroRdStepsEveryThreeLevels(t *testing.T) {
	for level := 5; level <= 14; level += 3 {
		antes, depois := barbaroRdForLevel(level), barbaroRdForLevel(level+3)
		if depois-antes != 2 {
			t.Errorf("do nível %d ao %d a RD foi de %d para %d, want +2", level, level+3, antes, depois)
		}
		// E não pode subir ANTES da hora: dois níveis depois ainda é o mesmo.
		if meio := barbaroRdForLevel(level + 2); meio != antes {
			t.Errorf("nível %d: RD = %d, want %d — subiu antes dos três níveis", level+2, meio, antes)
		}
	}
}

// "Especialização em Armadura" — Cavaleiro p54, Guerreiro p65: poder ESCOLHIDO
// com pré-requisito de 12º nível na classe, RD 5 fixa, só com armadura pesada.
//
// O motor dava ao Guerreiro a progressão do BÁRBARO a partir do 5º nível, o que
// não existe no livro: todo Guerreiro de 5 a 11 tinha RD que não deveria ter, e
// do 12º em diante tinha o valor errado (ALE-111). Estes testes fixam a regra
// certa, e não o comportamento antigo.
func TestEspecializacaoEmArmadura(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)

	rd := func(class string, level int, powers string, heavy bool) int {
		ch := Character{
			Level:       level,
			Classes:     []CharacterClass{{ClassName: class, Level: level}},
			ClassPowers: powers,
		}
		e := ItemEffects{Flags: map[string]bool{}}
		if heavy {
			e.Flags["armadura-pesada"] = true
		}
		_ = catalogs
		return characterDamageReduction(ch, e).Total
	}
	poder := func(class string) string {
		return `["class.` + class + `.especializacao-em-armadura"]`
	}

	for _, class := range []string{"guerreiro", "cavaleiro"} {
		nome := strings.ToUpper(class[:1]) + class[1:]
		if class == "cavaleiro" {
			nome = "Cavaleiro"
		} else {
			nome = "Guerreiro"
		}

		t.Run(nome+": sem o poder escolhido não há RD, em nenhum nível", func(t *testing.T) {
			for _, level := range []int{5, 11, 12, 20} {
				if got := rd(nome, level, `[]`, true); got != 0 {
					t.Errorf("nível %d sem o poder: RD = %d, want 0", level, got)
				}
			}
		})

		t.Run(nome+": com o poder, só a partir do 12º nível", func(t *testing.T) {
			if got := rd(nome, 11, poder(class), true); got != 0 {
				t.Errorf("nível 11: RD = %d, want 0 — o pré-requisito é 12º", got)
			}
			if got := rd(nome, 12, poder(class), true); got != 5 {
				t.Errorf("nível 12: RD = %d, want 5", got)
			}
		})

		t.Run(nome+": RD 5 é FIXA, não escala com o nível", func(t *testing.T) {
			if got := rd(nome, 20, poder(class), true); got != 5 {
				t.Errorf("nível 20: RD = %d, want 5 — voltou a escalar?", got)
			}
		})

		t.Run(nome+": sem armadura pesada não vale", func(t *testing.T) {
			if got := rd(nome, 20, poder(class), false); got != 0 {
				t.Errorf("sem armadura pesada: RD = %d, want 0", got)
			}
		})
	}

	// O poder é class-qualified: a escolha de uma classe não pode satisfazer a
	// outra num multiclasse. Antes o casamento era por sufixo.
	t.Run("a escolha de uma classe não vale para a outra", func(t *testing.T) {
		if got := rd("Cavaleiro", 12, poder("guerreiro"), true); got != 0 {
			t.Errorf("Cavaleiro com o poder do Guerreiro: RD = %d, want 0", got)
		}
	})

	// "cumulativa com a RD fornecida por Bastião" — as duas descrições se citam.
	t.Run("Bastião e Especialização se ACUMULAM (p54/p55)", func(t *testing.T) {
		ch := Character{
			Level:   12,
			Classes: []CharacterClass{{ClassName: "Cavaleiro", Level: 12}},
			ClassPowers: `["class.cavaleiro.caminho-bastiao",` +
				`"class.cavaleiro.especializacao-em-armadura"]`,
		}
		e := ItemEffects{Flags: map[string]bool{"armadura-pesada": true}}
		if got := characterDamageReduction(ch, e).Total; got != 10 {
			t.Errorf("Bastião + Especialização = %d, want 10", got)
		}
	})
}

// "PETRIFICADO. O personagem fica inconsciente e recebe redução de dano 8."
// (p394). A RD 8 não era modelável enquanto não existia alvo de modificador para
// redução de dano — a do motor vinha só de classe (ALE-115).
func TestPetrificadoGrantsDamageReduction(t *testing.T) {
	rd := func(conds []string, classes ...CharacterClass) RdBreakdown {
		mods := []Modifier{}
		for _, id := range conds {
			mods = append(mods, conditionModifiers(id)...)
		}
		vested := "vested"
		e := ComputeItemEffects([]ActiveItem{{Source: "Condições", Equipped: &vested, Modifiers: mods}})
		e.Flags = map[string]bool{}
		return characterDamageReduction(Character{Level: 10, Classes: classes}, e)
	}

	t.Run("petrificado sozinho dá RD 8", func(t *testing.T) {
		if got := rd([]string{"petrificado"}).Total; got != 8 {
			t.Errorf("RD = %d, want 8", got)
		}
	})

	// p226: efeitos de origens diferentes acumulam. Uma estátua de bárbaro tem
	// a RD da classe E a da condição.
	t.Run("soma com a RD de classe, que é de outra origem", func(t *testing.T) {
		barbaro := CharacterClass{ClassName: "Bárbaro", Level: 11} // RD 6 (p42)
		if got := rd(nil, barbaro).Total; got != 6 {
			t.Fatalf("só o Bárbaro: RD = %d, want 6", got)
		}
		if got := rd([]string{"petrificado"}, barbaro).Total; got != 14 {
			t.Errorf("Bárbaro petrificado: RD = %d, want 14 (6 + 8)", got)
		}
	})

	t.Run("sem condição nenhuma continua zero", func(t *testing.T) {
		if got := rd(nil).Total; got != 0 {
			t.Errorf("RD = %d, want 0", got)
		}
	})
}
