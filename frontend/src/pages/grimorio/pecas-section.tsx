import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { FramedPanel } from '@/shared/ui/framed-panel'
import { Input } from '@/shared/ui/input'
import { VitalBar } from '@/shared/ui/vital-bar'
import { FieldLabel, SectionLabel, SectionTitle } from '@/shared/ui/section-label'
import { ComparaRotulo, SpecBlock, SpecSection } from './spec-primitives'

/**
 * As peças do kit, desenhadas pelo componente DE VERDADE.
 *
 * Nada aqui é uma imitação com as mesmas classes: é `<Button>`, `<Badge>`,
 * `<Input>` e `<FramedPanel>` importados de `shared/ui`. Uma folha que
 * reproduzisse a aparência à mão passaria a mentir no primeiro dia em que
 * alguém mexesse no componente — que é exatamente o tipo de rombo que esta
 * página existe para não deixar acontecer.
 */

const BOTOES = ['default', 'secondary', 'outline', 'ghost', 'destructive', 'link'] as const
const TAMANHOS = ['xs', 'sm', 'default', 'lg'] as const
const CHIPS = ['default', 'secondary', 'outline', 'ghost', 'destructive'] as const

export function PecasSection() {
  return (
    <SpecSection id="pecas" titulo="Peças">
      <SpecBlock titulo="Botão — variantes">
        {BOTOES.map((v) => (
          <Button variant={v}>{v}</Button>
        ))}
      </SpecBlock>

      <SpecBlock titulo="Botão — tamanhos">
        {TAMANHOS.map((s) => (
          <Button size={s} variant="outline">
            {s}
          </Button>
        ))}
      </SpecBlock>

      <SpecBlock titulo="Chip" nota="O `Badge` tem seis variantes e quatro usos; a cena tem 17 chips escritos à mão (ALE-173, P6).">
        {CHIPS.map((v) => (
          <Badge variant={v}>{v}</Badge>
        ))}
      </SpecBlock>

      <SpecBlock
        titulo="Rótulos"
        nota="Três papéis, não um com variantes: o de campo NÃO é cabeçalho de nada, e vesti-lo de Cinzel mudaria a cara da ficha em 60 lugares. Eram 208 ocorrências em 59 grafias (ALE-173, P2)."
      >
        <div class="w-full space-y-2">
          <SectionTitle as="p">Distribua os atributos</SectionTitle>
          <SectionLabel>Kit da classe</SectionLabel>
          <p class="flex items-baseline gap-1">
            <FieldLabel>for</FieldLabel>
            <span class="font-mono tabular-nums text-foreground">+3</span>
          </p>
        </div>
      </SpecBlock>

      <SpecBlock
        titulo="O espaçamento do título — decisão pendente"
        nota="O mesmo título, no mesmo tamanho, escrito de dois jeitos por partes diferentes do app. A divisão é limpa: apertado em painel de tela densa, folgado em passo de cena. No tamanho de 11px a proporção se INVERTE e o folgado ganha de 18 a 4 — o que sugere uma regra (quanto maior, mais apertado) em vez de descuido."
      >
        <ComparaRotulo
          texto="Distribua os atributos"
          a={{
            classe: 'text-lg tracking-wide',
            nome: 'tracking-wide',
            onde: 'Mochila, Perícias, Grimório, iniciativa',
          }}
          b={{
            classe: 'text-lg tracking-[0.16em]',
            nome: 'tracking-[0.16em]',
            onde: 'passos da forja, ferramentas do /gm, a porta',
          }}
        />
      </SpecBlock>

      <SpecBlock titulo="Campo">
        <div class="w-64">
          <Input placeholder="Nome do combatente" />
        </div>
        <div class="w-64">
          <Input placeholder="Desabilitado" disabled />
        </div>
      </SpecBlock>

      <SpecBlock
        titulo="Barras vitais"
        nota="Cheia, ferida e crítica lado a lado — o par do meio é o que lê como cheia num relance."
      >
        <div class="w-64 space-y-2">
          <VitalBar label="PV" current={38} max={38} kind="hp" />
          <VitalBar label="PV" current={22} max={38} kind="hp" />
          <VitalBar label="PV" current={4} max={38} kind="hp" />
          <VitalBar label="PM" current={12} max={20} kind="mp" />
        </div>
      </SpecBlock>

      <SpecBlock
        titulo="Painel"
        nota="`FramedPanel` tem seis usos; a mesma receita escrita à mão aparece em 28 lugares. Só uma delas tem nome (ALE-173, P5)."
      >
        <FramedPanel class="w-64">
          <p class="text-sm">variante stone</p>
        </FramedPanel>
        <FramedPanel variant="parchment" class="w-64">
          <p class="text-sm">variante parchment</p>
        </FramedPanel>
      </SpecBlock>
    </SpecSection>
  )
}
