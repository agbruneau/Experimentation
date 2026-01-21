"""Tests pour Feature 3.5: Module 8 - Saga & Outbox."""
import pytest
from pathlib import Path
from app.integration.events.saga import SagaOrchestrator, SagaStatus, SubscriptionSaga
from app.integration.events.outbox import OutboxProcessor, OutboxStatus


@pytest.mark.asyncio
async def test_saga_basic_execution():
    """Test exécution basique d'une saga."""
    saga = SagaOrchestrator()

    async def step1(ctx):
        return {"step1": "done"}

    async def step2(ctx):
        return {"step2": "done"}

    saga.add_step(step1, name="step1")
    saga.add_step(step2, name="step2")

    result = await saga.execute({"initial": "value"})

    assert result["status"] == "COMPLETED"
    assert "step1" in result["context"]
    assert "step2" in result["context"]


@pytest.mark.asyncio
async def test_saga_compensation_on_failure():
    """Test compensation automatique en cas d'échec."""
    saga = SagaOrchestrator()
    compensated = []

    async def step1(ctx):
        return {"step1_done": True}

    async def compensate1(ctx):
        compensated.append("step1")

    async def step2(ctx):
        raise Exception("Step 2 failed")

    saga.add_step(step1, compensate=compensate1, name="step1")
    saga.add_step(step2, name="step2")

    result = await saga.execute({})

    assert result["status"] == "COMPENSATED"
    assert "step1" in compensated


@pytest.mark.asyncio
async def test_subscription_saga():
    """Test de la saga de souscription."""
    saga = SubscriptionSaga()

    result = await saga.execute({
        "quote_id": "Q001",
        "customer_id": "C001"
    })

    assert result["status"] == "COMPLETED"
    assert "policy_id" in result["context"]
    assert "invoice_id" in result["context"]


@pytest.mark.asyncio
async def test_saga_event_tracking():
    """Test du tracking des événements de saga."""
    saga = SagaOrchestrator()
    events = []

    saga.on_event(lambda e: events.append(e))

    async def step1(ctx):
        return {}

    saga.add_step(step1, name="step1")
    await saga.execute({})

    event_types = [e["type"] for e in events]
    assert "saga_started" in event_types
    assert "step_started" in event_types
    assert "step_completed" in event_types
    assert "saga_completed" in event_types


# Tests Outbox
@pytest.mark.asyncio
async def test_outbox_add_entry():
    """Test ajout d'entrée dans l'outbox."""
    outbox = OutboxProcessor()

    entry = await outbox.add_entry(
        aggregate_type="Policy",
        aggregate_id="POL-001",
        event_type="PolicyCreated",
        payload={"premium": 850}
    )

    assert entry.status == OutboxStatus.PENDING
    assert entry.event_type == "PolicyCreated"


@pytest.mark.asyncio
async def test_outbox_process_pending():
    """Test traitement des entrées en attente."""
    published = []

    async def mock_publisher(msg):
        published.append(msg)

    outbox = OutboxProcessor(publisher=mock_publisher)

    await outbox.add_entry(
        aggregate_type="Policy",
        aggregate_id="POL-001",
        event_type="PolicyCreated",
        payload={"premium": 850}
    )

    processed = await outbox.process_pending()

    assert processed == 1
    assert len(published) == 1
    assert published[0]["type"] == "PolicyCreated"


@pytest.mark.asyncio
async def test_outbox_stats():
    """Test des statistiques de l'outbox."""
    outbox = OutboxProcessor()

    await outbox.add_entry("Policy", "P1", "Created", {})
    await outbox.add_entry("Policy", "P2", "Created", {})

    stats = outbox.get_stats()

    assert stats["total"] == 2
    assert stats["pending"] == 2


@pytest.mark.asyncio
async def test_outbox_retry_on_failure():
    """Test retry après échec."""
    call_count = [0]

    async def failing_publisher(msg):
        call_count[0] += 1
        if call_count[0] < 2:
            raise Exception("Network error")

    outbox = OutboxProcessor(publisher=failing_publisher)

    await outbox.add_entry("Policy", "P1", "Created", {})

    # Premier traitement - échoue
    await outbox.process_pending()

    # L'entrée est toujours pending pour retry
    pending = outbox.get_pending_entries()
    assert len(pending) == 1
    assert pending[0].retries == 1


@pytest.mark.asyncio
async def test_outbox_event_notifications():
    """Test des notifications d'événements outbox."""
    events = []

    outbox = OutboxProcessor()
    outbox.on_event(lambda e: events.append(e))

    await outbox.add_entry("Policy", "P1", "Created", {})

    event_types = [e["type"] for e in events]
    assert "outbox_entry_added" in event_types


# Tests contenu théorique
def test_module8_content_files():
    """Test que les fichiers du module 8 existent."""
    base = Path("app/theory/content/08_saga_transactions")
    assert base.is_dir()
    assert (base / "01_distributed_transactions.md").exists()
    assert (base / "02_saga_orchestration.md").exists()
    assert (base / "03_saga_choreography.md").exists()
    assert (base / "04_outbox_pattern.md").exists()
    assert (base / "05_compensation.md").exists()


def test_module8_saga_content():
    """Test le contenu Saga."""
    content = Path("app/theory/content/08_saga_transactions/02_saga_orchestration.md").read_text(encoding='utf-8')
    assert "saga" in content.lower()
    assert "orchestrat" in content.lower()
    assert "compensat" in content.lower()


def test_module8_outbox_content():
    """Test le contenu Outbox."""
    content = Path("app/theory/content/08_saga_transactions/04_outbox_pattern.md").read_text(encoding='utf-8')
    assert "outbox" in content.lower()
    assert "atomic" in content.lower() or "atomique" in content.lower()
    assert "polling" in content.lower() or "cdc" in content.lower()


@pytest.mark.asyncio
async def test_scenarios_evt03_to_07(client):
    """Test que les scénarios EVT-03 à EVT-07 existent."""
    async with client:
        for scenario_id in ["EVT-03", "EVT-04", "EVT-05", "EVT-06", "EVT-07"]:
            r = await client.get(f"/api/sandbox/scenarios/{scenario_id}")
            assert r.status_code == 200
            data = r.json()
            assert "steps" in data
            assert data["pillar"] == "events"
