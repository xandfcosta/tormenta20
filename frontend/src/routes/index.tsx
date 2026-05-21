import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { meQueryOptions } from '@/lib/queries'

export const Route = createFileRoute('/')({
  component: HomePage,
})

function HomePage() {
  const me = useQuery(meQueryOptions)
  const user = me.data

  return (
    <div className="h-full space-y-4 overflow-y-auto p-6">
      <h1 className="text-3xl font-semibold">Tormenta 20</h1>
      <p className="text-muted-foreground">
        Session and character-sheet manager.
      </p>
      {user ? (
        <p>
          Logged in as <span className="font-medium">{user.name ?? user.email}</span>.
        </p>
      ) : (
        <div className="flex gap-2">
          <Link to="/login"><Button variant="outline">Login</Button></Link>
          <Link to="/register"><Button>Register</Button></Link>
        </div>
      )}
    </div>
  )
}
