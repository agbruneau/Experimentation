import { useCallback } from 'react';
import { useStore } from '../store/useStore';
import { SimulationStatus } from '../types';

const API_BASE = import.meta.env.DEV ? 'http://localhost:8082' : '';

export function useSimulation() {
  const { simulation, setSimulation } = useStore();

  const fetchStatus = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/simulation/status`);
      if (response.ok) {
        const data: SimulationStatus = await response.json();
        setSimulation(data);
      }
    } catch (err) {
      console.error('Failed to fetch simulation status:', err);
    }
  }, [setSimulation]);

  const startSimulation = useCallback(async (rate: number = 1) => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/simulation/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ rate }),
      });

      if (response.ok) {
        const data: SimulationStatus = await response.json();
        setSimulation(data);
        return true;
      }
      return false;
    } catch (err) {
      console.error('Failed to start simulation:', err);
      return false;
    }
  }, [setSimulation]);

  const stopSimulation = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/simulation/stop`, {
        method: 'POST',
      });

      if (response.ok) {
        const data: SimulationStatus = await response.json();
        setSimulation(data);
        return true;
      }
      return false;
    } catch (err) {
      console.error('Failed to stop simulation:', err);
      return false;
    }
  }, [setSimulation]);

  const produceEvent = useCallback(async (eventType: string) => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/events/produce`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ event_type: eventType }),
      });

      return response.ok;
    } catch (err) {
      console.error('Failed to produce event:', err);
      return false;
    }
  }, []);

  return {
    simulation,
    fetchStatus,
    startSimulation,
    stopSimulation,
    produceEvent,
  };
}
