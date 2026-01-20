import { useCallback, useMemo } from 'react';
import ReactFlow, {
  Background,
  Controls,
  Edge,
  Node,
  NodeTypes,
  Handle,
  Position,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useStore } from '../store/useStore';

// Custom node for services
function ServiceNode({ data }: { data: { label: string; type: string; status?: string } }) {
  const bgColor = {
    service: 'bg-primary-600',
    kafka: 'bg-kafka',
    database: 'bg-orange-600',
    ui: 'bg-purple-600',
  }[data.type] || 'bg-slate-600';

  const statusColor = {
    healthy: 'bg-success',
    degraded: 'bg-warning',
    unhealthy: 'bg-danger',
  }[data.status || 'healthy'];

  return (
    <div className={`${bgColor} rounded-lg p-4 min-w-[140px] shadow-lg border border-slate-600`}>
      <Handle type="target" position={Position.Left} className="!bg-slate-400" />
      <div className="flex items-center gap-2">
        {data.status && (
          <div className={`w-2 h-2 rounded-full ${statusColor}`} />
        )}
        <span className="text-white font-medium text-sm">{data.label}</span>
      </div>
      <Handle type="source" position={Position.Right} className="!bg-slate-400" />
    </div>
  );
}

// Custom node for topics
function TopicNode({ data }: { data: { label: string; count: number } }) {
  return (
    <div className="bg-slate-700 rounded p-3 min-w-[120px] shadow border border-slate-600">
      <Handle type="target" position={Position.Left} className="!bg-slate-400" />
      <div className="text-white text-xs font-medium mb-1">{data.label}</div>
      <div className="text-primary-400 text-lg font-bold">{data.count}</div>
      <Handle type="source" position={Position.Right} className="!bg-slate-400" />
    </div>
  );
}

const nodeTypes: NodeTypes = {
  service: ServiceNode,
  topic: TopicNode,
};

export function ArchitectureFlow() {
  const { eventStats, connected } = useStore();

  const nodes: Node[] = useMemo(
    () => [
      // Services
      {
        id: 'simulator',
        type: 'service',
        position: { x: 50, y: 150 },
        data: { label: 'Simulator', type: 'service', status: 'healthy' },
      },
      {
        id: 'kafka',
        type: 'service',
        position: { x: 250, y: 150 },
        data: { label: 'Kafka', type: 'kafka', status: 'healthy' },
      },
      {
        id: 'bancaire',
        type: 'service',
        position: { x: 600, y: 80 },
        data: { label: 'Bancaire', type: 'service', status: 'healthy' },
      },
      {
        id: 'gateway',
        type: 'service',
        position: { x: 600, y: 220 },
        data: { label: 'Gateway', type: 'service', status: 'healthy' },
      },
      {
        id: 'postgres',
        type: 'service',
        position: { x: 800, y: 80 },
        data: { label: 'PostgreSQL', type: 'database' },
      },
      {
        id: 'webui',
        type: 'service',
        position: { x: 800, y: 220 },
        data: { label: 'Web UI', type: 'ui', status: connected ? 'healthy' : 'unhealthy' },
      },
      // Topics
      {
        id: 'topic-compte',
        type: 'topic',
        position: { x: 420, y: 30 },
        data: { label: 'compte.ouvert', count: eventStats.compteOuvert },
      },
      {
        id: 'topic-depot',
        type: 'topic',
        position: { x: 420, y: 110 },
        data: { label: 'depot.effectue', count: eventStats.depotEffectue },
      },
      {
        id: 'topic-retrait',
        type: 'topic',
        position: { x: 420, y: 190 },
        data: { label: 'retrait.effectue', count: eventStats.retraitEffectue },
      },
      {
        id: 'topic-virement',
        type: 'topic',
        position: { x: 420, y: 270 },
        data: { label: 'virement.emis', count: eventStats.virementEmis },
      },
    ],
    [eventStats, connected]
  );

  const edges: Edge[] = useMemo(
    () => [
      // Simulator -> Kafka
      { id: 'e-sim-kafka', source: 'simulator', target: 'kafka', animated: true, style: { stroke: '#3b82f6' } },
      // Kafka -> Topics
      { id: 'e-kafka-t1', source: 'kafka', target: 'topic-compte', style: { stroke: '#64748b' } },
      { id: 'e-kafka-t2', source: 'kafka', target: 'topic-depot', style: { stroke: '#64748b' } },
      { id: 'e-kafka-t3', source: 'kafka', target: 'topic-retrait', style: { stroke: '#64748b' } },
      { id: 'e-kafka-t4', source: 'kafka', target: 'topic-virement', style: { stroke: '#64748b' } },
      // Topics -> Bancaire
      { id: 'e-t1-banc', source: 'topic-compte', target: 'bancaire', animated: true, style: { stroke: '#10b981' } },
      { id: 'e-t2-banc', source: 'topic-depot', target: 'bancaire', animated: true, style: { stroke: '#10b981' } },
      { id: 'e-t3-banc', source: 'topic-retrait', target: 'bancaire', animated: true, style: { stroke: '#10b981' } },
      { id: 'e-t4-banc', source: 'topic-virement', target: 'bancaire', animated: true, style: { stroke: '#10b981' } },
      // Topics -> Gateway
      { id: 'e-t1-gw', source: 'topic-compte', target: 'gateway', animated: true, style: { stroke: '#a855f7' } },
      { id: 'e-t2-gw', source: 'topic-depot', target: 'gateway', animated: true, style: { stroke: '#a855f7' } },
      { id: 'e-t3-gw', source: 'topic-retrait', target: 'gateway', animated: true, style: { stroke: '#a855f7' } },
      { id: 'e-t4-gw', source: 'topic-virement', target: 'gateway', animated: true, style: { stroke: '#a855f7' } },
      // Bancaire -> PostgreSQL
      { id: 'e-banc-pg', source: 'bancaire', target: 'postgres', style: { stroke: '#f97316' } },
      // Gateway -> Web UI
      { id: 'e-gw-ui', source: 'gateway', target: 'webui', animated: true, style: { stroke: '#a855f7' } },
    ],
    []
  );

  const onInit = useCallback(() => {
    console.log('Flow initialized');
  }, []);

  return (
    <div className="h-[400px] bg-slate-900 rounded-lg border border-slate-700">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onInit={onInit}
        fitView
        attributionPosition="bottom-left"
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#334155" gap={20} />
        <Controls className="!bg-slate-800 !border-slate-700" />
      </ReactFlow>
    </div>
  );
}
