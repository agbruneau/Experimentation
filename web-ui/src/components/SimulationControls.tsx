import { useState, useEffect } from 'react';
import { useSimulation } from '../hooks/useSimulation';

export function SimulationControls() {
  const { simulation, fetchStatus, startSimulation, stopSimulation, produceEvent } =
    useSimulation();
  const [rate, setRate] = useState(1);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  const handleStart = async () => {
    setLoading(true);
    await startSimulation(rate);
    setLoading(false);
  };

  const handleStop = async () => {
    setLoading(true);
    await stopSimulation();
    setLoading(false);
  };

  const handleProduceEvent = async (eventType: string) => {
    await produceEvent(eventType);
  };

  return (
    <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
      <h2 className="text-lg font-semibold text-white mb-4">Contrôles de Simulation</h2>

      <div className="space-y-4">
        {/* Rate control */}
        <div>
          <label className="block text-sm text-slate-400 mb-1">
            Taux d'événements (par seconde)
          </label>
          <input
            type="range"
            min="0.1"
            max="10"
            step="0.1"
            value={rate}
            onChange={(e) => setRate(parseFloat(e.target.value))}
            className="w-full"
            disabled={simulation.running}
          />
          <div className="text-center text-white">{rate.toFixed(1)} evt/s</div>
        </div>

        {/* Start/Stop buttons */}
        <div className="flex gap-2">
          <button
            onClick={handleStart}
            disabled={simulation.running || loading}
            className={`flex-1 py-2 px-4 rounded font-medium transition-colors ${
              simulation.running || loading
                ? 'bg-slate-600 text-slate-400 cursor-not-allowed'
                : 'bg-success text-white hover:bg-green-600'
            }`}
          >
            {loading ? '...' : 'Démarrer'}
          </button>
          <button
            onClick={handleStop}
            disabled={!simulation.running || loading}
            className={`flex-1 py-2 px-4 rounded font-medium transition-colors ${
              !simulation.running || loading
                ? 'bg-slate-600 text-slate-400 cursor-not-allowed'
                : 'bg-danger text-white hover:bg-red-600'
            }`}
          >
            {loading ? '...' : 'Arrêter'}
          </button>
        </div>

        {/* Status */}
        <div className="text-sm text-slate-400">
          <div className="flex justify-between">
            <span>État:</span>
            <span className={simulation.running ? 'text-success' : 'text-slate-500'}>
              {simulation.running ? 'En cours' : 'Arrêté'}
            </span>
          </div>
          <div className="flex justify-between">
            <span>Événements produits:</span>
            <span className="text-white">{simulation.events_produced}</span>
          </div>
        </div>

        {/* Manual event production */}
        <div className="border-t border-slate-700 pt-4 mt-4">
          <h3 className="text-sm font-medium text-slate-300 mb-2">Produire manuellement</h3>
          <div className="grid grid-cols-2 gap-2">
            <button
              onClick={() => handleProduceEvent('compte_ouvert')}
              className="py-1.5 px-3 bg-primary-600 text-white rounded text-sm hover:bg-primary-700 transition-colors"
            >
              Compte Ouvert
            </button>
            <button
              onClick={() => handleProduceEvent('depot_effectue')}
              className="py-1.5 px-3 bg-primary-600 text-white rounded text-sm hover:bg-primary-700 transition-colors"
            >
              Dépôt
            </button>
            <button
              onClick={() => handleProduceEvent('retrait_effectue')}
              className="py-1.5 px-3 bg-primary-600 text-white rounded text-sm hover:bg-primary-700 transition-colors"
            >
              Retrait
            </button>
            <button
              onClick={() => handleProduceEvent('virement_emis')}
              className="py-1.5 px-3 bg-primary-600 text-white rounded text-sm hover:bg-primary-700 transition-colors"
            >
              Virement
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
