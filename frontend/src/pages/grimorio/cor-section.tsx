import { ColorSwatch, SpecBlock, SpecSection } from './spec-primitives'

/**
 * As cores da casa, agrupadas pelo PAPEL que cada uma tem — não pela ordem em
 * que foram declaradas no CSS.
 *
 * Duas armadilhas registradas ficam à vista aqui, e é metade do motivo desta
 * seção existir: `--hp-hurt` é âmbar em matiz 70, a um fio do dourado em 85, e
 * uma barra pintada com ele lê como CHEIA — quem quer "ruim" usa
 * `--hp-critical`. E o `--grimorio-purple` significa "arcano" e não tem NENHUM
 * uso, enquanto vinte lugares escrevem `violet` cru do Tailwind (ALE-173, P3).
 */

const SUPERFICIES = [
  { classe: 'bg-grimorio-bg', token: '--grimorio-bg', nota: 'fundo da cena' },
  { classe: 'bg-grimorio-bg-2', token: '--grimorio-bg-2', nota: 'fundo elevado' },
  { classe: 'bg-grimorio-panel', token: '--grimorio-panel', nota: 'painel' },
  { classe: 'bg-grimorio-panel-raised', token: '--grimorio-panel-raised', nota: 'painel elevado' },
]

const ACENTOS = [
  { classe: 'bg-grimorio-gold', token: '--grimorio-gold', nota: 'a VEZ e a AÇÃO (ALE-200)' },
  { classe: 'bg-grimorio-iron', token: '--grimorio-iron', nota: 'borda padrão' },
  { classe: 'bg-grimorio-iron-light', token: '--grimorio-iron-light', nota: 'borda em relevo' },
  { classe: 'bg-grimorio-crimson', token: '--grimorio-crimson', nota: 'destruir, e o crachá' },
  { classe: 'bg-grimorio-purple', token: '--grimorio-purple', nota: 'arcano — ZERO usos' },
  { classe: 'bg-grimorio-parchment', token: '--grimorio-parchment', nota: 'superfície clara' },
]

/**
 * Os quatro papéis, cada um com o par bloco + tinta (ALE-173, P3). Eles ficam
 * lado a lado de propósito: é aqui que se vê que a cor de preencher e a de
 * escrever NÃO podem ser a mesma — a de baixo passa de 4.5:1 e a de cima não.
 */
const PAPEIS = [
  { classe: 'bg-penalty', token: '--penalty', nota: 'penalidade — bloco' },
  { classe: 'bg-penalty-ink', token: '--penalty-ink', nota: 'penalidade — tinta' },
  { classe: 'bg-bonus', token: '--bonus', nota: 'bônus — bloco' },
  { classe: 'bg-bonus-ink', token: '--bonus-ink', nota: 'bônus — tinta' },
  { classe: 'bg-arcane', token: '--arcane', nota: 'arcano — bloco' },
  { classe: 'bg-arcane-ink', token: '--arcane-ink', nota: 'arcano — tinta' },
  { classe: 'bg-warning', token: '--warning', nota: 'aviso — bloco' },
  { classe: 'bg-warning-ink', token: '--warning-ink', nota: 'aviso — tinta' },
  { classe: 'bg-marker', token: '--marker', nota: 'crachá — bloco' },
  { classe: 'bg-marker-foreground', token: '--marker-foreground', nota: 'crachá — tinta' },
]

const VITAIS = [
  { classe: 'bg-[var(--hp-full)]', token: '--hp-full', nota: 'vida cheia' },
  { classe: 'bg-[var(--hp-hurt)]', token: '--hp-hurt', nota: 'ferido — LÊ como cheia' },
  { classe: 'bg-[var(--hp-critical)]', token: '--hp-critical', nota: 'crítico; use este p/ ruim' },
  { classe: 'bg-[var(--mp-arcane)]', token: '--mp-arcane', nota: 'mana' },
]

export function CorSection() {
  return (
    <SpecSection id="cor" titulo="Cor">
      <SpecBlock
        titulo="Superfícies"
        nota="A pilha do escuro para o claro. Toda cena empilha nesta ordem: fundo, painel, painel elevado."
      >
        {SUPERFICIES.map((c) => (
          <ColorSwatch superficie classe={c.classe} token={c.token} nota={c.nota} />
        ))}
      </SpecBlock>

      <SpecBlock
        titulo="Acentos"
        nota="Ouro é a vez E a ação: desde a ALE-200 ele preenche o botão principal, e o crimson ficou reservado para destruir. Antes os dois eram o MESMO matiz 25, separados por 0,09 de luminosidade — e no print do dono o botão rotineiro parecia mais perigoso que o que apaga. A razão ao lado é contra o painel: abaixo de 4.5:1 a cor não serve de texto pequeno, só de bloco — e é por isso que a cena alcança cores cruas do Tailwind quando precisa ESCREVER em vermelho ou roxo (ALE-173, P3)."
      >
        {ACENTOS.map((c) => (
          <ColorSwatch classe={c.classe} token={c.token} nota={c.nota} />
        ))}
      </SpecBlock>

      <SpecBlock
        titulo="Papéis — bloco e tinta"
        nota="A cor de preencher e a de escrever são duas. As de bloco ficam abaixo de 4.5:1 e por isso só servem de fundo, barra e borda; as de tinta ficam todas na mesma luminosidade do dourado, e é com elas que se ESCREVE. O aviso e o dourado são vizinhos de matiz (70 e 85) — estão aqui um perto do outro para essa proximidade ser julgada, não descoberta depois."
      >
        {PAPEIS.map((c) => (
          <ColorSwatch classe={c.classe} token={c.token} nota={c.nota} />
        ))}
      </SpecBlock>

      {/* A terceira superfície, e a razão de ela ter bloco próprio: as duas
          acima medem contra o painel ESCURO, e uma tinta de pergaminho medida
          ali dá um número que não significa nada. O `--grimorio-parchment-crimson`
          existe porque o crimson base sobre o creme dá 3,95:1 (ALE-237), e sem
          um exemplo aqui a folha mentiria por omissão — quem consultasse a
          paleta não saberia que a superfície clara tem tinta própria. */}
      <SpecBlock
        titulo="Tinta sobre pergaminho"
        nota="O livro de campanhas é creme, e a conta inverte: o que é legível no painel escuro pode não ser aqui. O crimson base rende 3,95:1 sobre este fundo — abaixo do mínimo de texto —, e por isso a superfície clara tem tinta própria. Ela NÃO é um ajuste no crimson base: aquele é o `--destructive`, medido com branco por cima, e mexer nele mudaria o botão que apaga por um motivo que não é dele."
      >
        <div class="grimorio-parchment-bg flex items-center gap-3 rounded-sm p-3">
          <span class="font-heading text-sm uppercase tracking-wide text-grimorio-parchment-crimson">
            Sessão ao vivo
          </span>
          <code class="font-mono text-3xs text-grimorio-parchment-ink">--grimorio-parchment-crimson</code>
        </div>
      </SpecBlock>

      <SpecBlock
        titulo="Vitais"
        nota="O ferido e o dourado ficam lado a lado de propósito: eles são vizinhos de matiz (70 e 85), e é por isso que uma barra pintada de ferido lê como cheia."
      >
        {VITAIS.map((c) => (
          <ColorSwatch classe={c.classe} token={c.token} nota={c.nota} />
        ))}
      </SpecBlock>
    </SpecSection>
  )
}
