import { useStore } from '../store/useStore';
import { WebSocketMessage } from '../types';

const EVENT_TYPE_LABELS: Record<string, string> = {
  'bancaire.compte.ouvert': 'Compte Ouvert',
  'bancaire.depot.effectue': 'Dépôt',
  'bancaire.retrait.effectue': 'Retrait',
  'bancaire.virement.emis': 'Virement',
};

const EVENT_TYPE_COLORS: Record<string, string> = {
  'bancaire.compte.ouvert': 'bg-blue-500',
  'bancaire.depot.effectue': 'bg-green-500',
  'bancaire.retrait.effectue': 'bg-yellow-500',
  'bancaire.virement.emis': 'bg-purple-500',
};

export function EventList() {
  const { events, clearEvents } = useStore();

  return (
    <div className="bg-slate-800 rounded-lg p-4 border border-slate-700 flex flex-col h-full">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">Événements Récents</h2>
        <button
          onClick={clearEvents}
          className="text-xs text-slate-400 hover:text-white transition-colors"
        >
          Effacer
        </button>
      </div>

      <div className="flex-1 overflow-y-auto space-y-2 min-h-0">
        {events.length === 0 ? (
          <div className="text-center text-slate-500 py-8">
            Aucun événement reçu
          </div>
        ) : (
          events.map((event, index) => (
            <EventCard key={`${event.timestamp}-${index}`} event={event} />
          ))
        )}
      </div>
    </div>
  );
}

interface EventCardProps {
  event: WebSocketMessage;
}

function EventCard({ event }: EventCardProps) {
  const topic = event.topic || 'unknown';
  const label = EVENT_TYPE_LABELS[topic] || topic;
  const color = EVENT_TYPE_COLORS[topic] || 'bg-slate-500';
  const time = new Date(event.timestamp).toLocaleTimeString('fr-FR');

  return (
    <div className="bg-slate-700/50 rounded p-3 text-sm">
      <div className="flex items-center justify-between mb-2">
        <span className={`px-2 py-0.5 rounded text-xs text-white ${color}`}>
          {label}
        </span>
        <span className="text-slate-500 text-xs">{time}</span>
      </div>
      <div className="text-slate-300 font-mono text-xs break-all">
        {JSON.stringify(event.payload, null, 0).slice(0, 100)}
        {JSON.stringify(event.payload).length > 100 && '...'}
      </div>
    </div>
  );
}
