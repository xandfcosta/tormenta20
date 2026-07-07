import { useState } from 'react'
import { Minus, Pencil, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { NumberInput } from '@/components/ui/number-input'
import { accentStrong, dimText } from '@/lib/sheet-theme'
import { cn } from '@/lib/utils'

export function ResourceBar({
  label,
  current,
  max,
  fromColor,
  toColor,
  accent,
  className,
  onSetCurrent,
}: {
  label: string
  current: number
  max: number
  fromColor: string
  toColor: string
  accent: string
  className?: string
  onSetCurrent?: (next: number) => void
}) {
  const pct = max > 0 ? Math.max(0, Math.min(100, (current / max) * 100)) : 0
  return (
    <div
      className={cn(
        'rounded-lg border p-2.5',
        'border-amber-700/30 bg-amber-100/60 dark:border-zinc-800 dark:bg-zinc-900/60',
        className,
      )}
    >
      <div className="flex items-baseline justify-between">
        <p className={cn('text-[10px] uppercase tracking-[0.3em]', accent)}>
          {label}
        </p>
        <p className="font-mono text-base">
          <span className="font-bold">{current}</span>
          <span className={dimText}> / {max}</span>
        </p>
      </div>
      <div className="mt-1.5 h-2.5 overflow-hidden rounded-full border border-amber-700/30 bg-amber-50/70 dark:border-zinc-800 dark:bg-zinc-950">
        <div
          className={`h-full bg-gradient-to-r ${fromColor} ${toColor} transition-all`}
          style={{ width: `${pct}%` }}
        />
      </div>
      {onSetCurrent && (
        <ResourceControls
          label={label}
          current={current}
          max={max}
          onSetCurrent={onSetCurrent}
        />
      )}
    </div>
  )
}

function ResourceControls({
  label,
  current,
  max,
  onSetCurrent,
}: {
  label: string
  current: number
  max: number
  onSetCurrent: (next: number) => void
}) {
  return (
    <div className="mt-2 flex items-center gap-1.5">
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-7"
        disabled={current <= 0}
        onClick={() => onSetCurrent(current - 1)}
        aria-label={`Reduzir ${label} em 1`}
      >
        <Minus className="size-3.5" />
      </Button>
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-7"
        disabled={current >= max}
        onClick={() => onSetCurrent(current + 1)}
        aria-label={`Aumentar ${label} em 1`}
      >
        <Plus className="size-3.5" />
      </Button>
      <ResourceAdjustDialog
        label={label}
        current={current}
        max={max}
        onSetCurrent={onSetCurrent}
      />
    </div>
  )
}

function ResourceAdjustDialog({
  label,
  current,
  max,
  onSetCurrent,
}: {
  label: string
  current: number
  max: number
  onSetCurrent: (next: number) => void
}) {
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<'add' | 'remove'>('remove')
  const [amount, setAmount] = useState<string>('')

  const parsedAmount = Math.max(0, Number(amount) || 0)
  const delta = mode === 'add' ? parsedAmount : -parsedAmount
  const previewRaw = current + delta
  const preview = Math.max(0, Math.min(max, previewRaw))
  const clamped = preview !== previewRaw

  const reset = () => {
    setAmount('')
    setMode('remove')
  }

  const apply = () => {
    if (parsedAmount === 0) return
    onSetCurrent(preview)
    setOpen(false)
    reset()
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (!o) reset()
      }}
    >
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="size-7"
          aria-label={`Editar ${label}`}
        >
          <Pencil className="size-3.5" />
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            Ajustar {label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-2">
            <Button
              type="button"
              variant={mode === 'remove' ? 'default' : 'outline'}
              onClick={() => setMode('remove')}
              className="gap-1"
            >
              <Minus className="size-4" /> Remover
            </Button>
            <Button
              type="button"
              variant={mode === 'add' ? 'default' : 'outline'}
              onClick={() => setMode('add')}
              className="gap-1"
            >
              <Plus className="size-4" /> Adicionar
            </Button>
          </div>

          <div className="space-y-1">
            <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
              quantidade
            </span>
            <NumberInput
              value={amount}
              onChange={(v) => setAmount(String(v))}
              min={0}
              max={9999}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') apply()
              }}
              aria-label="Quantidade"
            />
          </div>

          <div
            className={cn(
              'flex items-center justify-between rounded-lg border px-4 py-2',
              'border-amber-700/30 bg-amber-100/70 dark:border-amber-500/30 dark:bg-zinc-900/60',
            )}
          >
            <div className="flex flex-col">
              <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
                novo total
              </span>
              <span className={cn('text-[10px]', dimText)}>
                {current} {delta >= 0 ? '+' : '−'} {Math.abs(delta)}
                {clamped && ' (limitado)'}
              </span>
            </div>
            <span
              className={cn(
                'font-mono text-2xl font-bold',
                accentStrong,
              )}
            >
              {preview}
              <span className={cn('ml-1 text-sm font-normal', dimText)}>
                / {max}
              </span>
            </span>
          </div>

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={apply} disabled={parsedAmount === 0}>
              Aplicar
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
