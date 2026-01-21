"""
Outbox Pattern pour publication fiable d'événements.

Garantit l'atomicité entre la modification de la base de données
et la publication d'événements.
"""
import asyncio
from datetime import datetime
from typing import Any, Callable, Dict, List, Optional
from dataclasses import dataclass, field
from enum import Enum
import uuid
import json


class OutboxStatus(Enum):
    """Statuts possibles d'une entrée outbox."""
    PENDING = "pending"
    PROCESSING = "processing"
    PUBLISHED = "published"
    FAILED = "failed"


@dataclass
class OutboxEntry:
    """Entrée dans la table outbox."""
    id: str
    aggregate_type: str
    aggregate_id: str
    event_type: str
    payload: Dict[str, Any]
    status: OutboxStatus = OutboxStatus.PENDING
    created_at: str = field(default_factory=lambda: datetime.now().isoformat())
    published_at: Optional[str] = None
    retries: int = 0
    last_error: Optional[str] = None

    def to_dict(self) -> Dict:
        return {
            "id": self.id,
            "aggregate_type": self.aggregate_type,
            "aggregate_id": self.aggregate_id,
            "event_type": self.event_type,
            "payload": self.payload,
            "status": self.status.value,
            "created_at": self.created_at,
            "published_at": self.published_at,
            "retries": self.retries,
            "last_error": self.last_error
        }


class OutboxProcessor:
    """
    Processeur Outbox pour publication fiable.

    Fonctionnalités:
    - Stockage atomique avec la transaction métier
    - Polling pour publication
    - Retry avec backoff
    - Gestion des échecs
    """

    def __init__(self, publisher: Callable = None):
        """
        Args:
            publisher: Fonction pour publier les messages (async)
        """
        self._entries: Dict[str, OutboxEntry] = {}
        self._publisher = publisher
        self._is_running = False
        self._poll_interval = 1.0  # secondes
        self._max_retries = 5
        self._event_handlers: List[Callable] = []

    def _generate_id(self) -> str:
        """Génère un ID unique."""
        return f"OBX-{uuid.uuid4().hex[:12].upper()}"

    async def add_entry(
        self,
        aggregate_type: str,
        aggregate_id: str,
        event_type: str,
        payload: Dict[str, Any]
    ) -> OutboxEntry:
        """
        Ajoute une entrée dans l'outbox.

        Cette méthode doit être appelée DANS la même transaction
        que la modification de la base de données métier.

        Args:
            aggregate_type: Type de l'agrégat (ex: "Policy")
            aggregate_id: ID de l'agrégat
            event_type: Type d'événement (ex: "PolicyCreated")
            payload: Données de l'événement

        Returns:
            L'entrée créée
        """
        entry = OutboxEntry(
            id=self._generate_id(),
            aggregate_type=aggregate_type,
            aggregate_id=aggregate_id,
            event_type=event_type,
            payload=payload
        )

        self._entries[entry.id] = entry

        await self._notify_event("outbox_entry_added", {
            "entry_id": entry.id,
            "event_type": event_type
        })

        return entry

    async def process_pending(self) -> int:
        """
        Traite les entrées en attente.

        Returns:
            Nombre d'entrées traitées
        """
        pending = [e for e in self._entries.values()
                   if e.status == OutboxStatus.PENDING]

        processed = 0

        for entry in pending:
            try:
                entry.status = OutboxStatus.PROCESSING

                await self._notify_event("outbox_processing", {
                    "entry_id": entry.id
                })

                # Publier le message
                await self._publish_entry(entry)

                # Marquer comme publié
                entry.status = OutboxStatus.PUBLISHED
                entry.published_at = datetime.now().isoformat()
                processed += 1

                await self._notify_event("outbox_published", {
                    "entry_id": entry.id,
                    "event_type": entry.event_type
                })

            except Exception as e:
                entry.retries += 1
                entry.last_error = str(e)

                if entry.retries >= self._max_retries:
                    entry.status = OutboxStatus.FAILED
                    await self._notify_event("outbox_failed", {
                        "entry_id": entry.id,
                        "error": str(e),
                        "retries": entry.retries
                    })
                else:
                    entry.status = OutboxStatus.PENDING  # Pour retry
                    await self._notify_event("outbox_retry_scheduled", {
                        "entry_id": entry.id,
                        "retry": entry.retries
                    })

        return processed

    async def _publish_entry(self, entry: OutboxEntry):
        """Publie une entrée via le publisher configuré."""
        if self._publisher:
            if asyncio.iscoroutinefunction(self._publisher):
                await self._publisher({
                    "type": entry.event_type,
                    "aggregate_type": entry.aggregate_type,
                    "aggregate_id": entry.aggregate_id,
                    "payload": entry.payload,
                    "timestamp": entry.created_at
                })
            else:
                self._publisher({
                    "type": entry.event_type,
                    "aggregate_type": entry.aggregate_type,
                    "aggregate_id": entry.aggregate_id,
                    "payload": entry.payload,
                    "timestamp": entry.created_at
                })
        else:
            # Simuler la publication
            await asyncio.sleep(0.01)

    async def start_polling(self):
        """Démarre le polling pour traiter les entrées en attente."""
        self._is_running = True

        while self._is_running:
            try:
                await self.process_pending()
            except Exception as e:
                print(f"Outbox polling error: {e}")

            await asyncio.sleep(self._poll_interval)

    def stop_polling(self):
        """Arrête le polling."""
        self._is_running = False

    # ========== GESTION DES ENTRÉES ==========

    def get_entry(self, entry_id: str) -> Optional[OutboxEntry]:
        """Récupère une entrée par son ID."""
        return self._entries.get(entry_id)

    def get_pending_entries(self) -> List[OutboxEntry]:
        """Retourne les entrées en attente."""
        return [e for e in self._entries.values()
                if e.status == OutboxStatus.PENDING]

    def get_failed_entries(self) -> List[OutboxEntry]:
        """Retourne les entrées en échec."""
        return [e for e in self._entries.values()
                if e.status == OutboxStatus.FAILED]

    def get_all_entries(self) -> List[Dict]:
        """Retourne toutes les entrées."""
        return [e.to_dict() for e in self._entries.values()]

    async def retry_failed(self, entry_id: str) -> bool:
        """
        Remet une entrée en échec en attente pour retry.

        Args:
            entry_id: ID de l'entrée

        Returns:
            True si réussi
        """
        entry = self._entries.get(entry_id)
        if entry and entry.status == OutboxStatus.FAILED:
            entry.status = OutboxStatus.PENDING
            entry.retries = 0
            entry.last_error = None
            return True
        return False

    async def purge_published(self, older_than_hours: int = 24):
        """
        Supprime les entrées publiées anciennes.

        Args:
            older_than_hours: Supprimer si plus vieux que N heures
        """
        cutoff = datetime.now().timestamp() - (older_than_hours * 3600)
        to_delete = []

        for entry_id, entry in self._entries.items():
            if entry.status == OutboxStatus.PUBLISHED:
                if entry.published_at:
                    published_ts = datetime.fromisoformat(entry.published_at).timestamp()
                    if published_ts < cutoff:
                        to_delete.append(entry_id)

        for entry_id in to_delete:
            del self._entries[entry_id]

        return len(to_delete)

    # ========== STATISTIQUES ==========

    def get_stats(self) -> Dict:
        """Retourne les statistiques de l'outbox."""
        stats = {
            "total": len(self._entries),
            "pending": 0,
            "processing": 0,
            "published": 0,
            "failed": 0
        }

        for entry in self._entries.values():
            stats[entry.status.value] += 1

        return stats

    # ========== ÉVÉNEMENTS ==========

    def on_event(self, handler: Callable):
        """Enregistre un handler pour les événements."""
        self._event_handlers.append(handler)

    async def _notify_event(self, event_type: str, data: Dict):
        """Notifie les handlers d'un événement."""
        for handler in self._event_handlers:
            try:
                if asyncio.iscoroutinefunction(handler):
                    await handler({"type": event_type, "data": data})
                else:
                    handler({"type": event_type, "data": data})
            except Exception:
                pass

    def reset(self):
        """Réinitialise l'outbox."""
        self._entries.clear()
        self._is_running = False


