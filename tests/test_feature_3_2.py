"""Tests pour Feature 3.2: Visualiseur Flux D3.js."""
import pytest
from pathlib import Path
from httpx import AsyncClient


def test_visualizer_exports():
    """Test que le visualiseur exporte les fonctions nécessaires."""
    js = Path("static/js/flow-visualizer.js").read_text()
    assert "initFlowVisualizer" in js
    assert "addNode" in js
    assert "animateMessage" in js
    assert "FlowVisualizer" in js


def test_visualizer_pillar_colors():
    """Test que les couleurs des piliers sont définies."""
    js = Path("static/js/flow-visualizer.js").read_text()
    assert "applications" in js
    assert "events" in js
    assert "data" in js
    assert "#3b82f6" in js  # Bleu
    assert "#f97316" in js  # Orange
    assert "#22c55e" in js  # Vert


def test_visualizer_features():
    """Test que les fonctionnalités principales sont présentes."""
    js = Path("static/js/flow-visualizer.js").read_text()
    # Force-directed layout
    assert "forceSimulation" in js
    assert "forceLink" in js
    assert "forceManyBody" in js
    # Zoom/Pan
    assert "d3.zoom" in js
    # Timeline
    assert "timeline" in js
    assert "replayTimeline" in js
    # SSE
    assert "EventSource" in js
    assert "connectSSE" in js


@pytest.mark.asyncio
async def test_visualizer_page(client):
    """Test que la page du visualiseur charge correctement."""
    async with client:
        r = await client.get("/sandbox/visualizer")
        assert r.status_code == 200
        # Vérifie que les scripts sont inclus
        assert "d3" in r.text.lower() or "flow-visualizer" in r.text.lower()


def test_sse_endpoint_exists():
    """Test que l'endpoint SSE est défini dans l'app."""
    from app.main import app
    routes = [r.path for r in app.routes]
    assert "/events/stream" in routes


def test_visualizer_drag_handlers():
    """Test que les handlers de drag sont présents."""
    js = Path("static/js/flow-visualizer.js").read_text()
    assert "dragStarted" in js
    assert "dragged" in js
    assert "dragEnded" in js


def test_visualizer_export_import():
    """Test que les fonctions export/import sont présentes."""
    js = Path("static/js/flow-visualizer.js").read_text()
    assert "exportState" in js
    assert "importState" in js


def test_visualizer_reset():
    """Test que la fonction reset est présente."""
    js = Path("static/js/flow-visualizer.js").read_text()
    assert "reset()" in js or "reset ()" in js


def test_visualizer_markers():
    """Test que les marqueurs de flèches sont définis."""
    js = Path("static/js/flow-visualizer.js").read_text()
    assert "marker" in js.lower()
    assert "arrow" in js.lower()
