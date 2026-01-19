# ADR 005: Choix de React Flow pour la visualisation

## Statut

Accepté

## Contexte

L'interface web doit visualiser les flux d'événements entre services en temps réel. Options considérées:

- React Flow (@xyflow/react)
- D3.js
- vis.js
- Cytoscape.js
- GoJS

## Décision

Nous avons choisi **React Flow** (@xyflow/react).

## Justification

### Avantages

1. **Intégration React**: Composants React natifs
2. **Performance**: Rendu optimisé avec virtualisation
3. **Interactivité**: Zoom, pan, drag-and-drop natifs
4. **Personnalisation**: Nodes et edges personnalisables
5. **Animations**: Support des animations CSS/JS
6. **Documentation**: Excellente documentation et exemples
7. **Communauté**: Activement maintenu, large adoption

### Inconvénients

1. **Courbe d'apprentissage**: Concepts spécifiques (handles, edges)
2. **Bundle size**: ~100KB gzipped
3. **React only**: Pas utilisable avec Vue/Angular

### Alternatives rejetées

| Alternative | Raison du rejet |
|-------------|-----------------|
| D3.js | Trop bas niveau, complexité excessive |
| vis.js | Moins bien intégré à React |
| Cytoscape.js | Plus orienté graphes scientifiques |
| GoJS | Licence commerciale |

## Conséquences

- Composants custom pour les nodes (ServiceNode)
- Edges animés pour visualiser le flux
- État géré via Zustand
- WebSocket pour les mises à jour temps réel

## Architecture des composants

```
web-ui/src/components/
├── FlowVisualization.tsx    # Conteneur principal React Flow
├── ServiceNode.tsx          # Node personnalisé pour un service
├── SimulationControls.tsx   # Contrôles start/stop/rate
└── MetricsDashboard.tsx     # Affichage des métriques
```

## Exemple de node personnalisé

```tsx
function ServiceNode({ data }: NodeProps<ServiceNodeData>) {
  return (
    <div className={`service-node ${data.status}`}>
      <Handle type="target" position={Position.Left} />
      <div className="icon">{data.icon}</div>
      <div className="label">{data.label}</div>
      <div className="metrics">{data.eventsCount} events</div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
```

## Intégration WebSocket

```typescript
// Mise à jour du flow lors de réception d'événements
useEffect(() => {
  socket.on('event', (event) => {
    // Animer l'edge correspondant
    setEdges((edges) =>
      edges.map((edge) =>
        edge.id === `${event.source}-${event.target}`
          ? { ...edge, animated: true }
          : edge
      )
    );
  });
}, [socket]);
```

## Dépendances

```json
{
  "@xyflow/react": "^12.x",
  "zustand": "^4.x",
  "lucide-react": "^0.x"
}
```
