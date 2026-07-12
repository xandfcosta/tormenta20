import { useEffect, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Send, Swords } from 'lucide-react'
import type { Monster } from '@tormenta20/t20-data'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Combobox } from '@/shared/ui/combobox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { useSessionSocket } from '@/shared/realtime/realtime'
import { rollD20 } from '@/shared/lib/dice'

// Mirror of backend `INITIATIVE_MAX_ENTRIES` — server rejects the 51st
// add with a BadRequestException. Frontend clamps too so batch feedback
// arrives in the UI (not just as a silent WS error mid-loop).
const INITIATIVE_MAX_ENTRIES = 50

type EncounterGroup = {
  monster: Monster
  quantity: number
}

/**
 * "Enviar para sessão" — bridges /gm/encounters output into the live
 * session tracker. Dialog lets the GM pick a campaign + an active
 * session, then batches one `initiative-add` per {monster × quantity}
 * suffixed with " #N" when quantity > 1 so combatants stay
 * distinguishable in the tracker. Initiative is rolled client-side per
 * entry (d20 raw — no DEX mod, since encounter builder only knows
 * monsters). Server enforces cap; we clamp locally so the user sees the
 * cutoff before the WS starts throwing.
 */
export function SendEncounterToSessionButton({
  groups,
}: {
  groups: EncounterGroup[]
}) {
  const [open, setOpen] = useState(false)
  const [campaignId, setCampaignId] = useState<string>('')
  const [sessionId, setSessionId] = useState<string>('')
  const [sending, setSending] = useState(false)
  const total = groups.reduce((s, g) => s + g.quantity, 0)

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (!o) {
          setCampaignId('')
          setSessionId('')
          setSending(false)
        }
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm" variant="outline" disabled={groups.length === 0}>
          <Send className="mr-1 size-4" /> Enviar para sessão
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 font-display tracking-wide">
            <Swords className="size-5 text-[color:var(--primary)]" />
            Enviar encontro para sessão
          </DialogTitle>
          <DialogDescription>
            {total} combatente{total === 1 ? '' : 's'} serão adicionados
            à ordem de iniciativa da sessão escolhida. Iniciativa rolada
            no cliente (d20 puro).
          </DialogDescription>
        </DialogHeader>
        {sending ? (
          <SessionSender
            campaignId={Number(campaignId)}
            sessionId={Number(sessionId)}
            groups={groups}
            onDone={() => setOpen(false)}
          />
        ) : (
          <PickerBody
            campaignId={campaignId}
            sessionId={sessionId}
            onCampaign={(v) => {
              setCampaignId(v)
              setSessionId('')
            }}
            onSession={setSessionId}
          />
        )}
        {!sending && (
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button
              disabled={!campaignId || !sessionId}
              onClick={() => setSending(true)}
            >
              <Send className="mr-1 size-4" /> Enviar
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  )
}

function PickerBody({
  campaignId,
  sessionId,
  onCampaign,
  onSession,
}: {
  campaignId: string
  sessionId: string
  onCampaign: (v: string) => void
  onSession: (v: string) => void
}) {
  const campaigns = useQuery(campaignsQueryOptions)
  const sessions = useQuery({
    ...campaignSessionsQueryOptions(Number(campaignId)),
    enabled: !!campaignId,
  })
  const activeSessions =
    sessions.data?.filter((s) => s.status === 'active') ?? []

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <label className="text-xs font-medium" htmlFor="picker-campaign">
          Campanha
        </label>
        <Combobox
          id="picker-campaign"
          options={(campaigns.data ?? []).map((c) => ({
            value: String(c.id),
            label: c.name,
          }))}
          value={campaignId}
          onChange={onCampaign}
          placeholder={
            campaigns.isLoading ? 'Carregando…' : 'Selecionar campanha'
          }
          searchPlaceholder="Buscar campanha…"
          emptyMessage="Nenhuma campanha."
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs font-medium" htmlFor="picker-session">
          Sessão ativa
        </label>
        <Combobox
          id="picker-session"
          options={activeSessions.map((s) => ({
            value: String(s.id),
            label: `Sessão ${s.sessionNumber}${s.title ? ` — ${s.title}` : ''}`,
          }))}
          value={sessionId}
          onChange={onSession}
          placeholder={
            !campaignId
              ? 'Escolha uma campanha primeiro'
              : sessions.isLoading
                ? 'Carregando…'
                : activeSessions.length === 0
                  ? 'Nenhuma sessão ativa'
                  : 'Selecionar sessão'
          }
          searchPlaceholder="Buscar sessão…"
          emptyMessage="Nenhuma sessão ativa. Inicie uma na página da campanha."
        />
      </div>
    </div>
  )
}

// Mounts the socket + fires the batch once. Isolated so the parent
// dialog can unmount it cleanly on cancel (socket teardown happens in
// useSessionSocket's effect cleanup).
function SessionSender({
  campaignId,
  sessionId,
  groups,
  onDone,
}: {
  campaignId: number
  sessionId: number
  groups: EncounterGroup[]
  onDone: () => void
}) {
  const rt = useSessionSocket(campaignId, sessionId)
  const navigate = useNavigate()
  const [fired, setFired] = useState(false)
  const [added, setAdded] = useState(0)

  const currentInInitiative = rt.state.initiative.length
  const remaining = Math.max(0, INITIATIVE_MAX_ENTRIES - currentInInitiative)
  const wanted = groups.reduce((s, g) => s + g.quantity, 0)
  const willAdd = Math.min(wanted, remaining)
  const clipped = wanted > remaining

  // `fired` state gates so the batch runs exactly once when the socket
  // first flips to connected. Extra deps (identities that churn) are
  // safe because the fired short-circuit guarantees a single execution.
  useEffect(() => {
    if (!rt.isConnected || fired) return
    setFired(true)

    let count = 0
    for (const g of groups) {
      for (let i = 1; i <= g.quantity; i++) {
        if (count >= remaining) break
        const label =
          g.quantity > 1 ? `${g.monster.name} #${i}` : g.monster.name
        rt.addEntry({
          label,
          initiative: rollD20(),
          type: 'npc',
          hpCurrent: g.monster.hp,
          hpMax: g.monster.hp,
        })
        count++
      }
      if (count >= remaining) break
    }
    setAdded(count)
    const t = setTimeout(() => {
      onDone()
      navigate({
        to: '/campaigns/$id/sessions/$sid',
        params: { id: String(campaignId), sid: String(sessionId) },
      })
    }, 400)
    return () => clearTimeout(t)
  }, [
    rt.isConnected,
    rt.addEntry,
    fired,
    groups,
    remaining,
    onDone,
    navigate,
    campaignId,
    sessionId,
  ])

  return (
    <div className="space-y-3 text-sm">
      <div className="flex items-center justify-between">
        <span>Conexão</span>
        <Badge variant={rt.isConnected ? 'default' : 'secondary'}>
          {rt.isConnected ? 'Conectado' : 'Conectando…'}
        </Badge>
      </div>
      {clipped && (
        <p className="rounded-md border border-[color:var(--hp-hurt)]/60 bg-[color-mix(in_oklch,var(--hp-hurt)_10%,transparent)] px-3 py-2 text-xs">
          Tracker no limite ({INITIATIVE_MAX_ENTRIES}). Enviando {willAdd}{' '}
          de {wanted} solicitados.
        </p>
      )}
      <div className="flex items-center justify-between">
        <span>Adicionados</span>
        <span className="font-hud tabular-nums text-[color:var(--primary)]">
          {added} / {willAdd}
        </span>
      </div>
    </div>
  )
}
