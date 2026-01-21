"""
Patterns de résilience: Retry, Backoff, Fallback et Timeout.

Ces patterns permettent de gérer les échecs temporaires et d'assurer
la disponibilité des services même en cas de problèmes ponctuels.
"""
import asyncio
import functools
import random
import time
from typing import Any, Callable, Optional, TypeVar, Dict, List, Union
from dataclasses import dataclass, field
from enum import Enum
from datetime import datetime


T = TypeVar('T')


class BackoffStrategy(Enum):
    """Stratégies de backoff disponibles."""
    FIXED = "fixed"              # Délai fixe entre chaque retry
    LINEAR = "linear"            # Délai augmente linéairement
    EXPONENTIAL = "exponential"  # Délai double à chaque retry
    JITTER = "jitter"            # Exponential avec variation aléatoire


@dataclass
class RetryConfig:
    """Configuration de la politique de retry."""
    max_retries: int = 3
    initial_delay: float = 1.0       # Délai initial en secondes
    max_delay: float = 60.0          # Délai maximum
    backoff_strategy: BackoffStrategy = BackoffStrategy.EXPONENTIAL
    jitter_factor: float = 0.1       # Facteur de variation aléatoire
    retry_on: tuple = (Exception,)   # Types d'exceptions à retry
    dont_retry_on: tuple = ()        # Types d'exceptions à ne pas retry


@dataclass
class RetryStats:
    """Statistiques des retries."""
    total_attempts: int = 0
    successful_attempts: int = 0
    failed_attempts: int = 0
    total_retries: int = 0
    total_delay: float = 0.0
    last_error: Optional[str] = None
    history: List[Dict] = field(default_factory=list)


