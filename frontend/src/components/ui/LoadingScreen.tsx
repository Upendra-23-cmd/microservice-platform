// LoadingScreen.tsx
import { Loader2 } from 'lucide-react';

export function LoadingScreen() {
  return (
    <div className="fixed inset-0 bg-slate-950 flex flex-col items-center justify-center gap-4">
      <div className="relative">
        <div className="w-12 h-12 rounded-full border-2 border-cyan-500/20 border-t-cyan-400 animate-spin" />
        <Loader2 size={20} className="absolute inset-0 m-auto text-cyan-400 animate-spin" />
      </div>
      <p className="text-sm text-slate-400 animate-pulse">Loading…</p>
    </div>
  );
}
