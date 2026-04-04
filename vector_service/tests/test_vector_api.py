import os
import pytest
import numpy as np
from fastapi.testclient import TestClient
from unittest.mock import patch
from app.main import app

client = TestClient(app)

# ==========================================
# 1. API Validation Tests (Black Box Mocks)
# ==========================================

def test_healthz():
    response = client.get("/healthz")
    assert response.status_code == 200
    assert response.json()["ok"] is True

def test_add_vector_missing_namespace():
    response = client.post("/v1/vectors/add", json={
        "id": 1,
        "vector": [0.1] * 1152
    })
    assert response.status_code == 422 

def test_add_vector_invalid_namespace():
    response = client.post("/v1/vectors/add", json={
        "namespace": "default", 
        "id": 1,
        "vector": [0.1] * 1152
    })
    assert response.status_code == 422

@patch("app.main.add_vector")
def test_add_vector_success(mock_add):
    mock_add.return_value = None 
    response = client.post("/v1/vectors/add", json={
        "namespace": "user_123",
        "id": 42,
        "vector": [0.5] * 1152,
        "normalize": True
    })
    assert response.status_code == 200
    data = response.json()
    assert data["ok"] is True
    
@patch("app.main.search_vectors")
def test_search_vector_success(mock_search):
    mock_search.return_value = [{"id": 42, "score": 0.98}]
    response = client.post("/v1/vectors/search", json={
        "namespace": "user_123",
        "vector": [0.1] * 1152,
        "k": 5
    })
    assert response.status_code == 200
    assert response.json()["results"][0]["id"] == 42

@patch("app.main.delete_vector")
def test_delete_vector(mock_delete):
    mock_delete.return_value = True
    response = client.post("/v1/vectors/delete", json={
        "namespace": "user_123",
        "id": 42
    })
    assert response.status_code == 200
    assert response.json()["deleted"] is True

# ==========================================
# 2. Core Logic & Math Tests (Unit)
# ==========================================

def test_l2_normalization_math():
    from app.core import _normalize
    vec = [3.0, 4.0]
    normed = _normalize(vec)
    assert np.isclose(normed[0][0], 0.6)
    assert np.isclose(normed[0][1], 0.8)

def test_vector_dimension_guardrail():
    from app.core import add_vector
    with pytest.raises(ValueError, match="Vector dimension mismatch"):
        add_vector("user_123", 1, [0.5] * 512)

# ==========================================
# 3. REAL HARDWARE TESTS (Live Milvus Database)
# ==========================================
# This test assumes Milvus is running on localhost:19530 (via docker-compose)

@pytest.mark.skipif(os.getenv("RUN_REAL_MILVUS_TESTS") != "1", reason="Requires local Milvus running via docker-compose")
def test_real_milvus_lifecycle():
    """End-to-End test hitting the live Milvus database."""
    test_namespace = "test_user_integration"
    test_vector = [0.1] * 1152
    
    # 1. Add Vector
    add_resp = client.post("/v1/vectors/add", json={
        "namespace": test_namespace,
        "id": 999,
        "vector": test_vector,
        "normalize": True
    })
    assert add_resp.status_code == 200
    assert add_resp.json()["ok"] is True
    
    # 2. Search Vector
    search_resp = client.post("/v1/vectors/search", json={
        "namespace": test_namespace,
        "vector": test_vector,
        "k": 5
    })
    assert search_resp.status_code == 200
    results = search_resp.json()["results"]
    assert len(results) > 0
    assert results[0]["id"] == 999
    
    # 3. Delete Vector
    del_resp = client.post("/v1/vectors/delete", json={
        "namespace": test_namespace,
        "id": 999
    })
    assert del_resp.status_code == 200
    assert del_resp.json()["deleted"] is True

    # 4. Clear Namespace
    clear_resp = client.delete(f"/v1/namespaces/{test_namespace}")
    assert clear_resp.status_code == 200
    assert clear_resp.json()["ok"] is True