class RetryPolicy:
    """
    Politique de retry configurable.

    Usage:
        policy = RetryPolicy(max_retries=3, backoff_strategy=BackoffStrategy.EXPONENTIAL)

        # Avec décorateur
        @policy.retry
        async def risky_call():
            ...

        # Ou avec execute
        result = await policy.execute(risky_function)
    """

    def __init__(
        self,
        max_retries: int = 3,
        initial_delay: float = 1.0,
        max_delay: float = 60.0,
        backoff_strategy: BackoffStrategy = BackoffStrategy.EXPONENTIAL,
        jitter_factor: float = 0.1,
        retry_on: tuple = (Exception,),
        dont_retry_on: tuple = (),
        on_retry: Optional[Callable] = None
    ):
        self.config = RetryConfig(
            max_retries=max_retries,
            initial_delay=initial_delay,
            max_delay=max_delay,
            backoff_strategy=backoff_strategy,
            jitter_factor=jitter_factor,
            retry_on=retry_on,
            dont_retry_on=dont_retry_on
        )
        self.stats = RetryStats()
        self._on_retry = on_retry

    def _calculate_delay(self, attempt: int) -> float:
        """Calcule le délai avant le prochain retry."""
        config = self.config

        if config.backoff_strategy == BackoffStrategy.FIXED:
            delay = config.initial_delay
        elif config.backoff_strategy == BackoffStrategy.LINEAR:
            delay = config.initial_delay * (attempt + 1)
        elif config.backoff_strategy == BackoffStrategy.EXPONENTIAL:
            delay = config.initial_delay * (2 ** attempt)
        elif config.backoff_strategy == BackoffStrategy.JITTER:
            base_delay = config.initial_delay * (2 ** attempt)
            jitter = base_delay * config.jitter_factor * random.uniform(-1, 1)
            delay = base_delay + jitter
        else:
            delay = config.initial_delay

        return min(delay, config.max_delay)

    def _should_retry(self, exception: Exception) -> bool:
        """Détermine si l'exception permet un retry."""
        # Ne pas retry sur les exceptions explicitement exclues
        if isinstance(exception, self.config.dont_retry_on):
            return False
        # Retry sur les exceptions configurées
        return isinstance(exception, self.config.retry_on)

    async def execute(
        self,
        func: Callable[..., T],
        *args,
        **kwargs
    ) -> T:
        """
        Exécute une fonction avec la politique de retry.

        Args:
            func: Fonction à exécuter (sync ou async)
            *args, **kwargs: Arguments de la fonction

        Returns:
            Le résultat de la fonction

        Raises:
            La dernière exception si tous les retries échouent
        """
        last_exception = None
        is_async = asyncio.iscoroutinefunction(func)

        for attempt in range(self.config.max_retries + 1):
            self.stats.total_attempts += 1
            start_time = time.time()

            try:
                if is_async:
                    result = await func(*args, **kwargs)
                else:
                    result = func(*args, **kwargs)

                self.stats.successful_attempts += 1
                self.stats.history.append({
                    "timestamp": datetime.now().isoformat(),
                    "attempt": attempt + 1,
                    "status": "success",
                    "duration": time.time() - start_time
                })
                return result

            except Exception as e:
                last_exception = e
                self.stats.last_error = str(e)

                # Vérifier si on doit retry
                if not self._should_retry(e) or attempt >= self.config.max_retries:
                    self.stats.failed_attempts += 1
                    self.stats.history.append({
                        "timestamp": datetime.now().isoformat(),
                        "attempt": attempt + 1,
                        "status": "failed",
                        "error": str(e),
                        "final": True
                    })
                    raise

                # Calculer le délai
                delay = self._calculate_delay(attempt)
                self.stats.total_retries += 1
                self.stats.total_delay += delay

                self.stats.history.append({
                    "timestamp": datetime.now().isoformat(),
                    "attempt": attempt + 1,
                    "status": "retry",
                    "error": str(e),
                    "delay": delay
                })

                # Callback optionnel
                if self._on_retry:
                    try:
                        self._on_retry(attempt + 1, e, delay)
                    except Exception:
                        pass

                # Attendre avant le prochain retry
                await asyncio.sleep(delay)

        # Ne devrait pas arriver, mais au cas où
        if last_exception:
            raise last_exception
        raise RuntimeError("Unexpected retry state")

    def retry(self, func: Callable) -> Callable:
        """Décorateur pour appliquer la politique de retry."""
        if asyncio.iscoroutinefunction(func):
            @functools.wraps(func)
            async def async_wrapper(*args, **kwargs):
                return await self.execute(func, *args, **kwargs)
            return async_wrapper
        else:
            @functools.wraps(func)
            def sync_wrapper(*args, **kwargs):
                return asyncio.get_event_loop().run_until_complete(
                    self.execute(func, *args, **kwargs)
                )
            return sync_wrapper

    def get_stats(self) -> Dict:
        """Retourne les statistiques."""
        return {
            "total_attempts": self.stats.total_attempts,
            "successful_attempts": self.stats.successful_attempts,
            "failed_attempts": self.stats.failed_attempts,
            "total_retries": self.stats.total_retries,
            "total_delay": round(self.stats.total_delay, 2),
            "last_error": self.stats.last_error,
            "config": {
                "max_retries": self.config.max_retries,
                "backoff_strategy": self.config.backoff_strategy.value,
                "initial_delay": self.config.initial_delay
            }
        }

    def reset_stats(self):
        """Réinitialise les statistiques."""
        self.stats = RetryStats()


def retry_with_backoff(
    max_retries: int = 3,
    initial_delay: float = 1.0,
    backoff_strategy: BackoffStrategy = BackoffStrategy.EXPONENTIAL,
    retry_on: tuple = (Exception,)
) -> Callable:
    """
    Décorateur simple pour ajouter du retry avec backoff.

    Usage:
        @retry_with_backoff(max_retries=3)
        async def risky_function():
            ...
    """
    def decorator(func: Callable) -> Callable:
        policy = RetryPolicy(
            max_retries=max_retries,
            initial_delay=initial_delay,
            backoff_strategy=backoff_strategy,
            retry_on=retry_on
        )
        return policy.retry(func)
    return decorator


# ========== FALLBACK PATTERN ==========

class FallbackError(Exception):
    """Exception levée quand le fallback échoue aussi."""
    pass


@dataclass
class FallbackConfig:
    """Configuration du fallback."""
    fallback_value: Any = None
    fallback_function: Optional[Callable] = None
    cache_duration: float = 0  # Durée de cache du fallback (0 = pas de cache)


