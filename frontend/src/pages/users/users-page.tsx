import { useQuery } from '@tanstack/react-query'
import { usersQueryOptions } from '@/entities/user/queries'

export function UsersPage() {
  const users = useQuery(usersQueryOptions)

  return (
    <div className="h-full space-y-4 overflow-y-auto p-6">
      <h2 className="text-2xl font-semibold">Users</h2>
      {users.isLoading && <p>Loading…</p>}
      {users.isError && (
        <p className="text-destructive">{(users.error as Error).message}</p>
      )}
      <ul className="divide-y rounded-md border">
        {users.data?.map((u) => (
          <li key={u.id} className="flex justify-between px-3 py-2">
            <span>{u.name ?? u.email}</span>
            <span className="text-muted-foreground">{u.email}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
