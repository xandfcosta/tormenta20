import { RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { useCreationWizard } from './creation-wizard-context'

/**
 * "Recomeçar" — clears the persisted draft and returns to step 1. Confirmed
 * behind a dialog because it is destructive (a stray tap would otherwise wipe
 * an in-progress build). The stock wizard only exposed "Cancelar" on step 1,
 * so there was no way to start over from a later step.
 */
export function WizardRestartButton() {
  const { restart } = useCreationWizard()
  const [open, setOpen] = useState(false)

  const confirm = () => {
    setOpen(false)
    restart()
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" variant="ghost" size="sm">
          <RotateCcw className="mr-1 size-3.5" /> Recomeçar
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Recomeçar a criação?</DialogTitle>
          <DialogDescription>
            Todas as escolhas deste rascunho serão descartadas e você volta ao
            primeiro passo. Esta ação não pode ser desfeita.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancelar
          </Button>
          <Button variant="destructive" onClick={confirm}>
            Recomeçar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
