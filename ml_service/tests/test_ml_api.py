import os
import sys
from unittest.mock import patch, MagicMock

# =================================================================
# 0. BYPASS HEAVY IMPORTS FOR CI/CD
# =================================================================
# We mock these libraries so Python doesn't crash when app.main 
# tries to import ml_core.py in our GitHub Actions environment.
sys.modules['torch'] = MagicMock()
sys.modules['transformers'] = MagicMock()
sys.modules['qwen_vl_utils'] = MagicMock()
sys.modules['pillow_heif'] = MagicMock()

import pytest
from fastapi.testclient import TestClient
import io
from PIL import Image

# Now it is safe to import the app
from app.main import app

client = TestClient(app)

def get_test_image_bytes():
    img = Image.new('RGB', (224, 224), color='red')
    img_byte_arr = io.BytesIO()
    img.save(img_byte_arr, format='JPEG')
    return img_byte_arr.getvalue()

# =================================================================
# 1. LIGHTWEIGHT API TESTS (Mocks - Runs fast in GitHub CI)
# =================================================================

def test_healthz():
    response = client.get("/healthz")
    assert response.status_code == 200

@patch("app.main.encode_text")
def test_mock_encode_text(mock_encode):
    mock_encode.return_value = [[0.1] * 1152]
    response = client.post("/v1/encode/text/", json={"texts": ["Test"]})
    assert response.status_code == 200
    assert len(response.json()["vectors"][0]) == 1152

@patch("app.main.encode_image_fast")
def test_mock_encode_image_fast(mock_fast):
    mock_fast.return_value = [[0.5] * 1152]
    response = client.post(
        "/v1/encode/image/fast/",
        files=[("files", ("test.jpg", get_test_image_bytes(), "image/jpeg"))]
    )
    assert response.status_code == 200
    assert len(response.json()["vectors"][0]) == 1152

@patch("app.main.encode_image_slow")
def test_mock_encode_image_slow(mock_slow):
    mock_slow.return_value = [{
        "description": "mocked", "tags": ["test"],
        "image_vector": [0.5] * 1152, "text_vector": [0.1] * 1152
    }]
    response = client.post(
        "/v1/encode/image/slow/",
        files=[("files", ("test.jpg", get_test_image_bytes(), "image/jpeg"))]
    )
    assert response.status_code == 200
    assert response.json()["results"][0]["description"] == "mocked"

# =================================================================
# 2. REAL HARDWARE TESTS (Runs only on your Laptop via ENV variable)
# =================================================================

@pytest.mark.skipif(os.getenv("RUN_REAL_GPU_TESTS") != "1", reason="Requires local GPU")
def test_real_fast_endpoint():
    print("\nTesting /fast endpoint...")
    response = client.post("/v1/encode/image/fast/", files=[("files", ("img.png", get_test_image_bytes(), "image/png"))])
    assert response.status_code == 200
    assert len(response.json()["vectors"][0]) == 1152

@pytest.mark.skipif(os.getenv("RUN_REAL_GPU_TESTS") != "1", reason="Requires local GPU")
def test_real_slow_endpoint():
    print("\nTesting /slow endpoint... (This will lazy-load Qwen)")
    response = client.post("/v1/encode/image/slow/", files=[("files", ("img.png", get_test_image_bytes(), "image/png"))])
    assert response.status_code == 200
    results = response.json()["results"][0]
    assert "description" in results
    assert len(results["text_vector"]) == 1152

@pytest.mark.skipif(os.getenv("RUN_REAL_GPU_TESTS") != "1", reason="Requires local GPU")
def test_real_text_endpoint():
    print("\nTesting /text endpoint...")
    response = client.post("/v1/encode/text/", json={"texts": ["A photo of a dog"]})
    assert response.status_code == 200
    assert len(response.json()["vectors"][0]) == 1152

