import pytest
from fastapi.testclient import TestClient
from unittest.mock import patch, MagicMock
from app.main import app

client = TestClient(app)

# --- 1. UNIT & WHITE-BOX TESTS ---

def test_l2_norm():
    """Unit: Tests the mathematical correctness of the L2 Normalization."""
    from app.ml_core import _l2_norm
    import torch
    
    vec = torch.tensor([[3.0, 4.0]]) # 3-4-5 triangle
    normed = _l2_norm(vec)
    
    assert len(normed[0]) == 2
    assert pytest.approx(normed[0][0]) == 0.6 # 3/5
    assert pytest.approx(normed[0][1]) == 0.8 # 4/5

@patch("app.ml_core._qwen_proc")
@patch("app.ml_core._qwen")
@patch("app.ml_core.encode_text")
def test_encode_image_slow_white_box(mock_encode_text, mock_qwen, mock_proc):
    """White-Box: Proves the Slow pipeline formats data correctly."""
    from app.ml_core import encode_image_slow
    from PIL import Image
    import io
    
    img = Image.new('RGB', (100, 100), color='red')
    img_byte_arr = io.BytesIO()
    img.save(img_byte_arr, format='JPEG')
    img_bytes = img_byte_arr.getvalue()

    # Mock the model output to return fake JSON
    mock_qwen.generate.return_value = [[1, 2, 3]]
    mock_proc.batch_decode.return_value = ['{"description": "red", "tags": ["color"]}']
    mock_encode_text.return_value = [[0.1] * 1152] # Fake SigLIP vector

    results = encode_image_slow([img_bytes])
    
    # Assertions
    assert results[0]["description"] == "red"
    assert "color" in results[0]["tags"]
    assert len(results[0]["text_vector"]) == 1152

@patch("app.ml_core._qwen_proc")
@patch("app.ml_core._qwen")
def test_json_fallback_logic(mock_qwen, mock_proc):
    """White-Box: Proves the system gracefully handles bad JSON from the LLM."""
    from app.ml_core import encode_image_slow
    from PIL import Image
    import io
    
    img = Image.new('RGB', (10, 10), color='blue')
    img_byte_arr = io.BytesIO()
    img.save(img_byte_arr, format='JPEG')
    
    mock_qwen.generate.return_value = [[1]]
    mock_proc.batch_decode.return_value = ['INVALID JSON TEXT']
    
    results = encode_image_slow([img_byte_arr.getvalue()])
    
    assert results[0]["description"] == "error"
    assert results[0]["tags"] == []

# --- 2. INTEGRATION TESTS ---

def test_healthz():
    """Integration: Tests that the FastAPI app boots and routes correctly."""
    response = client.get("/healthz")
    assert response.status_code == 200
    assert response.json() == {"ok": True, "status": "healthy"}

@patch("app.main.encode_text")
def test_encode_text_endpoint(mock_encode):
    """Integration: Tests the API payload validation."""
    mock_encode.return_value = [[0.5] * 1152]
    
    response = client.post(
        "/v1/encode/text/",
        json={"texts": ["A beautiful sunset"]}
    )
    
    assert response.status_code == 200
    assert "vectors" in response.json()
    assert len(response.json()["vectors"][0]) == 1152
