# Manual Técnico - Sistema de Monitoreo de Contenedores
**Carné: 202200214**  
**Curso: Sistemas Operativos 1**  
**Proyecto 2**
**Nombre: Pablo Alejandro Marroquin Cutz**


## Tabla de Contenidos
1. [Introducción](#introducción)
2. [Arquitectura del Sistema](#arquitectura-del-sistema)
3. [Módulos de Kernel](#módulos-de-kernel)
4. [Daemon en Go](#daemon-en-go)
5. [Sistema de Contenedores](#sistema-de-contenedores)
6. [Dashboard Grafana](#dashboard-grafana)
7. [Compilación e Instalación](#compilación-e-instalación)


## Introducción

Este proyecto implementa un sistema integral de monitoreo y gestión automática de contenedores Docker en Linux, compuesto por:
- **Dos módulos de kernel en C** para captura de métricas en tiempo real
- **Un daemon en Go** que actúa como orquestador del sistema
- **Scripts de automatización** para generación y gestión de contenedores
- **Dashboard en Grafana** para visualización de métricas

## Arquitectura del Sistema

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Módulos       │    │   Daemon Go      │    │   Grafana       │
│   Kernel        │    │                  │    │   Dashboard     │
│                 │    │  ┌─────────────┐ │    │                 │
│ continfo_so1    ├────┤  │ Analizador  │ ├────┤  Visualización  │
│ sysinfo_so1     │    │  │ Decisor     │ │    │  de Métricas    │
│                 │    │  │ SQLite DB   │ │    │                 │
└─────────────────┘    │  └─────────────┘ │    └─────────────────┘
         │              │                  │              │
         │              │  ┌─────────────┐ │              │
         │              │  │ Cronjob     │ │              │
         ▼              │  │ Manager     │ │              │
┌─────────────────┐    │  └─────────────┘ │              │
│   /proc/        │    │                  │              │
│   continfo_...  │    │  ┌─────────────┐ │              │
│   sysinfo_...   │◄───┤  │ Container   │ │              │
└─────────────────┘    │  │ Manager     │ │              │
                       │  └─────────────┘ │              │
                       └──────────────────┘              │
                              │                          │
                              ▼                          │
                       ┌──────────────────┐              │
                       │   Docker         │              │
                       │   Containers     │──────────────┘
                       └──────────────────┘
```

## Módulos de Kernel

### 1. Módulo continfo_so1_202200214.c

**Propósito**: Monitorea específicamente los procesos relacionados con contenedores Docker.

#### Funciones Principales:

##### `is_container_process(struct task_struct *task)`
```c
static int is_container_process(struct task_struct *task) {
    char cmdline[MAX_CMDLINE_LENGTH];
    int len;
    
    len = read_process_cmdline(task, cmdline, sizeof(cmdline));
    
    if (len > 0) {
        if (strstr(cmdline, "/cpu_stress.py") ||
            strstr(cmdline, "/ram_stress.py") ||
            strstr(cmdline, "/app.js") ||
            strstr(cmdline, "/app.sh")) {
            return 1;
        }
    }
    return 0;
}
```
**Funcionalidad**: Identifica procesos de contenedores basándose en patrones específicos en la línea de comandos.

##### `read_process_cmdline(struct task_struct *task, char *buffer, int max_len)`
```c
static int read_process_cmdline(struct task_struct *task, char *buffer, int max_len) {
    struct mm_struct *mm;
    char *cmdline_buffer;
    int len = 0;
    
    mm = get_task_mm(task);
    if (!mm) return 0;
    
    cmdline_buffer = kmalloc(max_len, GFP_KERNEL);
    len = access_process_vm(task, mm->arg_start, cmdline_buffer,
                           min_t(int, max_len - 1, mm->arg_end - mm->arg_start), 0);
    
    // Procesamiento y limpieza...
    return len;
}
```
**Funcionalidad**: Extrae la línea de comandos completa de un proceso accediendo directamente al espacio de memoria virtual.

##### `extract_container_id(char *cmdline, char *container_id, int max_len, int pid)`
Extrae identificadores de contenedores usando patrones como:
- `ctn_` prefix para contenedores del proyecto
- `--name` parameter en líneas de comando de Docker
- Fallback a `container_PID` si no se encuentra patrón

#### Salida JSON:
```json
{
  "totalram": 8388608,
  "freeram": 2097152,
  "usedram": 6291456,
  "total_containers": 5,
  "containers": [
    {
      "pid": 1234,
      "name": "python3",
      "cmdline": "python3 /cpu_stress.py",
      "container_id": "ctn_pesada_cpu_1_1695840123",
      "vsz": 102400,
      "rss": 51200,
      "mem_perc": 61,
      "cpu_perc": 850
    }
  ]
}
```

### 2. Módulo sysinfo_so1_202200214.c

**Propósito**: Monitorea todos los procesos del sistema operativo.

#### Funciones Principales:

##### `sysinfo_show(struct seq_file *m, void *v)`
- Itera sobre todos los procesos del sistema usando `for_each_process(task)`
- Calcula métricas de memoria y CPU con precisión de décimas
- Incluye el campo `state` para mostrar el estado del proceso

#### Cálculo de Métricas:
```c
// Memoria: precisión en décimas de porcentaje
mem_perc = (rss * 1000) / totalram;

// CPU: basado en tiempo de ejecución
unsigned long total_time = task->utime + task->stime;
unsigned long uptime = jiffies - task->start_time;
cpu_perc = (total_time * 1000) / uptime;
```

#### Diferencias con continfo:
- **Alcance**: Todos los procesos vs solo contenedores
- **Estado**: Incluye campo `state` con el estado del proceso (R, S, D, etc.)
- **Filtrado**: Sin filtrado específico de procesos

## Daemon en Go

### Arquitectura del Daemon

El daemon está estructurado en componentes modulares:

```go
type Daemon struct {
    db                    *sql.DB
    containersEliminated  int
    lowConsumptionCount   int
    highConsumptionCount  int
}
```

### Componentes Principales:

#### 1. Inicialización (`NewDaemon()`)
```go
func NewDaemon() (*Daemon, error) {
    dbPath := "./metrics.db"
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("error opening database: %v", err)
    }
    
    daemon := &Daemon{db: db}
    if err := daemon.initDB(); err != nil {
        db.Close()
        return nil, err
    }
    return daemon, nil
}
```

#### 2. Gestión de Base de Datos (`initDB()`)
Crea las siguientes tablas en SQLite:
- `ram_usage`: Uso de memoria del sistema
- `top5_ram`: Top 5 contenedores por consumo RAM
- `top5_cpu`: Top 5 contenedores por consumo CPU  
- `top5_ram_low`: Top 5 contenedores con menor consumo
- `containers_killed`: Contador de contenedores eliminados
- `system_processes`: Estadísticas de procesos del sistema

#### 3. Lectura de Datos del Kernel
```go
func (d *Daemon) readContainerInfo() (*ContInfo, error) {
    data, err := os.ReadFile("/proc/continfo_so1_202200214")
    if err != nil {
        return nil, fmt.Errorf("error reading continfo: %v", err)
    }
    
    var info ContInfo
    if err := json.Unmarshal(data, &info); err != nil {
        return nil, fmt.Errorf("error unmarshaling continfo JSON: %v", err)
    }
    return &info, nil
}
```

#### 4. Sistema de Análisis y Decisión (`analyzeAndManageContainers()`)

##### Mapeo de PIDs a Nombres Docker:
```go
func (d *Daemon) buildDockerPIDMap() map[int]string {
    pidToName := make(map[int]string)
    
    cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
    output, err := cmd.Output()
    
    names := strings.Split(strings.TrimSpace(string(output)), "\n")
    for _, name := range names {
        pidCmd := exec.Command("docker", "inspect", name, "--format", "{{.State.Pid}}")
        pidOutput, err := pidCmd.Output()
        if pid, err := strconv.Atoi(strings.TrimSpace(string(pidOutput))); err == nil {
            pidToName[pid] = name
        }
    }
    return pidToName
}
```

##### Clasificación de Contenedores:
```go
func (d *Daemon) identifyContainerType(container ContainerProc) string {
    cmdline := strings.ToLower(container.Cmdline)
    
    if strings.Contains(cmdline, "/app.sh") {
        return "liviana1"
    }
    if strings.Contains(cmdline, "/app.js") {
        return "liviana2"  
    }
    if strings.Contains(cmdline, "/cpu_stress.py") {
        return "pesada_cpu"
    }
    if strings.Contains(cmdline, "/ram_stress.py") {
        return "pesada_ram"
    }
    return "unknown"
}
```

##### Reglas de Gestión:
1. **Contenedores Livianos**: Máximo 3 permitidos
2. **Contenedores Pesados**: Máximo 2 permitidos  
3. **Contenedores Protegidos**: Nunca eliminar Grafana, clever_black, etc.

##### Algoritmo de Eliminación:
```go
// REGLA 1: EXACTAMENTE 3 contenedores livianos
if len(lightContainers) > 3 {
    excessLight := lightContainers[3:]
    d.killContainersByType(excessLight, "liviano (mantener solo 3)")
}

// REGLA 2: EXACTAMENTE 2 contenedores pesados
totalHeavy := len(heavyCPUContainers) + len(heavyRAMContainers)
if totalHeavy > 2 {
    var allHeavy []ContainerProc
    allHeavy = append(allHeavy, heavyCPUContainers...)
    allHeavy = append(allHeavy, heavyRAMContainers...)
    
    // Ordenar por consumo MENOR (mantener los más eficientes)
    sort.Slice(allHeavy, func(i, j int) bool {
        scoreI := float64(allHeavy[i].Rss)/1000 + float64(allHeavy[i].CpuPerc)
        scoreJ := float64(allHeavy[j].Rss)/1000 + float64(allHeavy[j].CpuPerc)
        return scoreI < scoreJ
    })
    
    excessHeavy := allHeavy[2:]
    d.killContainersByType(excessHeavy, "pesado (mantener solo 2)")
}
```

#### 5. Gestión del Cronjob (`setupCronJob()`)
```go
func (d *Daemon) setupCronJob() error {
    cronEntry := "*/3 * * * * /home/pablo/proyecto-2/bash/create_containers.sh >/dev/null 2>&1"
    cmd := exec.Command("bash", "-c", 
        fmt.Sprintf("(crontab -l 2>/dev/null; echo '%s') | crontab -", cronEntry))
    return cmd.Run()
}
```

#### 6. Loop Principal (`mainLoop()`)
Se ejecuta cada 20 segundos:
1. Lee datos de `/proc/continfo_so1_202200214` y `/proc/sysinfo_so1_202200214`
2. Analiza contenedores y aplica reglas de gestión
3. Guarda métricas en SQLite
4. Registra estadísticas y acciones tomadas

## Sistema de Contenedores

### Script create_containers.sh

#### Algoritmo de Generación:
```bash
for i in {1..10}; do
    if [ $i -le 3 ]; then
        # Primeros 3: contenedores livianos alternando
        if [ $((i % 2)) -eq 1 ]; then
            IMAGE=$LIVIANA1; TYPE="liviana1"
        else
            IMAGE=$LIVIANA2; TYPE="liviana2"
        fi
    elif [ $i -le 5 ]; then
        # 4to y 5to: contenedores pesados
        if [ $i -eq 4 ]; then
            IMAGE=$PESADA_CPU; TYPE="pesada_cpu"
        else
            IMAGE=$PESADA_RAM; TYPE="pesada_ram"
        fi
    else
        # Resto: aleatorio
        RAND_INDEX=$((RANDOM % 4))
        IMAGE=${IMAGES[$RAND_INDEX]}
        TYPE=${TYPES[$RAND_INDEX]}
    fi
done
```

#### Límites de Recursos por Tipo:
```bash
case "$TYPE" in
    "pesada_cpu")
        docker run -d --rm --name "$NAME" --memory="256m" --cpus="1.5" "$IMAGE"
        ;;
    "pesada_ram")  
        docker run -d --rm --name "$NAME" --memory="512m" --cpus="0.5" "$IMAGE"
        ;;
    "liviana1")
        docker run -d --rm --name "$NAME" --memory="64m" --cpus="0.3" "$IMAGE"
        ;;
    "liviana2")
        PORT=$((3000 + i + RANDOM % 1000))
        docker run -d --rm --name "$NAME" --memory="128m" --cpus="0.3" -p "${PORT}:3000" "$IMAGE"
        ;;
esac
```

### Dockerfiles

#### 1. Contenedor Liviano 1 (Alpine + Shell Script)
```dockerfile
FROM alpine:latest
RUN apk add --no-cache curl
RUN echo '#!/bin/sh' > /app.sh && \
    echo 'while true; do' >> /app.sh && \
    echo ' echo "Hola mundo desde contenedor liviano 1 - $(date)"' >> /app.sh && \
    echo ' sleep 300' >> /app.sh && \
    echo 'done' >> /app.sh && \
    chmod +x /app.sh
CMD ["/app.sh"]
```

#### 2. Contenedor Liviano 2 (Node.js HTTP Server)
```dockerfile
FROM node:alpine
RUN echo 'const http = require("http");' > /app.js && \
    echo 'const server = http.createServer((req, res) => {' >> /app.js && \
    echo ' res.writeHead(200, {"Content-Type": "text/plain"});' >> /app.js && \
    echo ' res.end("Hola mundo desde contenedor liviano 2\\n");' >> /app.js && \
    echo '});' >> /app.js && \
    echo 'server.listen(3000);' >> /app.js
EXPOSE 3000
CMD ["node", "/app.js"]
```

#### 3. Contenedor Pesado CPU (Python Multi-threading)
```python
def cpu_stress():
    while True:
        result = 0
        for i in range(1000000):
            result += i ** 2
        result = result % 999999
        for i in range(100000):
            result += (i * 3.14159) ** 0.5
        time.sleep(0.001)

num_threads = os.cpu_count() or 4
for i in range(num_threads):
    thread = threading.Thread(target=cpu_stress)
    thread.daemon = True
    thread.start()
```

#### 4. Contenedor Pesado RAM (Python Memory Allocation)
```python
def create_memory_chunks():
    chunks = []
    chunk_size = 50 * 1024 * 1024  # 50MB por chunk
    
    while True:
        chunk = bytearray(chunk_size)
        for i in range(0, chunk_size, 4096):
            chunk[i] = i % 256
        chunks.append(chunk)
        time.sleep(2)
```

## Dashboard Grafana

### Configuración (docker-compose.yml)
```yaml
version: '3.8'
services:
  grafana:
    image: grafana/grafana:latest
    container_name: grafana-so1-202200214
    restart: unless-stopped
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana
      - ../go-daemon/metrics.db:/var/lib/grafana/metrics.db:ro
    networks:
      - monitoring
```

### Datasource Configuration (datasources.yml)
```yaml
apiVersion: 1
datasources:
- name: SQLite_Metrics
  type: frser-sqlite-datasource
  access: proxy
  url: file:/var/lib/grafana/metrics.db
  isDefault: true
```

### Paneles Implementados:
1. **Total de RAM**: Gauge mostrando RAM total del sistema
2. **Memoria Libre**: Gauge con memoria disponible  
3. **Contenedores Eliminados**: Time series con contador acumulativo
4. **Uso de RAM**: Time series del consumo de memoria temporal
5. **Top 5 RAM**: Pie chart de contenedores con mayor consumo
6. **Top 5 CPU**: Pie chart de contenedores con mayor uso de CPU
7. **RAM Usada**: Stat panel con memoria actualmente en uso
8. **Panel Extra**: Top 5 contenedores con MENOR consumo de RAM

## Compilación e Instalación

### Requisitos del Sistema:
- Linux (Ubuntu/Debian recomendado)
- Kernel headers: `sudo apt-get install linux-headers-$(uname -r)`
- GCC y Make: `sudo apt-get install build-essential`
- Go 1.19+: Descarga desde https://golang.org/dl/
- Docker: `sudo apt-get install docker.io`
- SQLite3: `sudo apt-get install sqlite3`

### Pasos de Compilación:

#### 1. Módulos de Kernel:
```bash
cd modulo-kernel/
make clean
make all
```

#### 2. Cargar Módulos:
```bash
sudo make install
# O manualmente:
sudo insmod continfo_so1_202200214.ko
sudo insmod sysinfo_so1_202200214.ko
```

#### 3. Verificar Carga:
```bash
lsmod | grep so1
dmesg | tail -20
cat /proc/continfo_so1_202200214
cat /proc/sysinfo_so1_202200214
```

#### 4. Compilar Daemon Go:
```bash
cd go-daemon/
go mod tidy
go build -o mi-daemon main.go
```

#### 5. Construir Imágenes Docker:
```bash
chmod +x build_images.sh
./build_images.sh
```

#### 6. Ejecutar Sistema:
```bash
# Iniciar daemon (como root para cronjob)
sudo ./mi-daemon

# Verificar Grafana en http://localhost:3001
# Usuario: admin, Password: admin
```

### Desinstalación:
```bash
# Remover módulos
sudo rmmod continfo_so1_202200214
sudo rmmod sysinfo_so1_202200214

# Limpiar cronjob
crontab -l | grep -v 'create_containers.sh' | crontab -

# Detener contenedores
docker stop $(docker ps -aq)
```

## Anexos
![alt text](<WhatsApp Image 2025-09-24 at 8.20.14 PM.jpeg>)