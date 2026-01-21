"""
Observabilité - Les trois piliers: Logs, Metrics, Traces.

Ce module fournit une implémentation simple pour:
- Logging structuré avec corrélation
- Distributed Tracing
- Collection de métriques
"""
import asyncio
import json
import time
import uuid
from typing import Any, Callable, Dict, List, Optional
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from contextlib import contextmanager, asynccontextmanager
from contextvars import ContextVar


# Context variable pour la propagation du trace
_current_trace_context: ContextVar[Optional['TraceContext']] = ContextVar(
    'current_trace_context',
    default=None
)


# ========== LOGGING STRUCTURÉ ==========

class LogLevel(Enum):
    """Niveaux de log."""
    DEBUG = "DEBUG"
    INFO = "INFO"
    WARNING = "WARNING"
    ERROR = "ERROR"
    CRITICAL = "CRITICAL"


@dataclass
class LogEntry:
    """Entrée de log structurée."""
    timestamp: str
    level: LogLevel
    message: str
    service: str = "default"
    trace_id: Optional[str] = None
    span_id: Optional[str] = None
    attributes: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict:
        return {
            "timestamp": self.timestamp,
            "level": self.level.value,
            "message": self.message,
            "service": self.service,
            "trace_id": self.trace_id,
            "span_id": self.span_id,
            **self.attributes
        }

    def to_json(self) -> str:
        return json.dumps(self.to_dict())


class StructuredLogger:
    """
    Logger structuré avec support pour la corrélation.

    Features:
    - Logs en format JSON
    - Corrélation avec trace_id
    - Attributs personnalisés
    - Filtrage par niveau
    """

    def __init__(self, service_name: str = "default", min_level: LogLevel = LogLevel.DEBUG):
        self.service_name = service_name
        self.min_level = min_level
        self._logs: List[LogEntry] = []
        self._handlers: List[Callable[[LogEntry], None]] = []
        self._level_priority = {
            LogLevel.DEBUG: 0,
            LogLevel.INFO: 1,
            LogLevel.WARNING: 2,
            LogLevel.ERROR: 3,
            LogLevel.CRITICAL: 4
        }

    def add_handler(self, handler: Callable[[LogEntry], None]):
        """Ajoute un handler pour les logs."""
        self._handlers.append(handler)

    def _should_log(self, level: LogLevel) -> bool:
        """Vérifie si le niveau doit être logué."""
        return self._level_priority[level] >= self._level_priority[self.min_level]

    def _log(self, level: LogLevel, message: str, **attributes):
        """Crée et enregistre une entrée de log."""
        if not self._should_log(level):
            return

        # Récupérer le contexte de trace actuel
        trace_context = _current_trace_context.get()
        trace_id = trace_context.trace_id if trace_context else None
        span_id = trace_context.current_span_id if trace_context else None

        entry = LogEntry(
            timestamp=datetime.now().isoformat(),
            level=level,
            message=message,
            service=self.service_name,
            trace_id=trace_id,
            span_id=span_id,
            attributes=attributes
        )

        self._logs.append(entry)

        # Appeler les handlers
        for handler in self._handlers:
            try:
                handler(entry)
            except Exception:
                pass

        return entry

    def debug(self, message: str, **kwargs):
        """Log niveau DEBUG."""
        return self._log(LogLevel.DEBUG, message, **kwargs)

    def info(self, message: str, **kwargs):
        """Log niveau INFO."""
        return self._log(LogLevel.INFO, message, **kwargs)

    def warning(self, message: str, **kwargs):
        """Log niveau WARNING."""
        return self._log(LogLevel.WARNING, message, **kwargs)

    def error(self, message: str, **kwargs):
        """Log niveau ERROR."""
        return self._log(LogLevel.ERROR, message, **kwargs)

    def critical(self, message: str, **kwargs):
        """Log niveau CRITICAL."""
        return self._log(LogLevel.CRITICAL, message, **kwargs)

    def get_logs(
        self,
        limit: int = 100,
        level: Optional[LogLevel] = None,
        trace_id: Optional[str] = None
    ) -> List[Dict]:
        """Récupère les logs avec filtres optionnels."""
        filtered = self._logs

        if level:
            filtered = [l for l in filtered if l.level == level]

        if trace_id:
            filtered = [l for l in filtered if l.trace_id == trace_id]

        return [l.to_dict() for l in filtered[-limit:]]

    def clear(self):
        """Vide les logs."""
        self._logs = []


# ========== DISTRIBUTED TRACING ==========

