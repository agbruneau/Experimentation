import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1'

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Simulation API
export interface StartSimulationRequest {
  scenario?: string
  rate?: number
  duration?: number
  event_types?: string[]
}

export interface StartSimulationResponse {
  simulation_id: string
  status: string
  message?: string
}

export interface SimulationStatus {
  id: string
  status: 'stopped' | 'running' | 'paused'
  scenario: string
  events_produced: number
  events_failed: number
  started_at?: string
  stopped_at?: string
  duration_seconds: number
  rate_requested: number
  rate_actual: number
  last_event_at?: string
}

export interface StopSimulationResponse {
  status: string
  events_produced: number
  duration_seconds: number
  message?: string
}

export interface ProduceEventsRequest {
  event_type: string
  count: number
}

export interface ProduceEventsResponse {
  events_produced: number
  event_ids: string[]
}

export interface EventTypesResponse {
  event_types: string[]
}

export const simulationApi = {
  start: async (request: StartSimulationRequest): Promise<StartSimulationResponse> => {
    const { data } = await api.post<StartSimulationResponse>('/simulation/start', request)
    return data
  },

  stop: async (): Promise<StopSimulationResponse> => {
    const { data } = await api.post<StopSimulationResponse>('/simulation/stop')
    return data
  },

  status: async (): Promise<SimulationStatus> => {
    const { data } = await api.get<SimulationStatus>('/simulation/status')
    return data
  },

  produceEvents: async (request: ProduceEventsRequest): Promise<ProduceEventsResponse> => {
    const { data } = await api.post<ProduceEventsResponse>('/events/produce', request)
    return data
  },

  getEventTypes: async (): Promise<EventTypesResponse> => {
    const { data } = await api.get<EventTypesResponse>('/events/types')
    return data
  },
}

// Bancaire API
export interface Compte {
  id: string
  client_id: string
  type_compte: string
  solde: string
  devise: string
  statut: string
  created_at: string
  updated_at: string
}

export interface Transaction {
  id: string
  compte_id: string
  event_id: string
  type: string
  montant: string
  devise: string
  solde_apres: string
  reference: string
  description: string
  created_at: string
}

export const bancaireApi = {
  getCompte: async (id: string): Promise<Compte> => {
    const { data } = await api.get<Compte>(`/comptes/${id}`)
    return data
  },

  getTransactions: async (compteId: string, limit = 50): Promise<{ transactions: Transaction[] }> => {
    const { data } = await api.get(`/comptes/${compteId}/transactions?limit=${limit}`)
    return data
  },

  getComptesByClient: async (clientId: string): Promise<{ comptes: Compte[] }> => {
    const { data } = await api.get(`/clients/${clientId}/comptes`)
    return data
  },
}

// Health API
export interface HealthStatus {
  status: string
  service: string
  services?: Record<string, string>
  websocket?: { clients: number }
  timestamp: string
}

export const healthApi = {
  check: async (): Promise<HealthStatus> => {
    const { data } = await api.get<HealthStatus>('/health')
    return data
  },
}

export default api
