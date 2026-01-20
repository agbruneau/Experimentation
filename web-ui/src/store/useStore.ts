import { create } from 'zustand';
import { BankingEvent, SimulationStatus, WebSocketMessage } from '../types';

interface EventStats {
  compteOuvert: number;
  depotEffectue: number;
  retraitEffectue: number;
  virementEmis: number;
}

interface AppState {
  // WebSocket connection
  connected: boolean;
  setConnected: (connected: boolean) => void;

  // Events
  events: WebSocketMessage[];
  addEvent: (event: WebSocketMessage) => void;
  clearEvents: () => void;

  // Event statistics
  eventStats: EventStats;
  incrementStat: (eventType: string) => void;
  resetStats: () => void;

  // Simulation
  simulation: SimulationStatus;
  setSimulation: (status: SimulationStatus) => void;

  // Last events by type
  lastEvents: Record<string, BankingEvent>;
  setLastEvent: (eventType: string, event: BankingEvent) => void;
}

const MAX_EVENTS = 100;

export const useStore = create<AppState>((set) => ({
  // WebSocket connection
  connected: false,
  setConnected: (connected) => set({ connected }),

  // Events (limited to last 100)
  events: [],
  addEvent: (event) =>
    set((state) => ({
      events: [event, ...state.events].slice(0, MAX_EVENTS),
    })),
  clearEvents: () => set({ events: [] }),

  // Event statistics
  eventStats: {
    compteOuvert: 0,
    depotEffectue: 0,
    retraitEffectue: 0,
    virementEmis: 0,
  },
  incrementStat: (eventType) =>
    set((state) => {
      const stats = { ...state.eventStats };
      switch (eventType) {
        case 'bancaire.compte.ouvert':
          stats.compteOuvert++;
          break;
        case 'bancaire.depot.effectue':
          stats.depotEffectue++;
          break;
        case 'bancaire.retrait.effectue':
          stats.retraitEffectue++;
          break;
        case 'bancaire.virement.emis':
          stats.virementEmis++;
          break;
      }
      return { eventStats: stats };
    }),
  resetStats: () =>
    set({
      eventStats: {
        compteOuvert: 0,
        depotEffectue: 0,
        retraitEffectue: 0,
        virementEmis: 0,
      },
    }),

  // Simulation status
  simulation: {
    running: false,
    rate: 1,
    events_produced: 0,
  },
  setSimulation: (status) => set({ simulation: status }),

  // Last events by type
  lastEvents: {},
  setLastEvent: (eventType, event) =>
    set((state) => ({
      lastEvents: { ...state.lastEvents, [eventType]: event },
    })),
}));
