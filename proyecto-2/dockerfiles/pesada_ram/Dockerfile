FROM python:3.9-slim

RUN apt-get update && apt-get install -y htop && rm -rf /var/lib/apt/lists/*

COPY ram_stress.py /ram_stress.py
RUN chmod +x /ram_stress.py

ENV PYTHONUNBUFFERED=1

CMD ["python3", "/ram_stress.py"]
