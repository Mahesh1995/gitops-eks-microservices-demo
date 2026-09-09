FROM python:3.12-slim
WORKDIR /app

# Install deps first for better layer caching.
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Run as a non-root user.
RUN useradd --uid 10001 --no-create-home appuser
USER appuser

EXPOSE 5000
# gunicorn for a production-ish WSGI server (2 workers is plenty for a demo).
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "--workers", "2", "app:app"]
