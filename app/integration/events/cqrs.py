"""
CQRS (Command Query Responsibility Segregation).

Sépare les opérations d'écriture (commandes) des opérations de lecture (requêtes).
Implémentation pédagogique avec projections.
"""
import asyncio
from datetime import datetime
from typing import Any, Callable, Dict, List, Optional, Type
from dataclasses import dataclass, field
from abc import ABC, abstractmethod
import uuid

from app.integration.events.event_store import EventStore, Event, get_event_store


# ========== COMMANDES ==========

@dataclass
class Command(ABC):
    """Classe de base pour les commandes."""
    id: str = field(default_factory=lambda: f"CMD-{uuid.uuid4().hex[:8].upper()}")
    timestamp: str = field(default_factory=lambda: datetime.now().isoformat())
    correlation_id: Optional[str] = None


@dataclass
class CreatePolicyCommand(Command):
    """Commande pour créer une police."""
    customer_id: str = ""
    product: str = ""
    premium: float = 0.0
    coverages: List[str] = field(default_factory=list)


@dataclass
class ActivatePolicyCommand(Command):
    """Commande pour activer une police."""
    policy_id: str = ""
    start_date: str = ""
    end_date: str = ""


@dataclass
class ModifyPolicyCommand(Command):
    """Commande pour modifier une police."""
    policy_id: str = ""
    modifications: Dict[str, Any] = field(default_factory=dict)


@dataclass
class CancelPolicyCommand(Command):
    """Commande pour annuler une police."""
    policy_id: str = ""
    reason: str = ""


# ========== COMMAND HANDLERS ==========

class CommandHandler(ABC):
    """Interface pour les handlers de commandes."""

    @abstractmethod
    async def handle(self, command: Command) -> Dict:
        """Traite une commande et retourne le résultat."""
        pass


class PolicyCommandHandler(CommandHandler):
    """Handler pour les commandes liées aux polices."""

    def __init__(self, event_store: EventStore = None):
        self.event_store = event_store or get_event_store()

    async def handle(self, command: Command) -> Dict:
        """Dispatch la commande vers le bon handler."""
        if isinstance(command, CreatePolicyCommand):
            return await self._handle_create(command)
        elif isinstance(command, ActivatePolicyCommand):
            return await self._handle_activate(command)
        elif isinstance(command, ModifyPolicyCommand):
            return await self._handle_modify(command)
        elif isinstance(command, CancelPolicyCommand):
            return await self._handle_cancel(command)
        else:
            raise ValueError(f"Unknown command type: {type(command)}")

    async def _handle_create(self, cmd: CreatePolicyCommand) -> Dict:
        """Crée une nouvelle police."""
        policy_id = f"POL-{uuid.uuid4().hex[:8].upper()}"

        event = await self.event_store.append(
            aggregate_id=policy_id,
            event_data={
                "type": "PolicyCreated",
                "data": {
                    "policy_number": policy_id,
                    "customer_id": cmd.customer_id,
                    "product": cmd.product,
                    "premium": cmd.premium,
                    "coverages": cmd.coverages
                },
                "metadata": {
                    "command_id": cmd.id,
                    "correlation_id": cmd.correlation_id
                }
            }
        )

        return {
            "success": True,
            "policy_id": policy_id,
            "event_id": event.id,
            "version": event.version
        }

    async def _handle_activate(self, cmd: ActivatePolicyCommand) -> Dict:
        """Active une police."""
        event = await self.event_store.append(
            aggregate_id=cmd.policy_id,
            event_data={
                "type": "PolicyActivated",
                "data": {
                    "start_date": cmd.start_date,
                    "end_date": cmd.end_date
                },
                "metadata": {
                    "command_id": cmd.id,
                    "correlation_id": cmd.correlation_id
                }
            }
        )

        return {
            "success": True,
            "policy_id": cmd.policy_id,
            "event_id": event.id,
            "version": event.version
        }

    async def _handle_modify(self, cmd: ModifyPolicyCommand) -> Dict:
        """Modifie une police."""
        event = await self.event_store.append(
            aggregate_id=cmd.policy_id,
            event_data={
                "type": "PolicyModified",
                "data": cmd.modifications,
                "metadata": {
                    "command_id": cmd.id,
                    "correlation_id": cmd.correlation_id
                }
            }
        )

        return {
            "success": True,
            "policy_id": cmd.policy_id,
            "event_id": event.id,
            "version": event.version
        }

    async def _handle_cancel(self, cmd: CancelPolicyCommand) -> Dict:
        """Annule une police."""
        event = await self.event_store.append(
            aggregate_id=cmd.policy_id,
            event_data={
                "type": "PolicyCancelled",
                "data": {
                    "reason": cmd.reason
                },
                "metadata": {
                    "command_id": cmd.id,
                    "correlation_id": cmd.correlation_id
                }
            }
        )

        return {
            "success": True,
            "policy_id": cmd.policy_id,
            "event_id": event.id,
            "version": event.version
        }


# ========== REQUÊTES ==========

@dataclass
class Query(ABC):
    """Classe de base pour les requêtes."""
    id: str = field(default_factory=lambda: f"QRY-{uuid.uuid4().hex[:8].upper()}")


@dataclass
class GetPolicyQuery(Query):
    """Requête pour obtenir une police."""
    policy_id: str = ""


@dataclass
class ListPoliciesByCustomerQuery(Query):
    """Requête pour lister les polices d'un client."""
    customer_id: str = ""


@dataclass
class GetPolicySummaryQuery(Query):
    """Requête pour obtenir un résumé des polices."""
    pass


# ========== PROJECTIONS (READ MODELS) ==========