class Fallback:
    """
    Pattern Fallback - Solution de repli en cas d'échec.

    Usage:
        fallback = Fallback(fallback_value={"default": True})

        @fallback.with_fallback
        async def risky_call():
            ...
    """

    def __init__(
        self,
        fallback_value: Any = None,
        fallback_function: Optional[Callable] = None,
        on_fallback: Optional[Callable] = None
    ):
        self.fallback_value = fallback_value
        self.fallback_function = fallback_function
        self._on_fallback = on_fallback
        self._cached_result: Optional[Any] = None
        self._last_cache_time: Optional[float] = None
        self._cache_duration: float = 0
        self.stats = {
            "primary_calls": 0,
            "fallback_calls": 0,
            "cache_hits": 0
        }

    def set_cache(self, value: Any, duration: float = 300):
        """Définit une valeur en cache avec durée."""
        self._cached_result = value
        self._last_cache_time = time.time()
        self._cache_duration = duration

    def _get_cached(self) -> Optional[Any]:
        """Récupère la valeur en cache si valide."""
        if self._cached_result is None or self._last_cache_time is None:
            return None
        if self._cache_duration > 0:
            elapsed = time.time() - self._last_cache_time
            if elapsed > self._cache_duration:
                return None
        self.stats["cache_hits"] += 1
        return self._cached_result

    async def execute(
        self,
        func: Callable[..., T],
        *args,
        **kwargs
    ) -> T:
        """
        Exécute une fonction avec fallback.

        Args:
            func: Fonction à exécuter
            *args, **kwargs: Arguments de la fonction

        Returns:
            Le résultat ou la valeur de fallback
        """
        self.stats["primary_calls"] += 1
        is_async = asyncio.iscoroutinefunction(func)

        try:
            if is_async:
                result = await func(*args, **kwargs)
            else:
                result = func(*args, **kwargs)

            # Mettre en cache le résultat si cache activé
            if self._cache_duration > 0:
                self.set_cache(result, self._cache_duration)

            return result

        except Exception as e:
            self.stats["fallback_calls"] += 1

            # Callback optionnel
            if self._on_fallback:
                try:
                    self._on_fallback(e)
                except Exception:
                    pass

            # 1. Essayer le cache
            cached = self._get_cached()
            if cached is not None:
                return cached

            # 2. Essayer la fonction de fallback
            if self.fallback_function:
                try:
                    if asyncio.iscoroutinefunction(self.fallback_function):
                        return await self.fallback_function(*args, **kwargs)
                    return self.fallback_function(*args, **kwargs)
                except Exception:
                    pass

            # 3. Retourner la valeur de fallback
            if self.fallback_value is not None:
                return self.fallback_value

            # 4. Aucun fallback disponible
            raise FallbackError(f"Primary failed and no fallback available: {e}")

    def with_fallback(self, func: Callable) -> Callable:
        """Décorateur pour appliquer le fallback."""
        if asyncio.iscoroutinefunction(func):
            @functools.wraps(func)
            async def async_wrapper(*args, **kwargs):
                return await self.execute(func, *args, **kwargs)
            return async_wrapper
        else:
            @functools.wraps(func)
            def sync_wrapper(*args, **kwargs):
                return asyncio.get_event_loop().run_until_complete(
                    self.execute(func, *args, **kwargs)
                )
            return sync_wrapper


def with_fallback(
    fallback_value: Any = None,
    fallback_function: Optional[Callable] = None
) -> Callable:
    """
    Décorateur simple pour ajouter un fallback.

    Usage:
        @with_fallback(fallback_value={"default": True})
        async def risky_function():
            ...
    """
    def decorator(func: Callable) -> Callable:
        fb = Fallback(
            fallback_value=fallback_value,
            fallback_function=fallback_function
        )
        return fb.with_fallback(func)
    return decorator


# ========== TIMEOUT PATTERN ==========

class TimeoutError(Exception):
    """Exception levée quand le timeout est dépassé."""
    pass


@dataclass
class TimeoutConfig:
    """Configuration du timeout."""
    timeout: float = 30.0  # Timeout en secondes
    cancel_on_timeout: bool = True


