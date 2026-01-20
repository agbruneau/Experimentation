// Event types from Kafka
export interface CompteOuvert {
  event_id: string;
  timestamp: string;
  compte_id: string;
  client_id: string;
  type_compte: string;
  solde_initial: string;
  devise: string;
}

export interface DepotEffectue {
  event_id: string;
  timestamp: string;
  compte_id: string;
  montant: string;
  devise: string;
  reference: string;
}

export interface RetraitEffectue {
  event_id: string;
  timestamp: string;
  compte_id: string;
  montant: string;
  devise: string;
  reference: string;
}

export interface VirementEmis {
  event_id: string;
  timestamp: string;
  compte_source: string;
  compte_destination: string;
  montant: string;
  devise: string;
  motif: string;
  reference: string;
}

export type BankingEvent = CompteOuvert | DepotEffectue | RetraitEffectue | VirementEmis;

// WebSocket message types
export interface WebSocketMessage {
  type: 'event' | 'status' | 'error';
  topic?: string;
  payload?: BankingEvent;
  timestamp: string;
}

// Simulation status
export interface SimulationStatus {
  running: boolean;
  rate: number;
  events_produced: number;
  start_time?: string;
}

// Service health
export interface ServiceHealth {
  status: 'healthy' | 'degraded' | 'unhealthy';
  service: string;
  backends?: Record<string, boolean>;
  connections?: number;
}

// Flow node data
export interface EventNodeData {
  label: string;
  eventType: string;
  count: number;
  lastEvent?: BankingEvent;
}

export interface ServiceNodeData {
  label: string;
  status: 'healthy' | 'degraded' | 'unhealthy';
  type: 'service' | 'kafka' | 'database';
}
