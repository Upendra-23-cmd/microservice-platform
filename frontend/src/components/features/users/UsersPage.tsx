import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Search,
  Plus,
  Trash2,
  ChevronLeft,
  ChevronRight,
  Loader2,
  UserCircle,
  X,
} from 'lucide-react';
import { useUsers, useCreateUser, useDeleteUser } from '@/hooks/use-users';
import type { User } from '@/types';
import { cn } from '@/utils/cn';
import { format } from 'date-fns';

// ── Schema ────────────────────────────────────────────────────
const createSchema = z.object({
  firstName: z.string().min(1, 'Required'),
  lastName: z.string().min(1, 'Required'),
  email: z.string().email('Invalid email'),
  password: z.string().min(8, 'Min 8 characters'),
});
type CreateForm = z.infer<typeof createSchema>;

// ── Role Badge ────────────────────────────────────────────────
const roleBadge: Record<string, string> = {
  admin:   'bg-rose-500/10 text-rose-400 border-rose-500/20',
  manager: 'bg-violet-500/10 text-violet-400 border-violet-500/20',
  member:  'bg-cyan-500/10 text-cyan-400 border-cyan-500/20',
  guest:   'bg-slate-500/10 text-slate-400 border-slate-500/20',
};

// ── Create User Modal ─────────────────────────────────────────
function CreateUserModal({ onClose }: { onClose: () => void }) {
  const { mutateAsync, isPending } = useCreateUser();
  const { register, handleSubmit, formState: { errors } } = useForm<CreateForm>({
    resolver: zodResolver(createSchema),
  });

  const onSubmit = async (data: CreateForm) => {
    await mutateAsync(data);
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="w-full max-w-md bg-slate-900 border border-slate-700 rounded-2xl p-6 shadow-2xl">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-base font-semibold text-slate-100">Create User</h2>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-100 transition-colors">
            <X size={18} />
          </button>
        </div>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            {(['firstName', 'lastName'] as const).map((field) => (
              <div key={field}>
                <label className="block text-xs text-slate-400 mb-1 capitalize">
                  {field === 'firstName' ? 'First Name' : 'Last Name'}
                </label>
                <input
                  {...register(field)}
                  className={cn(inputClass, errors[field] && 'border-rose-500/60')}
                  placeholder={field === 'firstName' ? 'John' : 'Doe'}
                />
                {errors[field] && <p className="mt-1 text-xs text-rose-400">{errors[field]?.message}</p>}
              </div>
            ))}
          </div>
          {(['email', 'password'] as const).map((field) => (
            <div key={field}>
              <label className="block text-xs text-slate-400 mb-1 capitalize">{field}</label>
              <input
                {...register(field)}
                type={field === 'password' ? 'password' : 'email'}
                className={cn(inputClass, errors[field] && 'border-rose-500/60')}
                placeholder={field === 'email' ? 'john@example.com' : '••••••••'}
              />
              {errors[field] && <p className="mt-1 text-xs text-rose-400">{errors[field]?.message}</p>}
            </div>
          ))}
          <div className="flex gap-3 pt-2">
            <button type="button" onClick={onClose} className={secondaryBtnClass}>Cancel</button>
            <button type="submit" disabled={isPending} className={cn(primaryBtnClass, 'flex-1')}>
              {isPending ? <Loader2 size={15} className="animate-spin mx-auto" /> : 'Create User'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

const inputClass =
  'w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40 transition-all';
const primaryBtnClass =
  'flex items-center justify-center gap-2 rounded-lg px-4 py-2 bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-semibold text-sm transition-colors disabled:opacity-50';
const secondaryBtnClass =
  'rounded-lg px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm transition-colors';

// ── Users Page ────────────────────────────────────────────────
export default function UsersPage() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [showCreate, setShowCreate] = useState(false);

  const { data, isLoading } = useUsers({ page, pageSize: 10, search: search || undefined });
  const { mutate: deleteUser } = useDeleteUser();

  const users: User[] = data?.data ?? [];
  const totalPages = data?.pages ?? 1;

  return (
    <div className="p-6 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-slate-100">Users</h1>
          <p className="text-sm text-slate-400 mt-0.5">
            {data?.total ?? '—'} total users
          </p>
        </div>
        <button onClick={() => setShowCreate(true)} className={primaryBtnClass}>
          <Plus size={16} /> New User
        </button>
      </div>

      {/* Search */}
      <div className="relative max-w-sm">
        <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
        <input
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          placeholder="Search users…"
          className={cn(inputClass, 'pl-9')}
        />
      </div>

      {/* Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-800 bg-slate-800/40">
                {['User', 'Role', 'Status', 'Joined', 'Actions'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {isLoading ? (
                <tr>
                  <td colSpan={5} className="py-16 text-center">
                    <Loader2 size={24} className="animate-spin text-cyan-400 mx-auto" />
                  </td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-16 text-center text-slate-500 text-sm">
                    No users found
                  </td>
                </tr>
              ) : (
                users.map((u) => (
                  <tr key={u.id} className="hover:bg-slate-800/40 transition-colors group">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 rounded-full bg-slate-700 flex items-center justify-center shrink-0">
                          <UserCircle size={18} className="text-slate-400" />
                        </div>
                        <div>
                          <p className="font-medium text-slate-100">
                            {u.firstName} {u.lastName}
                          </p>
                          <p className="text-xs text-slate-400">{u.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className={cn('inline-flex px-2 py-0.5 rounded text-[11px] font-semibold uppercase tracking-wider border', roleBadge[u.role] ?? roleBadge.guest)}>
                        {u.role}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={cn('inline-flex items-center gap-1.5 text-xs font-medium', u.isActive ? 'text-emerald-400' : 'text-slate-500')}>
                        <span className={cn('w-1.5 h-1.5 rounded-full', u.isActive ? 'bg-emerald-400' : 'bg-slate-500')} />
                        {u.isActive ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-slate-400 text-xs">
                      {format(new Date(u.createdAt), 'MMM d, yyyy')}
                    </td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => deleteUser(u.id)}
                        className="opacity-0 group-hover:opacity-100 p-1.5 rounded text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 transition-all"
                      >
                        <Trash2 size={14} />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-slate-800">
            <span className="text-xs text-slate-400">
              Page {page} of {totalPages}
            </span>
            <div className="flex gap-1">
              <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1} className={paginationBtn}>
                <ChevronLeft size={15} />
              </button>
              <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page === totalPages} className={paginationBtn}>
                <ChevronRight size={15} />
              </button>
            </div>
          </div>
        )}
      </div>

      {showCreate && <CreateUserModal onClose={() => setShowCreate(false)} />}
    </div>
  );
}

const paginationBtn =
  'p-1.5 rounded text-slate-400 hover:bg-slate-800 hover:text-slate-100 disabled:opacity-30 disabled:cursor-not-allowed transition-colors';