@dataclass
class TraceContext:
    """Contexte de trace propagé."""
    trace_id: str
    current_span_id: Optional[str] = None
    parent_span_id: Optional[str] = None
    baggage: Dict[str, str] = field(default_factory=dict)

    def to_headers(self) -> Dict[str, str]:
        """Convertit en headers HTTP pour propagation."""
        return {
            "X-Trace-ID": self.trace_id,
            "X-Span-ID": self.current_span_id or "",
            "X-Parent-Span-ID": self.parent_span_id or "",
            "X-Baggage": json.dumps(self.baggage)
        }

    @classmethod
    def from_headers(cls, headers: Dict[str, str]) -> Optional['TraceContext']:
        """Crée un contexte depuis des headers HTTP."""
        trace_id = headers.get("X-Trace-ID")
        if not trace_id:
            return None

        baggage = {}
        if "X-Baggage" in headers:
            try:
                baggage = json.loads(headers["X-Baggage"])
            except Exception:
                pass

        return cls(
            trace_id=trace_id,
            current_span_id=headers.get("X-Span-ID") or None,
            parent_span_id=headers.get("X-Parent-Span-ID") or None,
            baggage=baggage
        )


@dataclass
class Span:
    """Représente une opération dans une trace."""
    span_id: str
    trace_id: str
    operation_name: str
    service_name: str
    parent_span_id: Optional[str] = None
    start_time: float = field(default_factory=time.time)
    end_time: Optional[float] = None
    status: str = "OK"
    tags: Dict[str, Any] = field(default_factory=dict)
    logs: List[Dict] = field(default_factory=list)
    error: Optional[str] = None

    @property
    def duration_ms(self) -> Optional[float]:
        """Durée en millisecondes."""
        if self.end_time is None:
            return None
        return (self.end_time - self.start_time) * 1000

    def set_tag(self, key: str, value: Any):
        """Ajoute un tag au span."""
        self.tags[key] = value

    def log_event(self, event: str, **kwargs):
        """Ajoute un événement au span."""
        self.logs.append({
            "timestamp": datetime.now().isoformat(),
            "event": event,
            **kwargs
        })

    def set_error(self, error: Exception):
        """Marque le span en erreur."""
        self.status = "ERROR"
        self.error = str(error)
        self.set_tag("error", True)
        self.set_tag("error.message", str(error))
        self.set_tag("error.type", type(error).__name__)

    def finish(self):
        """Termine le span."""
        self.end_time = time.time()

    def to_dict(self) -> Dict:
        return {
            "span_id": self.span_id,
            "trace_id": self.trace_id,
            "operation_name": self.operation_name,
            "service_name": self.service_name,
            "parent_span_id": self.parent_span_id,
            "start_time": datetime.fromtimestamp(self.start_time).isoformat(),
            "end_time": datetime.fromtimestamp(self.end_time).isoformat() if self.end_time else None,
            "duration_ms": self.duration_ms,
            "status": self.status,
            "tags": self.tags,
            "logs": self.logs,
            "error": self.error
        }


