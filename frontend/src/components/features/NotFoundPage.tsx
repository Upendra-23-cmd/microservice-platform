import { useNavigate } from 'react-router-dom';

export default function NotFoundPage() {
  const navigate = useNavigate();
  return (
    <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center gap-6 text-center px-4">
      <div>
        <p className="text-8xl font-black text-slate-800 select-none">404</p>
        <h1 className="text-xl font-bold text-slate-100 mt-2">Page not found</h1>
        <p className="text-sm text-slate-400 mt-1">The route you requested doesn't exist.</p>
      </div>
      <button
        onClick={() => navigate('/dashboard')}
        className="px-5 py-2 rounded-lg bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-semibold text-sm transition-colors"
      >
        Back to Dashboard
      </button>
    </div>
  );
}
