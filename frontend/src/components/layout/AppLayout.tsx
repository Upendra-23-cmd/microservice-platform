import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import {
  LayoutDashboard,
  Users,
  Package,
  LogOut,
  Menu,
  X,
  Sun,
  Moon,
  ChevronRight,
} from 'lucide-react';
import { useAuthStore } from '@/store/auth.store';
import { useUIStore } from '@/store/ui.store';
import { cn } from '@/utils/cn';

const NAV_ITEMS = [
  { to: '/dashboard', label: 'Dashboard', Icon: LayoutDashboard },
  { to: '/users',     label: 'Users',     Icon: Users },
  { to: '/products',  label: 'Products',  Icon: Package },
];

export function AppLayout() {
  const { user, logout } = useAuthStore();
  const { sidebarOpen, toggleSidebar, theme, toggleTheme } = useUIStore();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="flex h-screen overflow-hidden bg-slate-950 text-slate-100">
      {/* ── Sidebar ────────────────────────────────────────── */}
      <aside
        className={cn(
          'flex flex-col border-r border-slate-800 bg-slate-900 transition-all duration-300 ease-in-out',
          sidebarOpen ? 'w-64' : 'w-16',
        )}
      >
        {/* Logo */}
        <div className="flex h-16 items-center justify-between px-4 border-b border-slate-800">
          {sidebarOpen && (
            <span className="text-sm font-bold tracking-widest uppercase text-cyan-400">
              MSP
            </span>
          )}
          <button
            onClick={toggleSidebar}
            className="ml-auto rounded-md p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-100 transition-colors"
          >
            {sidebarOpen ? <X size={18} /> : <Menu size={18} />}
          </button>
        </div>

        {/* Nav Items */}
        <nav className="flex-1 space-y-1 p-2 mt-2">
          {NAV_ITEMS.map(({ to, label, Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all',
                  isActive
                    ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/20'
                    : 'text-slate-400 hover:bg-slate-800 hover:text-slate-100',
                  !sidebarOpen && 'justify-center',
                )
              }
              title={!sidebarOpen ? label : undefined}
            >
              <Icon size={18} className="shrink-0" />
              {sidebarOpen && <span>{label}</span>}
              {sidebarOpen && (
                <ChevronRight size={14} className="ml-auto opacity-40" />
              )}
            </NavLink>
          ))}
        </nav>

        {/* Bottom: User + actions */}
        <div className="border-t border-slate-800 p-2 space-y-1">
          <button
            onClick={toggleTheme}
            className={cn(
              'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-slate-400',
              'hover:bg-slate-800 hover:text-slate-100 transition-colors',
              !sidebarOpen && 'justify-center',
            )}
          >
            {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
            {sidebarOpen && <span>Theme</span>}
          </button>

          {sidebarOpen && user && (
            <div className="px-3 py-2 rounded-lg bg-slate-800/50">
              <p className="text-xs font-medium text-slate-100 truncate">
                {user.firstName} {user.lastName}
              </p>
              <p className="text-xs text-slate-400 truncate">{user.email}</p>
              <span className="mt-1 inline-block rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide bg-cyan-500/20 text-cyan-400">
                {user.role}
              </span>
            </div>
          )}

          <button
            onClick={handleLogout}
            className={cn(
              'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm',
              'text-rose-400 hover:bg-rose-500/10 transition-colors',
              !sidebarOpen && 'justify-center',
            )}
          >
            <LogOut size={18} />
            {sidebarOpen && <span>Logout</span>}
          </button>
        </div>
      </aside>

      {/* ── Main Content ───────────────────────────────────── */}
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}
