# temp_webapp/Dockerfile
FROM python:3.12-slim

# No heavy system deps; add curl if you want container-side tests
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install webapp deps (Flask, requests, etc.)
COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy the web app code
# If your main file is app_min.py (as in our example), copy the whole folder.
COPY . /app

# The app must bind 0.0.0.0 so Docker can expose it
EXPOSE 5000
# If your entry is app_min.py with Flask's built-in server:
#CMD ["python", "app_min.py"]

# Better: run Flask via gunicorn (production-ish). Adjust module name if different.
# e.g., if your Flask app instance is created in app_min.py as "app":
# gunicorn will serve it as app_min:app
RUN pip install --no-cache-dir gunicorn
CMD ["gunicorn", "-b", "0.0.0.0:5000", "app_min:app"]
