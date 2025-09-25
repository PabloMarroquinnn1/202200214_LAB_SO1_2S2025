FROM node:alpine

RUN echo 'const http = require("http");' > /app.js && \
    echo 'const server = http.createServer((req, res) => {' >> /app.js && \
    echo '  res.writeHead(200, {"Content-Type": "text/plain"});' >> /app.js && \
    echo '  res.end("Hola mundo desde contenedor liviano 2\\n");' >> /app.js && \
    echo '});' >> /app.js && \
    echo 'server.listen(3000);' >> /app.js && \
    echo 'console.log("Contenedor liviano 2 ejecutándose en puerto 3000");' >> /app.js && \
    echo 'setInterval(() => {' >> /app.js && \
    echo '  console.log("Ping desde liviano 2 -", new Date());' >> /app.js && \
    echo '}, 60000);' >> /app.js

EXPOSE 3000
CMD ["node", "/app.js"]

