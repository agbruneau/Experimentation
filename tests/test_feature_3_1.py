"""Tests pour Feature 3.1: Message Broker In-Memory."""
import pytest
import asyncio
from app.integration.events.broker import MessageBroker, MessageStatus


@pytest.mark.asyncio
async def test_queue_point_to_point():
    """Test queue point-à-point basique."""
    broker = MessageBroker()
    await broker.send_to_queue("test", {"id": 1})
    msg = await broker.receive_from_queue("test")
    assert msg is not None
    assert msg.payload["id"] == 1


@pytest.mark.asyncio
async def test_queue_ordering():
    """Test de l'ordre des messages dans une queue."""
    broker = MessageBroker()
    await broker.send_to_queue("order-test", {"seq": 1})
    await broker.send_to_queue("order-test", {"seq": 2})
    await broker.send_to_queue("order-test", {"seq": 3})

    msg1 = await broker.receive_from_queue("order-test")
    msg2 = await broker.receive_from_queue("order-test")
    msg3 = await broker.receive_from_queue("order-test")

    assert msg1.payload["seq"] == 1
    assert msg2.payload["seq"] == 2
    assert msg3.payload["seq"] == 3


@pytest.mark.asyncio
async def test_pubsub_multi():
    """Test pub/sub avec plusieurs abonnés."""
    broker = MessageBroker()
    received = []

    async def handler(msg):
        received.append(msg)

    await broker.subscribe("topic", handler)
    await broker.subscribe("topic", handler)
    await broker.publish("topic", {"data": "test"})

    # Attendre que les handlers soient exécutés
    await asyncio.sleep(0.2)

    assert len(received) == 2
    assert received[0]["data"] == "test"
    assert received[1]["data"] == "test"


@pytest.mark.asyncio
async def test_pubsub_sync_handler():
    """Test pub/sub avec handler synchrone."""
    broker = MessageBroker()
    received = []

    def sync_handler(msg):
        received.append(msg)

    await broker.subscribe("sync-topic", sync_handler)
    await broker.publish("sync-topic", {"value": 42})

    await asyncio.sleep(0.2)

    assert len(received) == 1
    assert received[0]["value"] == 42


@pytest.mark.asyncio
async def test_dlq():
    """Test Dead Letter Queue après échecs répétés."""
    broker = MessageBroker()

    async def fail(m):
        raise Exception("fail")

    await broker.subscribe("flaky", fail, max_retries=1)
    await broker.publish("flaky", {"x": 1})

    # Attendre les retries et l'envoi en DLQ
    await asyncio.sleep(1.0)

    dlq_size = broker.get_dlq_size("flaky")
    assert dlq_size >= 1, "DLQ should have at least one message"

    dlq = await broker.receive_from_dlq("flaky")
    assert dlq is not None
    assert dlq.status == MessageStatus.DEAD_LETTER
    assert dlq.payload["x"] == 1


@pytest.mark.asyncio
async def test_queue_timeout():
    """Test timeout sur queue vide."""
    broker = MessageBroker()
    msg = await broker.receive_from_queue("empty-queue", timeout=0.1)
    assert msg is None


@pytest.mark.asyncio
async def test_broker_stats():
    """Test des statistiques du broker."""
    broker = MessageBroker()

    await broker.send_to_queue("stat-queue", {"test": 1})
    await broker.send_to_queue("stat-queue", {"test": 2})
    await broker.receive_from_queue("stat-queue")

    stats = broker.get_stats()
    assert stats["messages_sent"] == 2
    assert stats["messages_received"] == 1
    assert "stat-queue" in stats["queues"]


@pytest.mark.asyncio
async def test_broker_reset():
    """Test réinitialisation du broker."""
    broker = MessageBroker()

    await broker.send_to_queue("reset-test", {"data": 1})
    await broker.publish("reset-topic", {"data": 2})

    broker.reset()

    assert broker.get_queue_size("reset-test") == 0
    stats = broker.get_stats()
    assert stats["messages_sent"] == 0
    assert len(stats["queues"]) == 0


@pytest.mark.asyncio
async def test_message_history():
    """Test de l'historique des messages."""
    broker = MessageBroker()

    await broker.send_to_queue("history-q", {"msg": 1})
    await broker.publish("history-t", {"msg": 2})

    history = broker.get_message_history()
    assert len(history) == 2


@pytest.mark.asyncio
async def test_unsubscribe():
    """Test désabonnement d'un topic."""
    broker = MessageBroker()
    received = []

    async def handler(msg):
        received.append(msg)

    sub_id = await broker.subscribe("unsub-topic", handler)
    broker.unsubscribe("unsub-topic", sub_id)

    await broker.publish("unsub-topic", {"data": "ignored"})
    await asyncio.sleep(0.2)

    assert len(received) == 0


@pytest.mark.asyncio
async def test_message_headers():
    """Test des headers de message."""
    broker = MessageBroker()

    await broker.send_to_queue(
        "header-queue",
        {"data": "test"},
        source="test-service",
        headers={"correlation-id": "123", "type": "test"}
    )

    msg = await broker.receive_from_queue("header-queue")
    assert msg.source == "test-service"
    assert msg.headers["correlation-id"] == "123"
    assert msg.headers["type"] == "test"


@pytest.mark.asyncio
async def test_api_broker_queues(client):
    """Test API du broker via HTTP."""
    async with client:
        # Envoyer un message
        r = await client.post(
            "/api/broker/queues/api-test/send",
            json={"payload": {"test": "value"}, "source": "test"}
        )
        assert r.status_code == 200
        assert "message" in r.json()

        # Récupérer les stats
        r = await client.get("/api/broker/stats")
        assert r.status_code == 200
        assert r.json()["messages_sent"] >= 1


@pytest.mark.asyncio
async def test_api_broker_topics(client):
    """Test API topics via HTTP."""
    async with client:
        # Publier un message
        r = await client.post(
            "/api/broker/topics/api-topic/publish",
            json={"payload": {"event": "test"}}
        )
        assert r.status_code == 200

        # Lister les topics
        r = await client.get("/api/broker/topics")
        assert r.status_code == 200
