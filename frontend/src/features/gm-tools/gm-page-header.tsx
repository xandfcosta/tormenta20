import { Link } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button } from '@/shared/ui/button'

/**
 * Shared header for every GM tool sub-page: a back-to-hub button + the page
 * title, with an optional `aside` slot (e.g. a result count or actions).
 * Keeps the five tool pages consistent and de-duplicated.
 */
export function GmPageHeader({
  title,
  aside,
}: {
  title: string
  aside?: ReactNode
}) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <Link to="/gm">
        <Button variant="outline" size="sm" aria-label="Voltar para ferramentas">
          <ArrowLeft className="size-4" />
        </Button>
      </Link>
      <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
      {aside}
    </div>
  )
}
