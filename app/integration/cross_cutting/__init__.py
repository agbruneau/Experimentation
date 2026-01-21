"""
Patterns transversaux (Cross-Cutting Concerns).

Ce module contient les implémentations des patterns de:
- Résilience (Circuit Breaker, Retry, Fallback, Timeout)
- Observabilité (Logging, Tracing, Metrics)
- Sécurité (JWT, OAuth, RBAC)
"""
from .circuit_breaker import CircuitBreaker, CircuitBreakerError, CircuitState
from .retry import (
    RetryPolicy,
    retry_with_backoff,
    with_fallback,
    with_timeout
)
from .observability import (
    Tracer,
    Span,
    MetricsCollector,
    StructuredLogger,
    get_tracer,
    get_metrics,
    get_logger
)
from .security import (
    JWTManager,
    RBACManager,
    SecurityContext,
    Permission,
    Role
)

__all__ = [
    # Circuit Breaker
    "CircuitBreaker",
    "CircuitBreakerError",
    "CircuitState",
    # Retry
    "RetryPolicy",
    "retry_with_backoff",
    "with_fallback",
    "with_timeout",
    # Observability
    "Tracer",
    "Span",
    "MetricsCollector",
    "StructuredLogger",
    "get_tracer",
    "get_metrics",
    "get_logger",
    # Security
    "JWTManager",
    "RBACManager",
    "SecurityContext",
    "Permission",
    "Role"
]
