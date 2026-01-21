"""Tests pour Feature 3.4: Module 7 - Event-Driven (Event Store, CQRS)."""
import pytest
from pathlib import Path
from app.integration.events.event_store import EventStore, policy_reducer


@pytest.mark.asyncio
async def test_event_store_append():
    """Test ajout d'événement dans l'Event Store."""
    es = EventStore()
    event = await es.append("p-1", {"type": "PolicyCreated", "data": {"premium": 850}})

    assert event.aggregate_id == "p-1"
    assert event.type == "PolicyCreated"
    assert event.version == 1


@pytest.mark.asyncio
async def test_event_store_multiple_events():
    """Test ajout de plusieurs événements."""
    es = EventStore()
    await es.append("p-1", {"type": "PolicyCreated", "data": {"status": "DRAFT"}})
    await es.append("p-1", {"type": "PolicyActivated", "data": {"status": "ACTIVE"}})

    events = await es.get_events("p-1")
    assert len(events) == 2
    assert events[0].version == 1
    assert events[1].version == 2


@pytest.mark.asyncio
async def test_event_store_rebuild_state():
    """Test reconstruction de l'état depuis les événements."""
    es = EventStore()
    await es.append("p-1", {"type": "PolicyCreated", "data": {"status": "DRAFT", "premium": 850}})
    await es.append("p-1", {"type": "PolicyActivated", "data": {"status": "ACTIVE"}})

    state = await es.rebuild_state("p-1")
    assert state["status"] == "ACTIVE"
    assert state["premium"] == 850


@pytest.mark.asyncio
async def test_event_store_with_policy_reducer():
    """Test avec le reducer spécifique aux polices."""
    es = EventStore()

    await es.append("p-1", {
        "type": "PolicyCreated",
        "data": {
            "policy_number": "POL-001",
            "customer_id": "C001",
            "product": "AUTO",
            "premium": 850
        }
    })

    await es.append("p-1", {
        "type": "PolicyActivated",
        "data": {
            "start_date": "2024-01-01",
            "end_date": "2024-12-31"
        }
    })

    state = await es.rebuild_state("p-1", policy_reducer)
    assert state["status"] == "ACTIVE"
    assert state["premium"] == 850
    assert state["start_date"] == "2024-01-01"


@pytest.mark.asyncio
async def test_event_store_snapshot():
    """Test création et utilisation de snapshot."""
    es = EventStore()
    await es.append("p-1", {"type": "PolicyCreated", "data": {"premium": 850}})

    state = await es.rebuild_state("p-1")
    await es.create_snapshot("p-1", state)

    snapshot = await es.get_snapshot("p-1")
    assert snapshot is not None
    assert snapshot["premium"] == 850


@pytest.mark.asyncio
async def test_event_store_version():
    """Test récupération de version."""
    es = EventStore()
    await es.append("p-1", {"type": "PolicyCreated", "data": {}})
    await es.append("p-1", {"type": "PolicyActivated", "data": {}})

    version = es.get_current_version("p-1")
    assert version == 2


@pytest.mark.asyncio
async def test_event_store_global_stream():
    """Test du flux global d'événements."""
    es = EventStore()
    await es.append("p-1", {"type": "PolicyCreated", "data": {}})
    await es.append("p-2", {"type": "PolicyCreated", "data": {}})

    stream = es.get_global_stream()
    assert len(stream) == 2


@pytest.mark.asyncio
async def test_event_store_stats():
    """Test des statistiques."""
    es = EventStore()
    await es.append("p-1", {"type": "PolicyCreated", "data": {}})
    await es.append("p-1", {"type": "PolicyActivated", "data": {}})

    stats = es.get_stats()
    assert stats["total_events"] == 2
    assert stats["aggregates_count"] == 1


# Tests CQRS
@pytest.mark.asyncio
async def test_cqrs_create_policy():
    """Test création de police via CQRS."""
    from app.integration.events.cqrs import (
        CQRSBus, CreatePolicyCommand
    )
    from app.integration.events.event_store import reset_event_store

    reset_event_store()
    bus = CQRSBus()

    command = CreatePolicyCommand(
        customer_id="C001",
        product="AUTO",
        premium=850.0,
        coverages=["RC", "VOL"]
    )

    result = await bus.send_command(command)

    assert result["success"] is True
    assert "policy_id" in result


@pytest.mark.asyncio
async def test_cqrs_query_policy():
    """Test requête de police via CQRS."""
    from app.integration.events.cqrs import (
        CQRSBus, CreatePolicyCommand, GetPolicyQuery
    )
    from app.integration.events.event_store import reset_event_store

    reset_event_store()
    bus = CQRSBus()

    # Créer une police
    create_result = await bus.send_command(CreatePolicyCommand(
        customer_id="C001",
        product="AUTO",
        premium=850.0
    ))

    policy_id = create_result["policy_id"]

    # Requête
    query_result = await bus.execute_query(GetPolicyQuery(policy_id=policy_id))

    assert query_result["found"] is True
    assert query_result["policy"]["customer_id"] == "C001"


# Tests contenu théorique
def test_module7_content_files():
    """Test que les fichiers du module 7 existent."""
    base = Path("app/theory/content/07_event_driven")
    assert base.is_dir()
    assert (base / "01_event_types.md").exists()
    assert (base / "02_event_notification.md").exists()
    assert (base / "03_event_sourcing.md").exists()
    assert (base / "04_cqrs.md").exists()
    assert (base / "05_projections.md").exists()


def test_module7_event_sourcing_content():
    """Test le contenu Event Sourcing."""
    content = Path("app/theory/content/07_event_driven/03_event_sourcing.md").read_text(encoding='utf-8')
    assert "event sourcing" in content.lower()
    assert "replay" in content.lower() or "rejouer" in content.lower()
    assert "append" in content.lower()


def test_module7_cqrs_content():
    """Test le contenu CQRS."""
    content = Path("app/theory/content/07_event_driven/04_cqrs.md").read_text(encoding='utf-8')
    assert "cqrs" in content.lower()
    assert "command" in content.lower()
    assert "query" in content.lower()
    assert "projection" in content.lower()
