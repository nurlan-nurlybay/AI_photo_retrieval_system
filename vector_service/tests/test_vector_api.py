import os
import pytest
from fastapi.testclient import TestClient
from unittest.mock import patch

# Mock env vars BEFORE importing the app to bypass strict config.py checks
os.environ["MILVUS_HOST"] = "mock_host"
os.environ["MILVUS_PORT"] = "19530"
os.environ["VECTOR_DIM"] = "1152"
os.environ["PORT"] = "8002"

from app.main import app

client = TestClient(app)

# ==========================================
# API Validation Tests (Mocks)
# ==========================================

@patch("app.core.add_image_batch")
def test_ingest_image_batch_success(mock_add):
    mock_add.return_value = None
    response = client.post("/v1/ingest/image", json={
        "namespace": "test_space",
        "items": [{"id": 42, "image_vector": [0.1] * 1152}]
    })
    assert response.status_code == 200
    assert "inserted 1 images" in response.json()["status"]

@patch("app.core.add_text_batch")
def test_ingest_text_batch_success(mock_add):
    mock_add.return_value = None
    response = client.post("/v1/ingest/text", json={
        "namespace": "test_space",
        "items": [{"id": 42, "text_vector": [0.1] * 1152, "tags": ["test", "tags"]}]
    })
    assert response.status_code == 200
    assert "inserted 1 texts" in response.json()["status"]

@patch("app.core.search_collection")
@patch("app.core.check_sync_status")
def test_hybrid_search(mock_sync, mock_search):
    mock_sync.return_value = True  # Trigger Qwen fusion
    
    class MockHit:
        def __init__(self, id, distance, tags=None):
            self.id = id
            self.distance = distance
            self.entity = {"tags": tags} if tags else {}
            
        def get(self, key):
            # Needed for main.py hit.entity.get("tags")
            return self.entity.get(key)
    
    # 1st call returns image search results, 2nd call returns text search results
    mock_search.side_effect = [
        [MockHit(42, 0.9)],
        [MockHit(42, 0.85, '["test"]')]
    ]
    
    response = client.post("/v1/search/hybrid", json={
        "namespace": "test_space",
        "query_text": "test query",
        "image_vector": [0.1] * 1152,
        "text_vector": [0.1] * 1152,
        "top_k": 3
    })
    
    assert response.status_code == 200
    data = response.json()
    assert data["used_qwen"] is True
    assert len(data["results"]) == 1
    assert data["results"][0]["id"] == 42

@patch("app.core.clear_namespace_data")
def test_clear_namespace(mock_clear):
    mock_clear.return_value = None
    # Updated to match @app.post("/v1/admin/clear/{namespace}")
    response = client.post("/v1/admin/clear/test_space")
    assert response.status_code == 200
    assert "cleared" in response.json()["status"]

@patch("app.core.clear_all_namespaces")
def test_nuke_system(mock_nuke):
    mock_nuke.return_value = (5, [])
    # Updated to match @app.post("/v1/admin/nuke")
    response = client.post("/v1/admin/nuke")
    assert response.status_code == 200
    assert response.json()["deleted_collections"] == 5
