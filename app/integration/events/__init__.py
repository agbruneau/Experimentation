"""
Module d'intégration par événements (Pilier Événements).

Ce module implémente les patterns d'intégration événementielle:
- Message Broker (queues et topics)
- Event Store (event sourcing)
- CQRS (Command Query Responsibility Segregation)
- Saga (transactions distribuées)
- Outbox (publication fiable)
"""

from app.integration.events.broker import (
    MessageBroker,
    Message,
    MessageStatus,
    get_broker,
    reset_broker
)

from app.integration.events.event_store import (
    EventStore,
    Event,
    ConcurrencyError,
    policy_reducer,
    get_event_store,
    reset_event_store
)

from app.integration.events.cqrs import (
    CQRSBus,
    Command,
    Query,
    CreatePolicyCommand,
    ActivatePolicyCommand,
    ModifyPolicyCommand,
    CancelPolicyCommand,
    GetPolicyQuery,
    ListPoliciesByCustomerQuery,
    GetPolicySummaryQuery,
    get_cqrs_bus
)

from app.integration.events.saga import (
    SagaOrchestrator,
    SagaStatus,
    SagaStep,
    SagaExecution,
    SubscriptionSaga
)

from app.integration.events.outbox import (
    OutboxProcessor,
    OutboxEntry,
    OutboxStatus,
    AtomicBusinessOperation,
    get_outbox,
    reset_outbox
)

__all__ = [
    # Broker
    "MessageBroker",
    "Message",
    "MessageStatus",
    "get_broker",
    "reset_broker",
    # Event Store
    "EventStore",
    "Event",
    "ConcurrencyError",
    "policy_reducer",
    "get_event_store",
    "reset_event_store",
    # CQRS
    "CQRSBus",
    "Command",
    "Query",
    "CreatePolicyCommand",
    "ActivatePolicyCommand",
    "ModifyPolicyCommand",
    "CancelPolicyCommand",
    "GetPolicyQuery",
    "ListPoliciesByCustomerQuery",
    "GetPolicySummaryQuery",
    "get_cqrs_bus",
    # Saga
    "SagaOrchestrator",
    "SagaStatus",
    "SagaStep",
    "SagaExecution",
    "SubscriptionSaga",
    # Outbox
    "OutboxProcessor",
    "OutboxEntry",
    "OutboxStatus",
    "AtomicBusinessOperation",
    "get_outbox",
    "reset_outbox"
]
