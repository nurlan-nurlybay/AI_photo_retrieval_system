import pytest
from fastapi.testclient import TestClient
from unittest.mock import patch
from app.main import app
import numpy as np

client = TestClient(app)

# ==========================================
# 1. API Validation Tests (Black Box)
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

# ==========================================
# 2. Integration & White Box Tests
# ==========================================

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
    assert data["namespace"] == "user_123"
    assert data["id"] == 42
    
    mock_add.assert_called_once_with("user_123", 42, [0.5]*1152, True)

@patch("app.main.search_vectors")
def test_search_vector_success(mock_search):
    mock_search.return_value = [{"id": 42, "score": 0.98}]
    
    response = client.post("/v1/vectors/search", json={
        "namespace": "user_123",
        "vector": [0.1] * 1152,
        "k": 5
    })
    
    assert response.status_code == 200
    data = response.json()
    assert data["ok"] is True
    assert len(data["results"]) == 1
    assert data["results"][0]["id"] == 42

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
# 3. Core Logic & Math Tests (Unit)
# ==========================================

def test_l2_normalization_math():
    from app.core import _normalize
    vec = [3.0, 4.0]
    normed = _normalize(vec)
    
    assert np.isclose(normed[0][0], 0.6)
    assert np.isclose(normed[0][1], 0.8)
    
    zero_vec = [0.0, 0.0]
    zero_normed = _normalize(zero_vec)
    assert np.isclose(zero_normed[0][0], 0.0)

def test_vector_dimension_guardrail():
    from app.core import add_vector
    import pytest
    
    with pytest.raises(ValueError, match="Vector dimension mismatch"):
        add_vector("user_123", 1, [0.5] * 512)