class Tracer:
    """
    Gestionnaire de tracing distribué.

    Features:
    - Création et gestion des traces
    - Propagation du contexte
    - Visualisation des spans
    """

    def __init__(self, service_name: str = "default"):
        self.service_name = service_name
        self._traces: Dict[str, List[Span]] = {}
        self._spans: Dict[str, Span] = {}

    def _generate_id(self) -> str:
        """Génère un ID unique."""
        return uuid.uuid4().hex[:16]

    def start_trace(self, operation_name: str, **tags) -> Span:
        """Démarre une nouvelle trace."""
        trace_id = self._generate_id()
        span_id = self._generate_id()

        span = Span(
            span_id=span_id,
            trace_id=trace_id,
            operation_name=operation_name,
            service_name=self.service_name,
            tags=tags
        )

        self._traces[trace_id] = [span]
        self._spans[span_id] = span

        # Mettre à jour le contexte
        context = TraceContext(
            trace_id=trace_id,
            current_span_id=span_id
        )
        _current_trace_context.set(context)

        return span

    def start_span(
        self,
        operation_name: str,
        parent_span: Optional[Span] = None,
        **tags
    ) -> Span:
        """Démarre un nouveau span dans la trace courante."""
        current_context = _current_trace_context.get()

        if current_context:
            trace_id = current_context.trace_id
            parent_span_id = parent_span.span_id if parent_span else current_context.current_span_id
        elif parent_span:
            trace_id = parent_span.trace_id
            parent_span_id = parent_span.span_id
        else:
            # Pas de contexte, démarrer une nouvelle trace
            return self.start_trace(operation_name, **tags)

        span_id = self._generate_id()

        span = Span(
            span_id=span_id,
            trace_id=trace_id,
            operation_name=operation_name,
            service_name=self.service_name,
            parent_span_id=parent_span_id,
            tags=tags
        )

        if trace_id not in self._traces:
            self._traces[trace_id] = []
        self._traces[trace_id].append(span)
        self._spans[span_id] = span

        # Mettre à jour le contexte
        new_context = TraceContext(
            trace_id=trace_id,
            current_span_id=span_id,
            parent_span_id=parent_span_id,
            baggage=current_context.baggage if current_context else {}
        )
        _current_trace_context.set(new_context)

        return span

    def finish_span(self, span: Span):
        """Termine un span et restaure le contexte parent."""
        span.finish()

        # Restaurer le contexte parent
        if span.parent_span_id:
            current_context = _current_trace_context.get()
            if current_context:
                parent_context = TraceContext(
                    trace_id=span.trace_id,
                    current_span_id=span.parent_span_id,
                    parent_span_id=None,
                    baggage=current_context.baggage
                )
                _current_trace_context.set(parent_context)

    @contextmanager
    def trace(self, operation_name: str, **tags):
        """Context manager pour tracer une opération."""
        span = self.start_trace(operation_name, **tags)
        try:
            yield span
        except Exception as e:
            span.set_error(e)
            raise
        finally:
            self.finish_span(span)

    @contextmanager
    def span(self, operation_name: str, **tags):
        """Context manager pour un span."""
        span = self.start_span(operation_name, **tags)
        try:
            yield span
        except Exception as e:
            span.set_error(e)
            raise
        finally:
            self.finish_span(span)

    @asynccontextmanager
    async def async_span(self, operation_name: str, **tags):
        """Context manager async pour un span."""
        span = self.start_span(operation_name, **tags)
        try:
            yield span
        except Exception as e:
            span.set_error(e)
            raise
        finally:
            self.finish_span(span)

    def get_trace(self, trace_id: str) -> Optional[List[Dict]]:
        """Récupère une trace complète."""
        if trace_id not in self._traces:
            return None
        return [s.to_dict() for s in self._traces[trace_id]]

    def get_trace_tree(self, trace_id: str) -> Optional[Dict]:
        """Récupère une trace sous forme d'arbre."""
        if trace_id not in self._traces:
            return None

        spans = self._traces[trace_id]
        spans_by_id = {s.span_id: s for s in spans}

        # Trouver le root span
        root_spans = [s for s in spans if s.parent_span_id is None]
        if not root_spans:
            return None

        def build_tree(span: Span) -> Dict:
            node = span.to_dict()
            children = [s for s in spans if s.parent_span_id == span.span_id]
            if children:
                node["children"] = [build_tree(c) for c in children]
            return node

        return build_tree(root_spans[0])

    def get_all_traces(self, limit: int = 50) -> List[Dict]:
        """Récupère toutes les traces récentes."""
        result = []
        for trace_id, spans in list(self._traces.items())[-limit:]:
            root_spans = [s for s in spans if s.parent_span_id is None]
            if root_spans:
                root = root_spans[0]
                result.append({
                    "trace_id": trace_id,
                    "operation": root.operation_name,
                    "service": root.service_name,
                    "start_time": datetime.fromtimestamp(root.start_time).isoformat(),
                    "duration_ms": root.duration_ms,
                    "status": root.status,
                    "span_count": len(spans)
                })
        return result

    def inject_context(self, headers: Dict[str, str]):
        """Injecte le contexte de trace dans des headers."""
        context = _current_trace_context.get()
        if context:
            headers.update(context.to_headers())

    def extract_context(self, headers: Dict[str, str]) -> Optional[TraceContext]:
        """Extrait le contexte de trace depuis des headers."""
        context = TraceContext.from_headers(headers)
        if context:
            _current_trace_context.set(context)
        return context

    def clear(self):
        """Vide toutes les traces."""
        self._traces = {}
        self._spans = {}


# ========== METRICS ==========

class MetricType(Enum):
    """Types de métriques."""
    COUNTER = "counter"      # Compteur qui ne fait qu'augmenter
    GAUGE = "gauge"          # Valeur qui peut monter ou descendre
    HISTOGRAM = "histogram"  # Distribution de valeurs
    TIMER = "timer"          # Mesure de durée


@dataclass
class Metric:
    """Représente une métrique."""
    name: str
    type: MetricType
    value: float
    timestamp: str = field(default_factory=lambda: datetime.now().isoformat())
    tags: Dict[str, str] = field(default_factory=dict)
    unit: str = ""

    def to_dict(self) -> Dict:
        return {
            "name": self.name,
            "type": self.type.value,
            "value": self.value,
            "timestamp": self.timestamp,
            "tags": self.tags,
            "unit": self.unit
        }


