package engine

import "testing"

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

// A RD do Guerreiro por armadura pesada NÃO EXISTE no livro como está modelada.
// Varredura completa do PDF (ALE-111): o que há é "Especialização em Armadura"
// (p65), um poder ESCOLHIDO de 12º nível dando RD 5 fixa. A progressão que o
// motor usa é a do Bárbaro, copiada — então um Guerreiro de 5º a 11º ganha RD
// que não deveria ter.
//
// Este teste fixa o comportamento ATUAL de propósito: corrigir muda o número de
// personagens já criados e é decisão do dono. Ele impede regressão silenciosa,
// NÃO valida a regra.
func TestGuerreiroRdRequiresHeavyArmor(t *testing.T) {
	t.Run("sem armadura pesada não há RD, em nenhum nível", func(t *testing.T) {
		for _, level := range []int{1, 5, 17, 20} {
			if got := guerreiroRdForLevel(level, false); got != 0 {
				t.Errorf("nível %d sem armadura pesada: RD = %d, want 0", level, got)
			}
		}
	})

	t.Run("com armadura pesada segue a progressão do Bárbaro", func(t *testing.T) {
		for _, level := range []int{4, 5, 11, 20} {
			if got, want := guerreiroRdForLevel(level, true), barbaroRdForLevel(level); got != want {
				t.Errorf("nível %d: RD = %d, want %d", level, got, want)
			}
		}
	})
}
