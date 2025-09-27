# app_min.py
import os
import requests
from flask import Flask, request, render_template_string, redirect

ML = os.getenv("ML_URL", "http://ml_service:8003")
VS = os.getenv("VS_URL", "http://vector_service:8002")

TEMPLATE = """
<!doctype html>
<title>Mini CLIP Search</title>
<h2>Add item (text embedding)</h2>
<form method="post" action="/add">
  <input name="id" placeholder="item id (e.g., img_123)" required>
  <input name="text" placeholder="text to embed (e.g., a red car)" required>
  <button>Add</button>
</form>

<h2>Search by text</h2>
<form method="post" action="/search_text">
  <input name="q" placeholder="query text (e.g., red sports car)" required>
  <input name="k" placeholder="k" value="5">
  <button>Search</button>
</form>

<h2>Search by image</h2>
<form method="post" action="/search_image" enctype="multipart/form-data">
  <input type="file" name="file" accept="image/*" required>
  <input name="k" placeholder="k" value="5">
  <button>Search</button>
</form>

{% if results %}
  <h3>Results</h3>
  <ol>
    {% for r in results %}
      <li>{{r}}</li>
    {% endfor %}
  </ol>
{% endif %}
"""

app = Flask(__name__)

@app.route("/", methods=["GET"])
def home():
    return render_template_string(TEMPLATE)

@app.route("/add", methods=["POST"])
def add():
    item_id = request.form["id"]
    text = request.form["text"]
    vec = requests.post(f"{ML}/v1/encode/text", json={"text": text}).json()["vector"]  # /v1/encode/text
    requests.post(f"{VS}/v1/vectors/add", json={"id": item_id, "vector": vec})         # /v1/vectors/add
    return redirect("/")

@app.route("/search_text", methods=["POST"])
def search_text():
    q = request.form["q"]
    k = int(request.form.get("k", 5))
    vec = requests.post(f"{ML}/v1/encode/text", json={"text": q}).json()["vector"]     # /v1/encode/text
    res = requests.post(f"{VS}/v1/vectors/search", json={"vector": vec, "k": k}).json()# /v1/vectors/search
    ids = [str(x) for x in res.get("results", [])]
    return render_template_string(TEMPLATE, results=ids)

@app.route("/search_image", methods=["POST"])
def search_image():
    k = int(request.form.get("k", 5))
    file = request.files["file"]
    vec = requests.post(f"{ML}/v1/encode/image", files={"file": (file.filename, file.stream, file.mimetype)}).json()["vector"]  # /v1/encode/image
    res = requests.post(f"{VS}/v1/vectors/search", json={"vector": vec, "k": k}).json()
    ids = [str(x) for x in res.get("results", [])]
    return render_template_string(TEMPLATE, results=ids)

if __name__ == "__main__":
    app.run("0.0.0.0", 5000, debug=True)
