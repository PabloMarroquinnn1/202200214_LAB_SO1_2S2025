FROM python:3.9-slim

RUN apt-get update && apt-get install -y htop && rm -rf /var/lib/apt/lists/*

COPY cpu_stress.py /cpu_stress.py
RUN chmod +x /cpu_stress.py

ENV PYTHONUNBUFFERED=1

CMD ["python3", "/cpu_stress.py"]
