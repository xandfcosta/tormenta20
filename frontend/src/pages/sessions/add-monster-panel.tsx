import { createSignal } from 'solid-js'
import type { Monster } from '@/shared/api/catalog-types'
import { MonsterPickerPanel } from '@/features/gm-tools/monster-picker-panel'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { toast } from '@/shared/ui/sonner'
import { type EntradaAjustada, AddMonsterDialog } from './add-monster-dialog'
import { MatchPeek } from './match-rail'

/**
 * O bestiário da sessão: clicar numa criatura a ABRE, e adicionar é o gesto
 * seguinte (ALE-208).
 *
 * O card inteiro era o botão de adicionar, então clicar para LER punha a
 * criatura na iniciativa ao vivo com PV do livro e um d20 cru. Não havia
 * caminho para só olhar, e o que entrava era quase sempre errado. Agora o card
 * abre o `AddMonsterDialog`, que mostra o bloco e deixa ajustar PV, iniciativa
 * e quantas antes de qualquer coisa chegar à mesa.
 *
 * A gaveta CONTINUA ABERTA depois de adicionar: uma emboscada é uma viagem, não
 * seis. Em tela larga ela é não modal, e o rastreador segue recebendo clique
 * atrás dela — com o espiar de rodada/vez no cabeçalho para o mestre não perder
 * o fio.
 *
 * Quem abre é o TRILHO das consultas: um overlay por vez (ALE-198).
 */
export function AddMonsterPanel(props: {
  rt: SessionRealtime
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [aberta, setAberta] = createSignal<Monster | null>(null)

  const adiciona = (monster: Monster, ajuste: EntradaAjustada) => {
    // Uma chamada por cópia, e o SERVIDOR numera os repetidos (ALE-192): a tela
    // não pode adivinhar um número que outro cliente acabou de usar. Todas
    // entram com a MESMA iniciativa — é o que a mesa faz com um bando.
    for (let i = 0; i < ajuste.quantidade; i++) {
      props.rt.addEntry({
        label: monster.name,
        initiative: ajuste.initiative,
        type: 'npc',
        monsterId: monster.id,
        hpCurrent: ajuste.hp,
        hpMax: ajuste.hp,
      })
    }
    const uma = ajuste.quantidade === 1
    toast(`${uma ? monster.name : `${ajuste.quantidade}× ${monster.name}`} na iniciativa`, {
      description: `PV ${ajuste.hp} · iniciativa ${ajuste.initiative}.`,
    })
  }

  return (
    <>
      <MonsterPickerPanel
        open={props.open}
        onOpenChange={props.onOpenChange}
        title="Adicionar do bestiário"
        description="Abra a criatura para ler o bloco e ajustar antes de ela entrar."
        header={<MatchPeek rt={props.rt} />}
        itemVerbo="Abrir"
        onPick={setAberta}
      />
      <AddMonsterDialog monster={aberta()} onAdd={adiciona} onClose={() => setAberta(null)} />
    </>
  )
}
