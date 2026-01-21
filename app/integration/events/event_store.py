"""
Event Store pour Event Sourcing.

Permet de stocker et rejouer des événements pour reconstruire l'état.
Implémentation in-memory pour simulation pédagogique.
"""
import asyncio
from datetime import datetime
from typing import Any, Dict, List, Optional, Callable
from dataclasses import dataclass, field
import uuid


@dataclass
class Event:
    """Représente un événement dans le store."""
    id: str
    aggregate_id: str
    type: str
    data: Dict[str, Any]
    timestamp: str = field(default_factory=lambda: datetime.now().isoformat())
    version: int = 0
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict:
        """Convertit l'événement en dictionnaire."""
        return {
            "id": self.id,
            "aggregate_id": self.aggregate_id,
            "type": self.type,
            "data": self.data,
            "timestamp": self.timestamp,
            "version": self.version,
            "metadata": self.metadata
        }


class EventStore:
    """
    Store d'événements append-only.

    Fonctionnalités:
    - Append d'événements
    - Lecture par aggregate
    - Reconstruction d'état (replay)
    - Snapshots pour optimisation
    """

    def __init__(self):
        # Stockage des événements par aggregate_id
        self._events: Dict[str, List[Event]] = {}
        # Snapshots (optionnel, pour optimisation)
        self._snapshots: Dict[str, Dict] = {}
        # Version courante par aggregate
        self._versions: Dict[str, int] = {}
        # Handlers de projection
        self._projections: List[Callable] = []
        # Historique global pour monitoring
        self._global_stream: List[Event] = []

    def _generate_event_id(self) -> str:
        """Génère un ID unique pour un événement."""
        return f"EVT-{uuid.uuid4().hex[:12].upper()}"

    async def append(
        self,
        aggregate_id: str,
        event_data: Dict[str, Any],
        expected_version: Optional[int] = None
    ) -> Event:
        """
        Ajoute un événement au store.

        Args:
            aggregate_id: ID de l'agrégat
            event_data: Données de l'événement (doit contenir "type" et "data")
            expected_version: Version attendue pour optimistic locking

        Returns:
            L'événement créé

        Raises:
            ConcurrencyError: Si la version ne correspond pas
        """
        # Initialiser la liste si nécessaire
        if aggregate_id not in self._events:
            self._events[aggregate_id] = []
            self._versions[aggregate_id] = 0

        current_version = self._versions[aggregate_id]

        # Vérification de concurrence optimiste
        if expected_version is not None and expected_version != current_version:
            raise ConcurrencyError(
                f"Expected version {expected_version}, but current is {current_version}"
            )

        # Créer l'événement
        new_version = current_version + 1
        event = Event(
            id=self._generate_event_id(),
            aggregate_id=aggregate_id,
            type=event_data.get("type", "Unknown"),
            data=event_data.get("data", event_data),
            version=new_version,
            metadata=event_data.get("metadata", {})
        )

        # Stocker
        self._events[aggregate_id].append(event)
        self._versions[aggregate_id] = new_version
        self._global_stream.append(event)

        # Notifier les projections
        await self._notify_projections(event)

        return event

    async def get_events(
        self,
        aggregate_id: str,
        from_version: int = 0,
        to_version: Optional[int] = None
    ) -> List[Event]:
        """
        Récupère les événements d'un agrégat.

        Args:
            aggregate_id: ID de l'agrégat
            from_version: Version de départ (inclusive)
            to_version: Version de fin (inclusive, None = toutes)

        Returns:
            Liste des événements
        """
        if aggregate_id not in self._events:
            return []

        events = self._events[aggregate_id]

        # Filtrer par version
        filtered = [e for e in events if e.version >= from_version]
        if to_version is not None:
            filtered = [e for e in filtered if e.version <= to_version]

        return filtered

    async def rebuild_state(
        self,
        aggregate_id: str,
        reducer: Optional[Callable] = None,
        to_version: Optional[int] = None
    ) -> Dict[str, Any]:
        """
        Reconstruit l'état d'un agrégat en rejouant les événements.

        Args:
            aggregate_id: ID de l'agrégat
            reducer: Fonction (state, event) -> new_state
            to_version: Reconstruire jusqu'à cette version

        Returns:
            L'état reconstruit
        """
        events = await self.get_events(aggregate_id, to_version=to_version)

        # Utiliser le reducer par défaut si non fourni
        if reducer is None:
            reducer = self._default_reducer

        # Partir du snapshot si disponible
        state = self._snapshots.get(aggregate_id, {})

        # Appliquer chaque événement
        for event in events:
            state = reducer(state, event)

        return state

    def _default_reducer(self, state: Dict, event: Event) -> Dict:
        """Reducer par défaut qui merge les données."""
        new_state = state.copy()

        # Appliquer les données de l'événement
        if isinstance(event.data, dict):
            new_state.update(event.data)

        # Mettre à jour les métadonnées
        new_state["_version"] = event.version
        new_state["_last_updated"] = event.timestamp
        new_state["_last_event_type"] = event.type

        return new_state

    async def create_snapshot(self, aggregate_id: str, state: Dict):
        """
        Crée un snapshot de l'état actuel.

        Args:
            aggregate_id: ID de l'agrégat
            state: État à sauvegarder
        """
        self._snapshots[aggregate_id] = {
            **state,
            "_snapshot_version": self._versions.get(aggregate_id, 0),
            "_snapshot_at": datetime.now().isoformat()
        }

    async def get_snapshot(self, aggregate_id: str) -> Optional[Dict]:
        """Récupère le snapshot d'un agrégat."""
        return self._snapshots.get(aggregate_id)

    def get_current_version(self, aggregate_id: str) -> int:
        """Retourne la version courante d'un agrégat."""
        return self._versions.get(aggregate_id, 0)

    # ========== PROJECTIONS ==========

    def register_projection(self, handler: Callable):
        """
        Enregistre un handler de projection.

        Le handler sera appelé pour chaque nouvel événement.
        """
        self._projections.append(handler)

    async def _notify_projections(self, event: Event):
        """Notifie toutes les projections d'un nouvel événement."""
        for handler in self._projections:
            try:
                if asyncio.iscoroutinefunction(handler):
                    await handler(event)
                else:
                    handler(event)
            except Exception as e:
                # Log l'erreur mais continue
                print(f"Projection error: {e}")

    # ========== GLOBAL STREAM ==========

    def get_global_stream(self, limit: int = 100) -> List[Dict]:
        """Retourne les derniers événements globaux."""
        return [e.to_dict() for e in self._global_stream[-limit:]]

    def get_events_by_type(self, event_type: str, limit: int = 100) -> List[Dict]:
        """Retourne les événements d'un certain type."""
        filtered = [e for e in self._global_stream if e.type == event_type]
        return [e.to_dict() for e in filtered[-limit:]]

    # ========== UTILITAIRES ==========

    def get_stats(self) -> Dict:
        """Retourne les statistiques du store."""
        return {
            "total_events": len(self._global_stream),
            "aggregates_count": len(self._events),
            "snapshots_count": len(self._snapshots),
            "event_types": list(set(e.type for e in self._global_stream))
        }

    def reset(self):
        """Réinitialise le store."""
        self._events.clear()
        self._snapshots.clear()
        self._versions.clear()
        self._global_stream.clear()