class Timeout:
    """
    Pattern Timeout - Limite le temps d'exécution.

    Usage:
        timeout = Timeout(seconds=5.0)

        @timeout.with_timeout
        async def slow_call():
            ...
    """

    def __init__(
        self,
        seconds: float = 30.0,
        on_timeout: Optional[Callable] = None
    ):
        self.timeout = seconds
        self._on_timeout = on_timeout
        self.stats = {
            "total_calls": 0,
            "completed_calls": 0,
            "timeout_calls": 0
        }

    async def execute(
        self,
        func: Callable[..., T],
        *args,
        **kwargs
    ) -> T:
        """
        Exécute une fonction avec timeout.

        Args:
            func: Fonction async à exécuter
            *args, **kwargs: Arguments de la fonction

        Returns:
            Le résultat de la fonction

        Raises:
            TimeoutError si le timeout est dépassé
        """
        self.stats["total_calls"] += 1

        if not asyncio.iscoroutinefunction(func):
            raise ValueError("Timeout requires an async function")

        try:
            result = await asyncio.wait_for(
                func(*args, **kwargs),
                timeout=self.timeout
            )
            self.stats["completed_calls"] += 1
            return result

        except asyncio.TimeoutError:
            self.stats["timeout_calls"] += 1

            if self._on_timeout:
                try:
                    self._on_timeout(self.timeout)
                except Exception:
                    pass

            raise TimeoutError(
                f"Operation timed out after {self.timeout} seconds"
            )

    def with_timeout(self, func: Callable) -> Callable:
        """Décorateur pour appliquer le timeout."""
        @functools.wraps(func)
        async def async_wrapper(*args, **kwargs):
            return await self.execute(func, *args, **kwargs)
        return async_wrapper


def with_timeout(seconds: float = 30.0) -> Callable:
    """
    Décorateur simple pour ajouter un timeout.

    Usage:
        @with_timeout(seconds=5.0)
        async def slow_function():
            ...
    """
    def decorator(func: Callable) -> Callable:
        t = Timeout(seconds=seconds)
        return t.with_timeout(func)
    return decorator


# ========== COMBINAISON DES PATTERNS ==========

class ResilientCall:
    """
    Combine tous les patterns de résilience en un seul.

    Usage:
        resilient = ResilientCall(
            timeout=5.0,
            max_retries=3,
            fallback_value={"default": True}
        )

        result = await resilient.execute(risky_function)
    """

    def __init__(
        self,
        timeout: float = 30.0,
        max_retries: int = 3,
        initial_delay: float = 1.0,
        backoff_strategy: BackoffStrategy = BackoffStrategy.EXPONENTIAL,
        fallback_value: Any = None,
        fallback_function: Optional[Callable] = None,
        on_retry: Optional[Callable] = None,
        on_timeout: Optional[Callable] = None,
        on_fallback: Optional[Callable] = None
    ):
        self.timeout_handler = Timeout(
            seconds=timeout,
            on_timeout=on_timeout
        )
        self.retry_policy = RetryPolicy(
            max_retries=max_retries,
            initial_delay=initial_delay,
            backoff_strategy=backoff_strategy,
            on_retry=on_retry
        )
        self.fallback_handler = Fallback(
            fallback_value=fallback_value,
            fallback_function=fallback_function,
            on_fallback=on_fallback
        )

    async def execute(
        self,
        func: Callable[..., T],
        *args,
        **kwargs
    ) -> T:
        """
        Exécute avec timeout, retry et fallback.

        Ordre d'application:
        1. Timeout wraps la fonction
        2. Retry appliqué sur le timeout
        3. Fallback si tous les retries échouent
        """
        async def with_timeout_wrapped():
            return await self.timeout_handler.execute(func, *args, **kwargs)

        async def with_retry_wrapped():
            return await self.retry_policy.execute(with_timeout_wrapped)

        return await self.fallback_handler.execute(with_retry_wrapped)

    def wrap(self, func: Callable) -> Callable:
        """Décorateur pour appliquer tous les patterns."""
        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            return await self.execute(func, *args, **kwargs)
        return wrapper

    def get_stats(self) -> Dict:
        """Retourne les statistiques combinées."""
        return {
            "timeout": self.timeout_handler.stats,
            "retry": self.retry_policy.get_stats(),
            "fallback": self.fallback_handler.stats
        }
