import { create } from 'zustand'

export type SimulationStatus = 'stopped' | 'running' | 'paused'
export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected'

export interface Event {
  id: string
  type: string
  topic: string
  timestamp: string
  data?: Record<string, unknown>
}

interface SimulationState {
  // Simulation state
  status: SimulationStatus
  simulationId: string | null
  scenario: string
  rate: number
  duration: number
  eventsProduced: number
  eventsFailed: number
  rateActual: number
  startedAt: string | null

  // Connection state
  connectionStatus: ConnectionStatus

  // Recent events
  events: Event[]

  // Actions
  setStatus: (status: SimulationStatus) => void
  setSimulationId: (id: string | null) => void
  setConnectionStatus: (status: ConnectionStatus) => void
  updateMetrics: (metrics: Partial<SimulationState>) => void
  addEvent: (event: Event) => void
  clearEvents: () => void
  reset: () => void
}

const MAX_EVENTS = 100

export const useSimulationStore = create<SimulationState>((set) => ({
  // Initial state
  status: 'stopped',
  simulationId: null,
  scenario: 'default',
  rate: 10,
  duration: 0,
  eventsProduced: 0,
  eventsFailed: 0,
  rateActual: 0,
  startedAt: null,
  connectionStatus: 'disconnected',
  events: [],

  // Actions
  setStatus: (status) => set({ status }),

  setSimulationId: (simulationId) => set({ simulationId }),

  setConnectionStatus: (connectionStatus) => set({ connectionStatus }),

  updateMetrics: (metrics) => set((state) => ({ ...state, ...metrics })),

  addEvent: (event) => set((state) => ({
    events: [event, ...state.events].slice(0, MAX_EVENTS),
    eventsProduced: state.eventsProduced + 1,
  })),

  clearEvents: () => set({ events: [] }),

  reset: () => set({
    status: 'stopped',
    simulationId: null,
    eventsProduced: 0,
    eventsFailed: 0,
    rateActual: 0,
    startedAt: null,
    events: [],
  }),
}))
