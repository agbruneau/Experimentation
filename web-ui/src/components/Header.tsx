import { useStore } from '../store/useStore';

export function Header() {
  const { connected } = useStore();

  return (
    <header className="bg-slate-800 border-b border-slate-700">
      <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="text-2xl font-bold text-white">EDA-Lab</div>
          <span className="text-slate-400 text-sm">Event Driven Architecture Simulation</span>
        </div>
        <div className="flex items-center gap-2">
          <div
            className={`w-3 h-3 rounded-full ${
              connected ? 'bg-success animate-pulse' : 'bg-danger'
            }`}
          />
          <span className="text-sm text-slate-300">
            {connected ? 'Connecté' : 'Déconnecté'}
          </span>
        </div>
      </div>
    </header>
  );
}