class MetricsCollector:
    """
    Collecteur de métriques.

    Features:
    - Compteurs, gauges, histogrammes
    - Tags pour le filtrage
    - Agrégation simple
    """

    def __init__(self, service_name: str = "default"):
        self.service_name = service_name
        self._counters: Dict[str, float] = {}
        self._gauges: Dict[str, float] = {}
        self._histograms: Dict[str, List[float]] = {}
        self._metrics_history: List[Metric] = []

    def _make_key(self, name: str, tags: Dict[str, str]) -> str:
        """Crée une clé unique pour une métrique."""
        tag_str = ",".join(f"{k}={v}" for k, v in sorted(tags.items()))
        return f"{name}:{tag_str}" if tag_str else name

    def increment(self, name: str, value: float = 1.0, **tags):
        """Incrémente un compteur."""
        key = self._make_key(name, tags)
        self._counters[key] = self._counters.get(key, 0) + value

        metric = Metric(
            name=name,
            type=MetricType.COUNTER,
            value=self._counters[key],
            tags={"service": self.service_name, **tags}
        )
        self._metrics_history.append(metric)

    def gauge(self, name: str, value: float, **tags):
        """Définit une gauge."""
        key = self._make_key(name, tags)
        self._gauges[key] = value

        metric = Metric(
            name=name,
            type=MetricType.GAUGE,
            value=value,
            tags={"service": self.service_name, **tags}
        )
        self._metrics_history.append(metric)

    def histogram(self, name: str, value: float, **tags):
        """Ajoute une valeur à un histogramme."""
        key = self._make_key(name, tags)
        if key not in self._histograms:
            self._histograms[key] = []
        self._histograms[key].append(value)

        metric = Metric(
            name=name,
            type=MetricType.HISTOGRAM,
            value=value,
            tags={"service": self.service_name, **tags}
        )
        self._metrics_history.append(metric)

    @contextmanager
    def timer(self, name: str, **tags):
        """Context manager pour mesurer une durée."""
        start = time.time()
        try:
            yield
        finally:
            duration = (time.time() - start) * 1000  # en ms
            self.histogram(name, duration, **tags)

    def get_counter(self, name: str, **tags) -> float:
        """Récupère la valeur d'un compteur."""
        key = self._make_key(name, tags)
        return self._counters.get(key, 0)

    def get_gauge(self, name: str, **tags) -> Optional[float]:
        """Récupère la valeur d'une gauge."""
        key = self._make_key(name, tags)
        return self._gauges.get(key)

    def get_histogram_stats(self, name: str, **tags) -> Optional[Dict]:
        """Récupère les statistiques d'un histogramme."""
        key = self._make_key(name, tags)
        if key not in self._histograms or not self._histograms[key]:
            return None

        values = self._histograms[key]
        sorted_values = sorted(values)
        n = len(sorted_values)

        return {
            "count": n,
            "min": min(values),
            "max": max(values),
            "avg": sum(values) / n,
            "p50": sorted_values[int(n * 0.5)],
            "p90": sorted_values[int(n * 0.9)],
            "p99": sorted_values[min(int(n * 0.99), n - 1)]
        }

    def get_all_metrics(self) -> Dict:
        """Récupère toutes les métriques actuelles."""
        return {
            "counters": self._counters.copy(),
            "gauges": self._gauges.copy(),
            "histograms": {
                k: self.get_histogram_stats(k)
                for k in self._histograms
            }
        }

    def get_metrics_history(self, limit: int = 100) -> List[Dict]:
        """Récupère l'historique des métriques."""
        return [m.to_dict() for m in self._metrics_history[-limit:]]

    def clear(self):
        """Réinitialise toutes les métriques."""
        self._counters = {}
        self._gauges = {}
        self._histograms = {}
        self._metrics_history = []


# ========== INSTANCES GLOBALES ==========

_tracer: Optional[Tracer] = None
_metrics: Optional[MetricsCollector] = None
_logger: Optional[StructuredLogger] = None


def get_tracer(service_name: str = "default") -> Tracer:
    """Récupère ou crée le tracer global."""
    global _tracer
    if _tracer is None:
        _tracer = Tracer(service_name)
    return _tracer


def get_metrics(service_name: str = "default") -> MetricsCollector:
    """Récupère ou crée le collecteur de métriques global."""
    global _metrics
    if _metrics is None:
        _metrics = MetricsCollector(service_name)
    return _metrics


def get_logger(service_name: str = "default") -> StructuredLogger:
    """Récupère ou crée le logger global."""
    global _logger
    if _logger is None:
        _logger = StructuredLogger(service_name)
    return _logger


def reset_observability():
    """Réinitialise toutes les instances globales."""
    global _tracer, _metrics, _logger
    if _tracer:
        _tracer.clear()
    if _metrics:
        _metrics.clear()
    if _logger:
        _logger.clear()
    _tracer = None
    _metrics = None
    _logger = None
