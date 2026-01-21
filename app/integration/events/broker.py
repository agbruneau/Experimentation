"""
Message Broker In-Memory pour simulation d'intégration par événements.

Fonctionnalités:
- Queue Point-à-Point (un seul consommateur)
- Topic Pub/Sub (multiple consommateurs)
- Garantie at-least-once
- Dead Letter Queue (DLQ)
- Métriques et monitoring
"""
import asyncio
import uuid
from datetime import datetime
from typing import Any, Callable, Dict, List, Optional, Set
from enum import Enum
from dataclasses import dataclass, field


class MessageStatus(Enum):
    """Statuts possibles d'un message."""
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"
    DEAD_LETTER = "dead_letter"


@dataclass
class Message:
    """Représente un message dans le broker."""
    id: str
    payload: Dict[str, Any]
    source: str = ""
    timestamp: str = field(default_factory=lambda: datetime.now().isoformat())
    status: MessageStatus = MessageStatus.PENDING
    retries: int = 0
    max_retries: int = 3
    error: Optional[str] = None
    headers: Dict[str, str] = field(default_factory=dict)

    def to_dict(self) -> Dict:
        """Convertit le message en dictionnaire."""
        return {
            "id": self.id,
            "payload": self.payload,
            "source": self.source,
            "timestamp": self.timestamp,
            "status": self.status.value,
            "retries": self.retries,
            "max_retries": self.max_retries,
            "error": self.error,
            "headers": self.headers
        }


@dataclass
class Subscription:
    """Représente un abonnement à un topic."""
    id: str
    handler: Callable
    max_retries: int = 3
    active: bool = True


