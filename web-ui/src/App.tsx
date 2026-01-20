import { Header } from './components/Header';
import { SimulationControls } from './components/SimulationControls';
import { EventStats } from './components/EventStats';
import { EventList } from './components/EventList';
import { ArchitectureFlow } from './components/ArchitectureFlow';
import { useWebSocket } from './hooks/useWebSocket';

function App() {
  // Initialize WebSocket connection
  useWebSocket();

  return (
    <div className="min-h-screen bg-slate-900">
      <Header />

      <main className="max-w-7xl mx-auto px-4 py-6">
        {/* Architecture Visualization */}
        <section className="mb-6">
          <h2 className="text-xl font-semibold text-white mb-4">
            Architecture EDA - Pub/Sub
          </h2>
          <ArchitectureFlow />
        </section>

        {/* Controls and Stats */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Left column - Controls */}
          <div className="space-y-6">
            <SimulationControls />
            <EventStats />
          </div>

          {/* Right columns - Event List */}
          <div className="lg:col-span-2 h-[500px]">
            <EventList />
          </div>
        </div>

        {/* Footer info */}
        <footer className="mt-8 text-center text-slate-500 text-sm">
          <p>EDA-Lab - Itération 1: Pub/Sub Pattern</p>
          <p className="mt-1">
            Simulator → Kafka → Bancaire | Gateway → WebSocket → UI
          </p>
        </footer>
      </main>
    </div>
  );
}

export default App;
