"""API de contrôle du Message Broker."""
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from typing import Dict, Any, List, Optional

from app.integration.events.broker import get_broker, Message


router = APIRouter()


class SendMessageRequest(BaseModel):
    """Requête pour envoyer un message."""
    payload: Dict[str, Any]
    source: Optional[str] = ""
    headers: Optional[Dict[str, str]] = None


class PublishMessageRequest(BaseModel):
    """Requête pour publier un message sur un topic."""
    payload: Dict[str, Any]
    source: Optional[str] = ""
    headers: Optional[Dict[str, str]] = None


# ========== QUEUES ==========

@router.get("/queues")
async def list_queues():
    """Liste toutes les queues."""
    broker = get_broker()
    return {"queues": broker.get_queues()}


@router.post("/queues/{queue_name}/send")
async def send_to_queue(queue_name: str, request: SendMessageRequest):
    """Envoie un message dans une queue."""
    broker = get_broker()
    message = await broker.send_to_queue(
        queue_name=queue_name,
        payload=request.payload,
        source=request.source,
        headers=request.headers
    )
    return {"message": message.to_dict()}


@router.get("/queues/{queue_name}/receive")
async def receive_from_queue(queue_name: str, timeout: float = 5.0):
    """Reçoit un message d'une queue."""
    broker = get_broker()
    message = await broker.receive_from_queue(queue_name, timeout=timeout)
    if message:
        return {"message": message.to_dict()}
    return {"message": None, "info": "No message available"}


@router.get("/queues/{queue_name}/messages")
async def get_queue_messages(queue_name: str, limit: int = 50):
    """Récupère les messages d'une queue."""
    broker = get_broker()
    return {"messages": broker.get_queue_messages(queue_name, limit)}


@router.get("/queues/{queue_name}/size")
async def get_queue_size(queue_name: str):
    """Récupère la taille d'une queue."""
    broker = get_broker()
    return {"queue": queue_name, "size": broker.get_queue_size(queue_name)}


# ========== TOPICS ==========

@router.get("/topics")
async def list_topics():
    """Liste tous les topics."""
    broker = get_broker()
    return {"topics": broker.get_topics()}


@router.post("/topics/{topic_name}/publish")
async def publish_to_topic(topic_name: str, request: PublishMessageRequest):
    """Publie un message sur un topic."""
    broker = get_broker()
    message = await broker.publish(
        topic_name=topic_name,
        payload=request.payload,
        source=request.source,
        headers=request.headers
    )
    return {"message": message.to_dict()}


# ========== DEAD LETTER QUEUES ==========

@router.get("/dlq/{source_name}")
async def get_dlq(source_name: str, limit: int = 50):
    """Récupère les messages d'une DLQ."""
    broker = get_broker()
    return {
        "dlq": f"{source_name}.dlq",
        "size": broker.get_dlq_size(source_name),
        "messages": broker.get_dlq_messages(source_name, limit)
    }


@router.post("/dlq/{source_name}/receive")
async def receive_from_dlq(source_name: str):
    """Récupère un message de la DLQ."""
    broker = get_broker()
    message = await broker.receive_from_dlq(source_name)
    if message:
        return {"message": message.to_dict()}
    return {"message": None, "info": "DLQ is empty"}


# ========== MÉTRIQUES ET CONTRÔLE ==========

@router.get("/stats")
async def get_stats():
    """Récupère les statistiques du broker."""
    broker = get_broker()
    return broker.get_stats()


@router.get("/history")
async def get_history(limit: int = 100):
    """Récupère l'historique des messages."""
    broker = get_broker()
    return {"messages": broker.get_message_history(limit)}


@router.post("/reset")
async def reset_broker():
    """Réinitialise le broker."""
    broker = get_broker()
    broker.reset()
    return {"status": "ok", "message": "Broker reset successfully"}
