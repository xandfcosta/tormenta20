import { type ReactNode, useState } from 'react'
import { Info } from 'lucide-react'
import { getCatalogItem } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import type {
  CharacterItem,
  CreateItemInput,
} from '@/shared/api/api'
import { accentStrong, dimText, subtleText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { CatalogInfoBody } from './catalog-info-body'
import { formatLoad } from './item-describe'

/**
 * Read-only info dialog for a single inventory row. Falls back to a
 * "custom item" message when the row has no catalog link.
 */
export function ItemInfoDialog({ item }: { item: CharacterItem }) {
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            'size-7',
            subtleText,
            'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
          )}
          aria-label={`Informações de ${item.name}`}
        >
          <Info className="size-3.5" />
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-md sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            {item.name}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div
            className={cn(
              'rounded-md border px-3 py-2 text-xs',
              'border-amber-700/30 bg-amber-100/70 dark:border-amber-500/30 dark:bg-zinc-900/60',
            )}
          >
            <p>
              quantidade <span className="font-mono">{item.quantity}</span> •
              espaços <span className="font-mono">{formatLoad(item.slots)}</span>{' '}
              • total{' '}
              <span className={cn('font-mono font-semibold', accentStrong)}>
                {formatLoad(item.quantity * item.slots)}
              </span>
            </p>
            <p className={dimText}>
              equipado:{' '}
              {item.equipped
                ? item.equipped === 'wielded'
                  ? '1 mão'
                  : item.equipped === 'wielded2'
                    ? '2 mãos'
                    : 'vestido'
                : '—'}
            </p>
          </div>
          {catalog ? (
            <CatalogInfoBody catalog={catalog} />
          ) : (
            <p className={cn('text-xs', dimText)}>
              Item customizado, sem dados de catálogo.
            </p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Form dialog used for both "novo item custom" (create) and per-item
 * edit. `initial` seeds the form; leave undefined for create. The
 * shape passes name/quantity/slots — catalog-based items go through
 * `AddCatalogItemDialog` instead.
 */
export function ItemFormDialog({
  title,
  submitLabel,
  trigger,
  initial,
  onSubmit,
}: {
  title: string
  submitLabel: string
  trigger: ReactNode
  initial?: Partial<CreateItemInput>
  onSubmit: (input: CreateItemInput, onError: (e: Error) => void) => void
}) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState(initial?.name ?? '')
  const [quantity, setQuantity] = useState<string>(String(initial?.quantity ?? 1))
  const [slots, setSlots] = useState<string>(String(initial?.slots ?? 1))
  const [error, setError] = useState<string | null>(null)

  const reset = () => {
    setName(initial?.name ?? '')
    setQuantity(String(initial?.quantity ?? 1))
    setSlots(String(initial?.slots ?? 1))
    setError(null)
  }

  const apply = () => {
    const trimmed = name.trim()
    const qty = Number(quantity)
    const sl = Number(slots)
    if (!trimmed) {
      setError('Informe um nome.')
      return
    }
    if (!Number.isInteger(qty) || qty < 1) {
      setError('Quantidade deve ser inteiro ≥ 1.')
      return
    }
    if (!Number.isFinite(sl) || sl < 0.5 || !Number.isInteger(sl * 2)) {
      setError('Espaços deve ser múltiplo de 0,5 (mínimo 0,5).')
      return
    }
    onSubmit({ name: trimmed, quantity: qty, slots: sl }, (e) =>
      setError(e.message),
    )
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
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            {title}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-1">
            <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
              nome
            </span>
            <Input
              value={name}
              onChange={(e) => {
                setName(e.target.value)
                if (error) setError(null)
              }}
              placeholder="Ex: Espada longa"
              autoFocus
              maxLength={80}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <span
                className={cn('text-[10px] uppercase tracking-widest', dimText)}
              >
                quantidade
              </span>
              <NumberInput
                value={quantity}
                onChange={(v) => setQuantity(String(v))}
                min={1}
                max={9999}
                step={1}
                aria-label="Quantidade"
              />
            </div>
            <div className="space-y-1">
              <span
                className={cn('text-[10px] uppercase tracking-widest', dimText)}
              >
                espaços
              </span>
              <NumberInput
                value={slots}
                onChange={(v) => setSlots(String(v))}
                min={0.5}
                max={9999}
                step={0.5}
                aria-label="Espaços"
              />
            </div>
          </div>
          <p className={cn('text-[11px]', dimText)}>
            Espaços é múltiplo de 0,5 (ex.: 0,5 / 1 / 1,5).
          </p>
          {error && (
            <p className="text-xs text-destructive" role="alert">
              {error}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={apply}>
              {submitLabel}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