class ConcurrencyError(Exception):
    """Erreur de concurrence lors de l'ajout d'événement."""
    pass


# ========== POLICY-SPECIFIC REDUCERS ==========

def policy_reducer(state: Dict, event: Event) -> Dict:
    """
    Reducer spécifique pour les polices d'assurance.

    Gère les événements:
    - PolicyCreated
    - PolicyActivated
    - PolicyModified
    - PolicyCancelled
    - PolicyRenewed
    """
    new_state = state.copy()

    event_type = event.type
    data = event.data

    if event_type == "PolicyCreated":
        new_state = {
            "policy_number": data.get("policy_number"),
            "customer_id": data.get("customer_id"),
            "product": data.get("product"),
            "premium": data.get("premium"),
            "status": "DRAFT",
            "created_at": event.timestamp,
            "history": []
        }

    elif event_type == "PolicyActivated":
        new_state["status"] = "ACTIVE"
        new_state["activated_at"] = event.timestamp
        new_state["start_date"] = data.get("start_date")
        new_state["end_date"] = data.get("end_date")

    elif event_type == "PolicyModified":
        # Merge les modifications
        for key, value in data.items():
            if key not in ["type", "timestamp"]:
                new_state[key] = value

    elif event_type == "PolicyCancelled":
        new_state["status"] = "CANCELLED"
        new_state["cancelled_at"] = event.timestamp
        new_state["cancellation_reason"] = data.get("reason")

    elif event_type == "PolicyRenewed":
        new_state["status"] = "ACTIVE"
        new_state["renewed_at"] = event.timestamp
        new_state["start_date"] = data.get("new_start_date")
        new_state["end_date"] = data.get("new_end_date")
        new_state["premium"] = data.get("new_premium", new_state.get("premium"))

    # Ajouter à l'historique
    if "history" not in new_state:
        new_state["history"] = []
    new_state["history"].append({
        "event": event_type,
        "timestamp": event.timestamp,
        "version": event.version
    })

    new_state["_version"] = event.version
    new_state["_last_updated"] = event.timestamp

    return new_state


# Instance singleton
_event_store_instance: Optional[EventStore] = None


def get_event_store() -> EventStore:
    """Retourne l'instance singleton de l'Event Store."""
    global _event_store_instance
    if _event_store_instance is None:
        _event_store_instance = EventStore()
    return _event_store_instance


def reset_event_store():
    """Réinitialise l'Event Store."""
    global _event_store_instance
    if _event_store_instance:
        _event_store_instance.reset()
    _event_store_instance = EventStore()
