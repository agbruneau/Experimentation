import { useState } from 'react'
import { Activity, Database, Radio, Gauge } from 'lucide-react'
import FlowVisualization from './components/FlowVisualization'
import SimulationControls from './components/SimulationControls'
import MetricsDashboard from './components/MetricsDashboard'
import { useSimulationStore } from './store/simulationStore'
import { useWebSocket } from './hooks/useWebSocket'

function App() {
  const [activeTab, setActiveTab] = useState<'flow' | 'metrics'>('flow')
  const { status, connectionStatus } = useSimulationStore()

  // Connect to WebSocket
  useWebSocket()

  return (
    <div className="flex flex-col h-screen bg-slate-50">
      {/* Header */}
      <header className="bg-white border-b border-slate-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Radio className="h-8 w-8 text-primary-600" />
            <div>
              <h1 className="text-xl font-bold text-slate-900">EDA-Lab</h1>
              <p className="text-sm text-slate-500">Event Driven Architecture Laboratory</p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <StatusIndicator
              label="WebSocket"
              status={connectionStatus === 'connected' ? 'success' : 'error'}
            />
            <StatusIndicator
              label="Simulation"
              status={status === 'running' ? 'success' : status === 'stopped' ? 'idle' : 'warning'}
            />
          </div>
        </div>
      </header>

      {/* Main Content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <aside className="w-80 bg-white border-r border-slate-200 p-4 overflow-y-auto">
          <SimulationControls />
        </aside>

        {/* Main Area */}
        <main className="flex-1 flex flex-col">
          {/* Tabs */}
          <div className="bg-white border-b border-slate-200 px-4">
            <nav className="flex gap-4">
              <TabButton
                active={activeTab === 'flow'}
                onClick={() => setActiveTab('flow')}
                icon={<Activity className="w-4 h-4" />}
              >
                Flow Visualization
              </TabButton>
              <TabButton
                active={activeTab === 'metrics'}
                onClick={() => setActiveTab('metrics')}
                icon={<Gauge className="w-4 h-4" />}
              >
                Metrics
              </TabButton>
            </nav>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-hidden">
            {activeTab === 'flow' ? (
              <FlowVisualization />
            ) : (
              <MetricsDashboard />
            )}
          </div>
        </main>
      </div>

      {/* Footer */}
      <footer className="bg-white border-t border-slate-200 px-6 py-2">
        <div className="flex items-center justify-between text-sm text-slate-500">
          <div className="flex items-center gap-4">
            <span className="flex items-center gap-1">
              <Database className="w-4 h-4" />
              Events: {useSimulationStore.getState().eventsProduced}
            </span>
          </div>
          <span>EDA-Lab v1.0.0</span>
        </div>
      </footer>
    </div>
  )
}

// Status Indicator Component
function StatusIndicator({ label, status }: { label: string; status: 'success' | 'warning' | 'error' | 'idle' }) {
  const colors = {
    success: 'bg-green-500',
    warning: 'bg-yellow-500',
    error: 'bg-red-500',
    idle: 'bg-slate-400',
  }

  return (
    <div className="flex items-center gap-2">
      <div className={`w-2 h-2 rounded-full ${colors[status]}`} />
      <span className="text-sm text-slate-600">{label}</span>
    </div>
  )
}

// Tab Button Component
function TabButton({
  active,
  onClick,
  icon,
  children
}: {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-2 px-4 py-3 border-b-2 transition-colors ${
        active
          ? 'border-primary-600 text-primary-600'
          : 'border-transparent text-slate-500 hover:text-slate-700'
      }`}
    >
      {icon}
      {children}
    </button>
  )
}

export default App
