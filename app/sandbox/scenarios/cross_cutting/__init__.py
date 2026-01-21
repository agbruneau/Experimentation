"""
Scénarios Cross-Cutting - Patterns transversaux.

Ces scénarios démontrent les patterns de résilience, observabilité et sécurité.
"""
from .cross_01_circuit_breaker import scenario as cross_01
from .cross_02_tracing import scenario as cross_02
from .cross_03_security import scenario as cross_03

__all__ = ["cross_01", "cross_02", "cross_03"]