# ========== EXEMPLE D'UTILISATION ATOMIQUE ==========

class AtomicBusinessOperation:
    """
    Exemple d'opération métier avec outbox atomique.

    Simule une transaction qui modifie la base ET ajoute à l'outbox.
    """

    def __init__(self, outbox: OutboxProcessor):
        self.outbox = outbox
        self._db = {}  # Simule la base de données

    async def create_policy(self, customer_id: str, product: str, premium: float) -> Dict:
        """
        Crée une police avec publication d'événement garantie.

        Simulation d'une transaction atomique:
        - INSERT dans la table policies
        - INSERT dans la table outbox
        - COMMIT (les deux ensemble)
        """
        policy_id = f"POL-{uuid.uuid4().hex[:8].upper()}"

        # Début de "transaction"
        try:
            # 1. Sauvegarder en base
            policy = {
                "id": policy_id,
                "customer_id": customer_id,
                "product": product,
                "premium": premium,
                "status": "ACTIVE",
                "created_at": datetime.now().isoformat()
            }
            self._db[policy_id] = policy

            # 2. Ajouter à l'outbox (DANS LA MÊME TRANSACTION)
            await self.outbox.add_entry(
                aggregate_type="Policy",
                aggregate_id=policy_id,
                event_type="PolicyCreated",
                payload={
                    "policy_id": policy_id,
                    "customer_id": customer_id,
                    "product": product,
                    "premium": premium
                }
            )

            # 3. Commit (atomique)
            return {"success": True, "policy": policy}

        except Exception as e:
            # Rollback
            if policy_id in self._db:
                del self._db[policy_id]
            raise e

    def get_policy(self, policy_id: str) -> Optional[Dict]:
        """Récupère une police."""
        return self._db.get(policy_id)


# Instance singleton
_outbox_instance: Optional[OutboxProcessor] = None


def get_outbox() -> OutboxProcessor:
    """Retourne l'instance singleton de l'outbox."""
    global _outbox_instance
    if _outbox_instance is None:
        _outbox_instance = OutboxProcessor()
    return _outbox_instance


def reset_outbox():
    """Réinitialise l'outbox."""
    global _outbox_instance
    if _outbox_instance:
        _outbox_instance.reset()
    _outbox_instance = OutboxProcessor()