class MessageBroker:
    """
    Broker de messages in-memory simulant un système de messaging.

    Supporte:
    - Queues (point-à-point)
    - Topics (pub/sub)
    - Dead Letter Queues
    - Métriques
    """

    def __init__(self):
        # Queues point-à-point
        self._queues: Dict[str, asyncio.Queue] = {}
        self._queue_messages: Dict[str, List[Message]] = {}

        # Topics pub/sub
        self._topics: Dict[str, List[Subscription]] = {}

        # Dead Letter Queues
        self._dlq: Dict[str, List[Message]] = {}

        # Métriques
        self._stats = {
            "messages_sent": 0,
            "messages_received": 0,
            "messages_failed": 0,
            "messages_dlq": 0
        }

        # Event handlers pour notification externe (SSE)
        self._event_handlers: List[Callable] = []

        # Historique des messages pour replay
        self._message_history: List[Message] = []

    def _generate_message_id(self) -> str:
        """Génère un ID unique pour un message."""
        return f"MSG-{uuid.uuid4().hex[:12].upper()}"

    async def _notify_event(self, event_type: str, data: Dict):
        """Notifie les handlers externes d'un événement."""
        event = {
            "type": event_type,
            "data": data,
            "timestamp": datetime.now().isoformat()
        }
        for handler in self._event_handlers:
            try:
                if asyncio.iscoroutinefunction(handler):
                    await handler(event)
                else:
                    handler(event)
            except Exception:
                pass  # Ignore les erreurs des handlers

    def on_event(self, handler: Callable):
        """Enregistre un handler pour les événements du broker."""
        self._event_handlers.append(handler)

    # ========== QUEUE (Point-à-Point) ==========

    def _ensure_queue(self, queue_name: str):
        """S'assure qu'une queue existe."""
        if queue_name not in self._queues:
            self._queues[queue_name] = asyncio.Queue()
            self._queue_messages[queue_name] = []

    async def send_to_queue(
        self,
        queue_name: str,
        payload: Dict[str, Any],
        source: str = "",
        headers: Dict[str, str] = None
    ) -> Message:
        """
        Envoie un message dans une queue.

        Args:
            queue_name: Nom de la queue
            payload: Contenu du message
            source: Service source
            headers: Headers optionnels

        Returns:
            Le message créé
        """
        self._ensure_queue(queue_name)

        message = Message(
            id=self._generate_message_id(),
            payload=payload,
            source=source,
            headers=headers or {}
        )

        await self._queues[queue_name].put(message)
        self._queue_messages[queue_name].append(message)
        self._message_history.append(message)
        self._stats["messages_sent"] += 1

        await self._notify_event("queue_message", {
            "queue": queue_name,
            "message": message.to_dict()
        })

        return message

    async def receive_from_queue(
        self,
        queue_name: str,
        timeout: float = 5.0
    ) -> Optional[Message]:
        """
        Reçoit un message d'une queue.

        Args:
            queue_name: Nom de la queue
            timeout: Délai d'attente en secondes

        Returns:
            Le message reçu ou None si timeout
        """
        self._ensure_queue(queue_name)

        try:
            message = await asyncio.wait_for(
                self._queues[queue_name].get(),
                timeout=timeout
            )
            message.status = MessageStatus.PROCESSING
            self._stats["messages_received"] += 1

            await self._notify_event("queue_receive", {
                "queue": queue_name,
                "message": message.to_dict()
            })

            return message
        except asyncio.TimeoutError:
            return None

    def get_queue_size(self, queue_name: str) -> int:
        """Retourne la taille d'une queue."""
        if queue_name not in self._queues:
            return 0
        return self._queues[queue_name].qsize()

    def get_queue_messages(self, queue_name: str, limit: int = 50) -> List[Dict]:
        """Retourne les messages d'une queue."""
        if queue_name not in self._queue_messages:
            return []
        return [m.to_dict() for m in self._queue_messages[queue_name][-limit:]]

    # ========== TOPIC (Pub/Sub) ==========

    def _ensure_topic(self, topic_name: str):
        """S'assure qu'un topic existe."""
        if topic_name not in self._topics:
            self._topics[topic_name] = []

    async def subscribe(
        self,
        topic_name: str,
        handler: Callable,
        max_retries: int = 3
    ) -> str:
        """
        S'abonne à un topic.

        Args:
            topic_name: Nom du topic
            handler: Fonction appelée pour chaque message
            max_retries: Nombre max de tentatives

        Returns:
            ID de l'abonnement
        """
        self._ensure_topic(topic_name)

        subscription = Subscription(
            id=f"SUB-{uuid.uuid4().hex[:8].upper()}",
            handler=handler,
            max_retries=max_retries
        )

        self._topics[topic_name].append(subscription)

        await self._notify_event("topic_subscribe", {
            "topic": topic_name,
            "subscription_id": subscription.id
        })

        return subscription.id

    def unsubscribe(self, topic_name: str, subscription_id: str) -> bool:
        """
        Se désabonne d'un topic.

        Args:
            topic_name: Nom du topic
            subscription_id: ID de l'abonnement

        Returns:
            True si désabonné, False sinon
        """
        if topic_name not in self._topics:
            return False

        for sub in self._topics[topic_name]:
            if sub.id == subscription_id:
                sub.active = False
                self._topics[topic_name].remove(sub)
                return True
        return False

    async def publish(
        self,
        topic_name: str,
        payload: Dict[str, Any],
        source: str = "",
        headers: Dict[str, str] = None
    ) -> Message:
        """
        Publie un message sur un topic.

        Args:
            topic_name: Nom du topic
            payload: Contenu du message
            source: Service source
            headers: Headers optionnels

        Returns:
            Le message publié
        """
        self._ensure_topic(topic_name)

        message = Message(
            id=self._generate_message_id(),
            payload=payload,
            source=source,
            headers=headers or {}
        )

        self._message_history.append(message)
        self._stats["messages_sent"] += 1

        await self._notify_event("topic_publish", {
            "topic": topic_name,
            "message": message.to_dict(),
            "subscribers_count": len(self._topics[topic_name])
        })

        # Délivre le message à tous les abonnés
        for subscription in self._topics[topic_name]:
            if subscription.active:
                asyncio.create_task(
                    self._deliver_to_subscriber(
                        topic_name,
                        message,
                        subscription
                    )
                )

        return message

    async def _deliver_to_subscriber(
        self,
        topic_name: str,
        message: Message,
        subscription: Subscription
    ):
        """Délivre un message à un abonné avec gestion des erreurs."""
        retries = 0
        last_error = None

        while retries <= subscription.max_retries:
            try:
                if asyncio.iscoroutinefunction(subscription.handler):
                    await subscription.handler(message.payload)
                else:
                    subscription.handler(message.payload)

                self._stats["messages_received"] += 1

                await self._notify_event("topic_delivered", {
                    "topic": topic_name,
                    "message_id": message.id,
                    "subscription_id": subscription.id
                })
                return

            except Exception as e:
                retries += 1
                last_error = str(e)

                await self._notify_event("topic_delivery_failed", {
                    "topic": topic_name,
                    "message_id": message.id,
                    "subscription_id": subscription.id,
                    "retry": retries,
                    "error": last_error
                })

                if retries <= subscription.max_retries:
                    # Backoff exponentiel
                    await asyncio.sleep(0.1 * (2 ** retries))

        # Échec final - envoi en DLQ
        message.status = MessageStatus.DEAD_LETTER
        message.error = last_error
        message.retries = retries

        await self._send_to_dlq(topic_name, message)

    # ========== DEAD LETTER QUEUE ==========

    async def _send_to_dlq(self, source_name: str, message: Message):
        """Envoie un message en Dead Letter Queue."""
        dlq_name = f"{source_name}.dlq"

        if dlq_name not in self._dlq:
            self._dlq[dlq_name] = []

        message.status = MessageStatus.DEAD_LETTER
        self._dlq[dlq_name].append(message)
        self._stats["messages_dlq"] += 1
        self._stats["messages_failed"] += 1

        await self._notify_event("dlq_message", {
            "dlq": dlq_name,
            "message": message.to_dict()
        })

    async def receive_from_dlq(
        self,
        source_name: str
    ) -> Optional[Message]:
        """
        Récupère un message de la DLQ.

        Args:
            source_name: Nom de la source (queue ou topic)

        Returns:
            Le message ou None
        """
        dlq_name = f"{source_name}.dlq"

        if dlq_name not in self._dlq or not self._dlq[dlq_name]:
            return None

        return self._dlq[dlq_name].pop(0)

    def get_dlq_size(self, source_name: str) -> int:
        """Retourne la taille d'une DLQ."""
        dlq_name = f"{source_name}.dlq"
        if dlq_name not in self._dlq:
            return 0
        return len(self._dlq[dlq_name])

    def get_dlq_messages(self, source_name: str, limit: int = 50) -> List[Dict]:
        """Retourne les messages d'une DLQ."""
        dlq_name = f"{source_name}.dlq"
        if dlq_name not in self._dlq:
            return []
        return [m.to_dict() for m in self._dlq[dlq_name][-limit:]]

    # ========== MÉTRIQUES ET CONTRÔLE ==========

    def get_stats(self) -> Dict:
        """Retourne les statistiques du broker."""
        return {
            **self._stats,
            "queues": list(self._queues.keys()),
            "topics": list(self._topics.keys()),
            "dlqs": list(self._dlq.keys()),
            "active_subscriptions": sum(
                len([s for s in subs if s.active])
                for subs in self._topics.values()
            )
        }

    def get_topics(self) -> List[Dict]:
        """Retourne la liste des topics avec leurs abonnements."""
        result = []
        for topic_name, subscriptions in self._topics.items():
            result.append({
                "name": topic_name,
                "subscribers": [
                    {"id": s.id, "active": s.active}
                    for s in subscriptions
                ]
            })
        return result

    def get_queues(self) -> List[Dict]:
        """Retourne la liste des queues avec leurs tailles."""
        result = []
        for queue_name in self._queues:
            result.append({
                "name": queue_name,
                "size": self.get_queue_size(queue_name),
                "dlq_size": self.get_dlq_size(queue_name)
            })
        return result

    def get_message_history(self, limit: int = 100) -> List[Dict]:
        """Retourne l'historique des messages."""
        return [m.to_dict() for m in self._message_history[-limit:]]

    def reset(self):
        """Réinitialise le broker."""
        self._queues.clear()
        self._queue_messages.clear()
        self._topics.clear()
        self._dlq.clear()
        self._message_history.clear()
        self._stats = {
            "messages_sent": 0,
            "messages_received": 0,
            "messages_failed": 0,
            "messages_dlq": 0
        }


# Instance singleton du broker
_broker_instance: Optional[MessageBroker] = None


def get_broker() -> MessageBroker:
    """Retourne l'instance singleton du broker."""
    global _broker_instance
    if _broker_instance is None:
        _broker_instance = MessageBroker()
    return _broker_instance


def reset_broker():
    """Réinitialise l'instance du broker."""
    global _broker_instance
    if _broker_instance:
        _broker_instance.reset()
    _broker_instance = MessageBroker()
