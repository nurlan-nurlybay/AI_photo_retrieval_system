import os
import pytest
from fastapi.testclient import TestClient
from unittest.mock import patch
from app.main import app

client = TestClient(app)

# ==========================================
# 1. API Validation Tests (Mocks)
# ==========================================

@patch("app.core.add_image")
def test_ingest_image_success(mock_add):
    mock_add.return_value = None
    response = client.post("/v1/ingest/image", json={
        "namespace": "test_space",
        "id": 42,
        "image_vector": [0.1] * 1152
    })
    assert response.status_code == 200
    assert response.json()["status"] == "image_added"

@patch("app.core.add_text")
def test_ingest_text_success(mock_add):
    mock_add.return_value = None
    response = client.post("/v1/ingest/text", json={
        "namespace": "test_space",
        "id": 42,
        "text_vector": [0.1] * 1152,
        "tags": ["test", "tags"]
    })
    assert response.status_code == 200
    assert response.json()["status"] == "text_and_tags_added"

@patch("app.core.clear_namespace_data")
def test_clear_namespace(mock_clear):
    mock_clear.return_value = None
    # Using request("DELETE", ...) because httpx handles DELETE with body differently
    response = client.request("DELETE", "/v1/namespace/clear", json={
        "namespace": "test_space"
    })
    assert response.status_code == 200

@patch("app.core.clear_all_namespaces")
def test_nuke_system(mock_nuke):
    mock_nuke.return_value = (5, [])
    response = client.delete("/v1/system/nuke")
    assert response.status_code == 200
    assert response.json()["deleted_count"] == 5

# ==========================================
# 2. REAL HARDWARE TESTS
# ==========================================
@pytest.mark.skipif(os.getenv("RUN_REAL_MILVUS_TESTS") != "1", reason="Requires live Milvus")
def test_real_milvus_lifecycle():
    """End-to-End test hitting the live Milvus database."""
    test_namespace = "test_integration"
    
    # 1. Add Image
    client.post("/v1/ingest/image", json={
        "namespace": test_namespace, "id": 999, "image_vector": [0.1] * 1152
    })
    
    # 2. Add Text
    client.post("/v1/ingest/text", json={
        "namespace": test_namespace, "id": 999, "text_vector": [0.1] * 1152, "tags": ["test"]
    })
    
    # 3. Hybrid Search
    search_resp = client.post("/v1/search/hybrid", json={
        "namespace": test_namespace,
        "query_text": "test",
        "image_vector": [0.1] * 1152,
        "text_vector": [0.1] * 1152,
        "top_k": 5
    })
    assert search_resp.status_code == 200
    assert len(search_resp.json()) > 0
    assert search_resp.json()[0]["id"] == 999
    
    # 4. Clear
    client.request("DELETE", "/v1/namespace/clear", json={"namespace": test_namespace})