class Projection(ABC):
    """Interface pour les projections (modèles de lecture)."""

    @abstractmethod
    async def apply(self, event: Event):
        """Applique un événement à la projection."""
        pass

    @abstractmethod
    def get_state(self) -> Dict:
        """Retourne l'état actuel de la projection."""
        pass


class PolicyListProjection(Projection):
    """Projection listant toutes les polices."""

    def __init__(self):
        self._policies: Dict[str, Dict] = {}

    async def apply(self, event: Event):
        """Met à jour la liste des polices."""
        policy_id = event.aggregate_id

        if event.type == "PolicyCreated":
            self._policies[policy_id] = {
                "policy_number": policy_id,
                "customer_id": event.data.get("customer_id"),
                "product": event.data.get("product"),
                "premium": event.data.get("premium"),
                "status": "DRAFT",
                "created_at": event.timestamp
            }

        elif event.type == "PolicyActivated":
            if policy_id in self._policies:
                self._policies[policy_id]["status"] = "ACTIVE"
                self._policies[policy_id]["start_date"] = event.data.get("start_date")
                self._policies[policy_id]["end_date"] = event.data.get("end_date")

        elif event.type == "PolicyCancelled":
            if policy_id in self._policies:
                self._policies[policy_id]["status"] = "CANCELLED"
                self._policies[policy_id]["cancelled_at"] = event.timestamp

        elif event.type == "PolicyModified":
            if policy_id in self._policies:
                for key, value in event.data.items():
                    self._policies[policy_id][key] = value

    def get_state(self) -> Dict:
        return {"policies": list(self._policies.values())}

    def get_policy(self, policy_id: str) -> Optional[Dict]:
        return self._policies.get(policy_id)

    def get_by_customer(self, customer_id: str) -> List[Dict]:
        return [
            p for p in self._policies.values()
            if p.get("customer_id") == customer_id
        ]


class PolicySummaryProjection(Projection):
    """Projection pour les statistiques des polices."""

    def __init__(self):
        self._stats = {
            "total": 0,
            "by_status": {},
            "by_product": {},
            "total_premium": 0.0
        }

    async def apply(self, event: Event):
        """Met à jour les statistiques."""
        if event.type == "PolicyCreated":
            self._stats["total"] += 1

            # Par statut
            self._stats["by_status"]["DRAFT"] = \
                self._stats["by_status"].get("DRAFT", 0) + 1

            # Par produit
            product = event.data.get("product", "UNKNOWN")
            self._stats["by_product"][product] = \
                self._stats["by_product"].get(product, 0) + 1

            # Premium total
            self._stats["total_premium"] += event.data.get("premium", 0)

        elif event.type == "PolicyActivated":
            self._stats["by_status"]["DRAFT"] = \
                max(0, self._stats["by_status"].get("DRAFT", 0) - 1)
            self._stats["by_status"]["ACTIVE"] = \
                self._stats["by_status"].get("ACTIVE", 0) + 1

        elif event.type == "PolicyCancelled":
            self._stats["by_status"]["ACTIVE"] = \
                max(0, self._stats["by_status"].get("ACTIVE", 0) - 1)
            self._stats["by_status"]["CANCELLED"] = \
                self._stats["by_status"].get("CANCELLED", 0) + 1

    def get_state(self) -> Dict:
        return self._stats


# ========== QUERY HANDLER ==========

class QueryHandler:
    """Handler pour les requêtes (lectures)."""

    def __init__(self):
        self.policy_list = PolicyListProjection()
        self.policy_summary = PolicySummaryProjection()

        # Enregistrer les projections auprès de l'event store
        event_store = get_event_store()
        event_store.register_projection(self.policy_list.apply)
        event_store.register_projection(self.policy_summary.apply)

    async def handle(self, query: Query) -> Dict:
        """Dispatch la requête vers le bon handler."""
        if isinstance(query, GetPolicyQuery):
            return self._handle_get_policy(query)
        elif isinstance(query, ListPoliciesByCustomerQuery):
            return self._handle_list_by_customer(query)
        elif isinstance(query, GetPolicySummaryQuery):
            return self._handle_summary(query)
        else:
            raise ValueError(f"Unknown query type: {type(query)}")

    def _handle_get_policy(self, query: GetPolicyQuery) -> Dict:
        """Retourne une police."""
        policy = self.policy_list.get_policy(query.policy_id)
        return {
            "found": policy is not None,
            "policy": policy
        }

    def _handle_list_by_customer(self, query: ListPoliciesByCustomerQuery) -> Dict:
        """Liste les polices d'un client."""
        policies = self.policy_list.get_by_customer(query.customer_id)
        return {
            "customer_id": query.customer_id,
            "count": len(policies),
            "policies": policies
        }

    def _handle_summary(self, query: GetPolicySummaryQuery) -> Dict:
        """Retourne le résumé des polices."""
        return self.policy_summary.get_state()


# ========== CQRS BUS ==========

class CQRSBus:
    """
    Bus central pour CQRS.

    Coordonne les commandes et requêtes.
    """

    def __init__(self):
        self.command_handler = PolicyCommandHandler()
        self.query_handler = QueryHandler()

    async def send_command(self, command: Command) -> Dict:
        """Envoie une commande."""
        return await self.command_handler.handle(command)

    async def execute_query(self, query: Query) -> Dict:
        """Exécute une requête."""
        return await self.query_handler.handle(query)


# Instance singleton
_cqrs_bus: Optional[CQRSBus] = None


def get_cqrs_bus() -> CQRSBus:
    """Retourne l'instance singleton du bus CQRS."""
    global _cqrs_bus
    if _cqrs_bus is None:
        _cqrs_bus = CQRSBus()
    return _cqrs_bus
