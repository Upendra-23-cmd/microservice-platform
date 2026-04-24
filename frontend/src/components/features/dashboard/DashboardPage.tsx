import {
  Users,
  Package,
  ShoppingCart,
  TrendingUp,
  ArrowUpRight,
  ArrowDownRight,
  Activity,
} from 'lucide-react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
} from 'recharts';
import { useUsers } from '@/hooks/use-users';
import { useProducts } from '@/hooks/use-products';
import { cn } from '@/utils/cn';

// ── Mock trend data (replace with real API calls in production) ──
const revenueData = [
  { month: 'Jan', revenue: 42000 },
  { month: 'Feb', revenue: 53000 },
  { month: 'Mar', revenue: 48000 },
  { month: 'Apr', revenue: 61000 },
  { month: 'May', revenue: 55000 },
  { month: 'Jun', revenue: 72000 },
  { month: 'Jul', revenue: 68000 },
];

const ordersData = [
  { day: 'Mon', orders: 34 },
  { day: 'Tue', orders: 52 },
  { day: 'Wed', orders: 41 },
  { day: 'Thu', orders: 67 },
  { day: 'Fri', orders: 58 },
  { day: 'Sat', orders: 29 },
  { day: 'Sun', orders: 18 },
];

interface StatCardProps {
  title: string;
  value: string | number;
  change: number;
  Icon: React.ElementType;
  color: string;
}

function StatCard({ title, value, change, Icon, color }: StatCardProps) {
  const positive = change >= 0;
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 hover:border-slate-700 transition-colors">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">{title}</p>
          <p className="mt-2 text-2xl font-bold text-slate-100">{value}</p>
        </div>
        <div className={cn('p-2.5 rounded-lg', color)}>
          <Icon size={18} />
        </div>
      </div>
      <div className="mt-3 flex items-center gap-1.5">
        {positive ? (
          <ArrowUpRight size={14} className="text-emerald-400" />
        ) : (
          <ArrowDownRight size={14} className="text-rose-400" />
        )}
        <span className={cn('text-xs font-medium', positive ? 'text-emerald-400' : 'text-rose-400')}>
          {Math.abs(change)}%
        </span>
        <span className="text-xs text-slate-500">vs last month</span>
      </div>
    </div>
  );
}

export default function DashboardPage() {
  const { data: usersData } = useUsers({ pageSize: 1 });
  const { data: productsData } = useProducts({ pageSize: 1 });

  const stats: StatCardProps[] = [
    {
      title: 'Total Users',
      value: usersData?.total ?? '—',
      change: 12.5,
      Icon: Users,
      color: 'bg-cyan-500/10 text-cyan-400',
    },
    {
      title: 'Products',
      value: productsData?.total ?? '—',
      change: 8.1,
      Icon: Package,
      color: 'bg-violet-500/10 text-violet-400',
    },
    {
      title: 'Orders',
      value: '1,248',
      change: -3.2,
      Icon: ShoppingCart,
      color: 'bg-amber-500/10 text-amber-400',
    },
    {
      title: 'Revenue',
      value: '$72,400',
      change: 18.7,
      Icon: TrendingUp,
      color: 'bg-emerald-500/10 text-emerald-400',
    },
  ];

  const tooltipStyle = {
    backgroundColor: '#1e293b',
    border: '1px solid #334155',
    borderRadius: '8px',
    color: '#f1f5f9',
    fontSize: '12px',
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-slate-100">Dashboard</h1>
          <p className="text-sm text-slate-400 mt-0.5">Platform overview</p>
        </div>
        <div className="flex items-center gap-2 bg-emerald-500/10 border border-emerald-500/20 rounded-lg px-3 py-1.5">
          <Activity size={14} className="text-emerald-400 animate-pulse" />
          <span className="text-xs font-medium text-emerald-400">All systems operational</span>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map((s) => (
          <StatCard key={s.title} {...s} />
        ))}
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Revenue Area Chart */}
        <div className="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-xl p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-slate-100">Revenue</h2>
            <span className="text-xs text-slate-400">Last 7 months</span>
          </div>
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={revenueData}>
              <defs>
                <linearGradient id="revenueGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#06b6d4" stopOpacity={0.15} />
                  <stop offset="95%" stopColor="#06b6d4" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
              <XAxis dataKey="month" tick={{ fontSize: 11, fill: '#64748b' }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 11, fill: '#64748b' }} axisLine={false} tickLine={false} tickFormatter={(v) => `$${v / 1000}k`} />
              <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => [`$${v.toLocaleString()}`, 'Revenue']} />
              <Area type="monotone" dataKey="revenue" stroke="#06b6d4" strokeWidth={2} fill="url(#revenueGrad)" />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Orders Bar Chart */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-slate-100">Orders</h2>
            <span className="text-xs text-slate-400">This week</span>
          </div>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={ordersData} barSize={18}>
              <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
              <XAxis dataKey="day" tick={{ fontSize: 11, fill: '#64748b' }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 11, fill: '#64748b' }} axisLine={false} tickLine={false} />
              <Tooltip contentStyle={tooltipStyle} />
              <Bar dataKey="orders" fill="#8b5cf6" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* System Info */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
        <h2 className="text-sm font-semibold text-slate-100 mb-4">Infrastructure</h2>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {[
            { label: 'gRPC', port: '50051', status: 'healthy' },
            { label: 'HTTP Gateway', port: '8080', status: 'healthy' },
            { label: 'PostgreSQL', port: '5432', status: 'healthy' },
            { label: 'MongoDB', port: '27017', status: 'healthy' },
          ].map(({ label, port, status }) => (
            <div key={label} className="flex items-center gap-3 bg-slate-800/60 rounded-lg px-3 py-2.5">
              <div className="w-2 h-2 rounded-full bg-emerald-400 shadow-[0_0_6px_#34d399] shrink-0" />
              <div>
                <p className="text-xs font-medium text-slate-200">{label}</p>
                <p className="text-[10px] text-slate-500">:{port} · {status}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
