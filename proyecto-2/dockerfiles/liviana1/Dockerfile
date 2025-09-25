FROM alpine:latest

RUN apk add --no-cache curl

RUN echo '#!/bin/sh' > /app.sh && \
    echo 'echo "Contenedor liviano 1 iniciado - $(date)"' >> /app.sh && \
    echo 'while true; do' >> /app.sh && \
    echo '  echo "Hola mundo desde contenedor liviano 1 - $(date)"' >> /app.sh && \
    echo '  sleep 300' >> /app.sh && \
    echo 'done' >> /app.sh && \
    chmod +x /app.sh

CMD ["/app.sh"]
