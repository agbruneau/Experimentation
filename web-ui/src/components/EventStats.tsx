import { useStore } from '../store/useStore';

export function EventStats() {
  const { eventStats, resetStats } = useStore();
  const total =
    eventStats.compteOuvert +
    eventStats.depotEffectue +
    eventStats.retraitEffectue +
    eventStats.virementEmis;

  return (
    <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">Statistiques</h2>
        <button
          onClick={resetStats}
          className="text-xs text-slate-400 hover:text-white transition-colors"
        >
          Réinitialiser
        </button>
      </div>

      <div className="space-y-3">
        <StatRow
          label="Comptes Ouverts"
          value={eventStats.compteOuvert}
          color="bg-blue-500"
          total={total}
        />
        <StatRow
          label="Dépôts"
          value={eventStats.depotEffectue}
          color="bg-green-500"
          total={total}
        />
        <StatRow
          label="Retraits"
          value={eventStats.retraitEffectue}
          color="bg-yellow-500"
          total={total}
        />
        <StatRow
          label="Virements"
          value={eventStats.virementEmis}
          color="bg-purple-500"
          total={total}
        />

        <div className="border-t border-slate-700 pt-3 mt-3">
          <div className="flex justify-between text-white">
            <span className="font-medium">Total</span>
            <span className="font-bold">{total}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

interface StatRowProps {
  label: string;
  value: number;
  color: string;
  total: number;
}

function StatRow({ label, value, color, total }: StatRowProps) {
  const percentage = total > 0 ? (value / total) * 100 : 0;

  return (
    <div>
      <div className="flex justify-between text-sm mb-1">
        <span className="text-slate-400">{label}</span>
        <span className="text-white">{value}</span>
      </div>
      <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
        <div
          className={`h-full ${color} transition-all duration-300`}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}
