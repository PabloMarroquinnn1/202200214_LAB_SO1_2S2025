package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Estructura que mapea exactamente con el JSON del módulo de contenedores
type ContainerProc struct {
	Pid         int    `json:"pid"`
	Name        string `json:"name"`
	Cmdline     string `json:"cmdline"`
	ContainerID string `json:"container_id"`
	Vsz         uint64 `json:"vsz"`
	Rss         uint64 `json:"rss"`
	MemPerc     uint64 `json:"mem_perc"`
	CpuPerc     uint64 `json:"cpu_perc"`
}

// Estructura que mapea exactamente con el JSON del módulo de sistema
type SystemProc struct {
	Pid     int    `json:"pid"`
	Name    string `json:"name"`
	Cmdline string `json:"cmdline"`
	Vsz     uint64 `json:"vsz"`
	Rss     uint64 `json:"rss"`
	MemPerc uint64 `json:"mem_perc"`
	CpuPerc uint64 `json:"cpu_perc"`
	State   string `json:"state"`
}

type SysInfo struct {
	TotalRAM   uint64       `json:"totalram"`
	FreeRAM    uint64       `json:"freeram"`
	UsedRAM    uint64       `json:"usedram"`
	TotalProcs int          `json:"total_procs"`
	Processes  []SystemProc `json:"processes"`
}

type ContInfo struct {
	TotalRAM        uint64          `json:"totalram"`
	FreeRAM         uint64          `json:"freeram"`
	UsedRAM         uint64          `json:"usedram"`
	TotalContainers int             `json:"total_containers"`
	Containers      []ContainerProc `json:"containers"`
}

type Daemon struct {
	db                   *sql.DB
	containersEliminated int
	lowConsumptionCount  int
	highConsumptionCount int
}

func NewDaemon() (*Daemon, error) {
	// Crear la base de datos con permisos explícitos
	dbPath := "./metrics.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Verificar que podemos escribir a la base de datos
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	daemon := &Daemon{
		db: db,
	}

	if err := daemon.initDB(); err != nil {
		db.Close()
		return nil, err
	}

	log.Println("✅ Base de datos SQLite inicializada correctamente")
	return daemon, nil
}

func (d *Daemon) initDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS ram_usage (
			timestamp INTEGER, 
			used_ram INTEGER,
			total_ram INTEGER,
			free_ram INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS top5_ram (
			timestamp INTEGER, 
			pid INTEGER, 
			name TEXT, 
			cmdline TEXT, 
			container_id TEXT, 
			rss INTEGER, 
			mem_perc REAL
		)`,
		`CREATE TABLE IF NOT EXISTS top5_cpu (
			timestamp INTEGER, 
			pid INTEGER, 
			name TEXT, 
			cmdline TEXT, 
			container_id TEXT, 
			cpu_perc REAL
		)`,
		`CREATE TABLE IF NOT EXISTS containers_killed (
			timestamp INTEGER,
			count INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS system_processes (
			timestamp INTEGER,
			total_processes INTEGER,
			running_processes INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS top5_ram_low (
			timestamp INTEGER, 
			pid INTEGER, 
			name TEXT, 
			cmdline TEXT, 
			container_id TEXT, 
			rss INTEGER, 
			mem_perc REAL
		)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("error creating table: %v", err)
		}
	}

	return nil
}

func (d *Daemon) readSystemInfo() (*SysInfo, error) {
	data, err := os.ReadFile("/proc/sysinfo_so1_202200214")
	if err != nil {
		return nil, fmt.Errorf("error reading sysinfo: %v", err)
	}

	var info SysInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("error unmarshaling sysinfo JSON: %v", err)
	}

	return &info, nil
}

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

func (d *Daemon) buildDockerPIDMap() map[int]string {
	pidToName := make(map[int]string)

	// Ejecutar docker ps para obtener todos los nombres
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error obteniendo lista de contenedores: %v", err)
		return pidToName
	}

	names := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, name := range names {
		if name == "" {
			continue
		}

		// Obtener PID de cada contenedor
		pidCmd := exec.Command("docker", "inspect", name, "--format", "{{.State.Pid}}")
		pidOutput, err := pidCmd.Output()
		if err != nil {
			continue
		}

		pidStr := strings.TrimSpace(string(pidOutput))
		if pid, err := strconv.Atoi(pidStr); err == nil {
			pidToName[pid] = name
		}
	}

	log.Printf("Mapeado %d PIDs de Docker a nombres de contenedores", len(pidToName))
	return pidToName
}

func (d *Daemon) analyzeAndManageContainers(containers []ContainerProc) {
	// Lista de contenedores que NUNCA se deben considerar para eliminación
	protectedContainers := []string{
		"grafana", "clever_black", "oracle", "postgres", "mysql", "mongodb", "redis", "test-heavy",
	}

	// Construir mapeo PID → Nombre Docker real
	pidToDockerName := d.buildDockerPIDMap()

	// Filtrar contenedores protegidos y corregir nombres
	var managedContainers []ContainerProc
	var protectedCount int

	for _, container := range containers {
		// Usar el nombre real de Docker si está disponible
		realName := container.ContainerID
		if dockerName, exists := pidToDockerName[container.Pid]; exists {
			realName = dockerName
		} else {
			// Si no está en el mapeo, usar extractContainerName como fallback
			extracted := d.extractContainerName(container.Cmdline, container.ContainerID)
			if extracted != "" {
				realName = extracted
			}
		}

		// Verificar si está protegido
		isProtected := false
		if realName != "" {
			realNameLower := strings.ToLower(realName)
			for _, protected := range protectedContainers {
				if strings.Contains(realNameLower, strings.ToLower(protected)) {
					isProtected = true
					protectedCount++
					break
				}
			}
		}

		if !isProtected {
			// Actualizar el container con el nombre real
			container.ContainerID = realName
			managedContainers = append(managedContainers, container)
		}
	}

	// Categorizar contenedores gestionables
	var lightContainers []ContainerProc    // img_liviana:v1, img_liviana2:v1
	var heavyCPUContainers []ContainerProc // img_pesada_cpu:v1
	var heavyRAMContainers []ContainerProc // img_pesada_ram:v1

	for _, container := range managedContainers {
		containerType := d.identifyContainerType(container)

		switch containerType {
		case "liviana1", "liviana2":
			lightContainers = append(lightContainers, container)
		case "pesada_cpu":
			heavyCPUContainers = append(heavyCPUContainers, container)
		case "pesada_ram":
			heavyRAMContainers = append(heavyRAMContainers, container)
		default:
			// Clasificar por métricas si no se puede identificar el tipo
			if container.Rss > 100000 { // >100MB RAM
				heavyRAMContainers = append(heavyRAMContainers, container)
			} else if container.CpuPerc > 50 { // >50% CPU
				heavyCPUContainers = append(heavyCPUContainers, container)
			} else {
				lightContainers = append(lightContainers, container)
			}
		}
	}

	log.Printf("Contenedores: Protegidos: %d, Livianos: %d, CPU pesados: %d, RAM pesados: %d",
		protectedCount, len(lightContainers), len(heavyCPUContainers), len(heavyRAMContainers))

	// Aplicar reglas ESTRICTAS del proyecto
	d.lowConsumptionCount = len(lightContainers)
	d.highConsumptionCount = len(heavyCPUContainers) + len(heavyRAMContainers)

	containersToKill := 0

	// REGLA 1: EXACTAMENTE 3 contenedores livianos
	if len(lightContainers) > 3 {
		// Eliminar los más antiguos (últimos en la lista)
		excessLight := lightContainers[3:]
		containersToKill += len(excessLight)
		d.killContainersByType(excessLight, "liviano (mantener solo 3)")
		log.Printf("Eliminando %d contenedores livianos excedentes (máximo: 3)", len(excessLight))
	}

	// REGLA 2: EXACTAMENTE 2 contenedores pesados en total
	totalHeavy := len(heavyCPUContainers) + len(heavyRAMContainers)
	if totalHeavy > 2 {
		// Priorizar mantener los que menos recursos consumen para estabilidad
		var allHeavy []ContainerProc
		allHeavy = append(allHeavy, heavyCPUContainers...)
		allHeavy = append(allHeavy, heavyRAMContainers...)

		// Ordenar por consumo MENOR (mantener los más eficientes)
		sort.Slice(allHeavy, func(i, j int) bool {
			scoreI := float64(allHeavy[i].Rss)/1000 + float64(allHeavy[i].CpuPerc)
			scoreJ := float64(allHeavy[j].Rss)/1000 + float64(allHeavy[j].CpuPerc)
			return scoreI < scoreJ // MENOR consumo primero
		})

		excessHeavy := allHeavy[2:] // Eliminar desde el índice 2 en adelante
		containersToKill += len(excessHeavy)
		d.killContainersByType(excessHeavy, "pesado (mantener solo 2)")
		log.Printf("Eliminando %d contenedores pesados excedentes (máximo: 2)", len(excessHeavy))
	}

	// Actualizar contadores después de la limpieza
	d.lowConsumptionCount = min(len(lightContainers), 3)
	d.highConsumptionCount = min(totalHeavy, 2)

	if containersToKill > 0 {
		d.containersEliminated += containersToKill
		log.Printf("ACCION: Eliminados %d contenedores para cumplir reglas (total eliminados: %d)",
			containersToKill, d.containersEliminated)
	} else {
		log.Printf("ESTADO: Sistema cumple reglas - %d livianos (máx 3), %d pesados (máx 2), %d protegidos",
			len(lightContainers), totalHeavy, protectedCount)
	}
}

// Función auxiliar min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (d *Daemon) identifyContainerType(container ContainerProc) string {
	cmdline := strings.ToLower(container.Cmdline)

	// Identificar por cmdline
	if strings.Contains(cmdline, "/app.sh") {
		return "liviana1"
	}
	if strings.Contains(cmdline, "/app.js") || strings.Contains(cmdline, "node /app.js") {
		return "liviana2"
	}
	if strings.Contains(cmdline, "/cpu_stress.py") || strings.Contains(cmdline, "cpu_stress.py") {
		return "pesada_cpu"
	}
	if strings.Contains(cmdline, "/ram_stress.py") || strings.Contains(cmdline, "ram_stress.py") {
		return "pesada_ram"
	}

	// Identificar por nombre de contenedor
	if strings.Contains(container.ContainerID, "liviana1") {
		return "liviana1"
	}
	if strings.Contains(container.ContainerID, "liviana2") {
		return "liviana2"
	}
	if strings.Contains(container.ContainerID, "pesada_cpu") {
		return "pesada_cpu"
	}
	if strings.Contains(container.ContainerID, "pesada_ram") {
		return "pesada_ram"
	}

	return "unknown"
}

func (d *Daemon) killContainersByType(containers []ContainerProc, tipo string) {
	// Lista de contenedores que NUNCA se deben eliminar
	protectedContainers := []string{
		"grafana", "clever_black", "oracle", "postgres", "mysql", "mongodb", "redis", "test-heavy",
	}

	for _, container := range containers {
		containerName := container.ContainerID // Ahora ya tiene el nombre real de Docker

		if containerName != "" {
			// Verificar si está en la lista de protegidos (doble verificación)
			isProtected := false
			containerNameLower := strings.ToLower(containerName)

			for _, protected := range protectedContainers {
				if strings.Contains(containerNameLower, strings.ToLower(protected)) {
					isProtected = true
					break
				}
			}

			if isProtected {
				log.Printf("⛔ Saltando contenedor protegido: %s (PID: %d)", containerName, container.Pid)
				continue
			}

			log.Printf("🔴 Eliminando contenedor %s (PID: %d, tipo: %s)", containerName, container.Pid, tipo)

			// Intentar detener el contenedor con Docker usando el nombre real
			cmd := exec.Command("docker", "stop", containerName)
			if err := cmd.Run(); err != nil {
				log.Printf("⚠️  Error deteniendo contenedor %s: %v", containerName, err)

				// Si falla docker stop, intentar kill del proceso
				killCmd := exec.Command("kill", "-9", strconv.Itoa(container.Pid))
				if killErr := killCmd.Run(); killErr != nil {
					log.Printf("⚠️  Error matando proceso %d: %v", container.Pid, killErr)
				} else {
					log.Printf("✅ Proceso %d eliminado por kill", container.Pid)
				}
			} else {
				log.Printf("✅ Contenedor %s detenido exitosamente", containerName)
			}
		} else {
			log.Printf("⚠️  No se pudo determinar nombre del contenedor para PID %d", container.Pid)
		}
	}
}

func (d *Daemon) extractContainerName(cmdline, containerID string) string {
	// Intentar extraer nombre del container_id primero
	if containerID != "" && !strings.HasPrefix(containerID, "container_") && !strings.HasPrefix(containerID, "stress_container_") {
		return containerID
	}

	// Buscar patrones en cmdline
	if strings.Contains(cmdline, "ctn_") {
		parts := strings.Fields(cmdline)
		for _, part := range parts {
			if strings.Contains(part, "ctn_") {
				return part
			}
		}
	}

	// Buscar --name en cmdline
	if strings.Contains(cmdline, "--name") {
		parts := strings.Fields(cmdline)
		for i, p := range parts {
			if p == "--name" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}

	return ""
}

func (d *Daemon) saveMetrics(sysInfo *SysInfo, contInfo *ContInfo) {
	timestamp := time.Now().Unix()

	// Guardar uso de RAM del sistema
	_, err := d.db.Exec(`INSERT INTO ram_usage (timestamp, used_ram, total_ram, free_ram) VALUES (?, ?, ?, ?)`,
		timestamp, contInfo.UsedRAM, contInfo.TotalRAM, contInfo.FreeRAM)
	if err != nil {
		log.Printf("Error guardando ram_usage: %v", err)
	}

	// Guardar info de procesos del sistema
	runningProcs := 0
	for _, proc := range sysInfo.Processes {
		if proc.State == "R" {
			runningProcs++
		}
	}

	_, err = d.db.Exec(`INSERT INTO system_processes (timestamp, total_processes, running_processes) VALUES (?, ?, ?)`,
		timestamp, sysInfo.TotalProcs, runningProcs)
	if err != nil {
		log.Printf("Error guardando system_processes: %v", err)
	}

	// Guardar top 5 contenedores por RAM
	containersCopy := make([]ContainerProc, len(contInfo.Containers))
	copy(containersCopy, contInfo.Containers)

	sort.Slice(containersCopy, func(i, j int) bool {
		return containersCopy[i].Rss > containersCopy[j].Rss
	})

	for i := 0; i < 5 && i < len(containersCopy); i++ {
		p := containersCopy[i]
		// Convertir de décimas a porcentaje real para almacenamiento
		memPercReal := float64(p.MemPerc) / 10.0
		_, err = d.db.Exec(`INSERT INTO top5_ram (timestamp, pid, name, cmdline, container_id, rss, mem_perc) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			timestamp, p.Pid, p.Name, p.Cmdline, p.ContainerID, p.Rss, memPercReal)
		if err != nil {
			log.Printf("Error guardando top5_ram: %v", err)
		}
	}

	// Guardar top 5 contenedores por CPU
	sort.Slice(containersCopy, func(i, j int) bool {
		return containersCopy[i].CpuPerc > containersCopy[j].CpuPerc
	})

	for i := 0; i < 5 && i < len(containersCopy); i++ {
		p := containersCopy[i]
		// Convertir de décimas a porcentaje real para almacenamiento
		cpuPercReal := float64(p.CpuPerc) / 10.0
		_, err = d.db.Exec(`INSERT INTO top5_cpu (timestamp, pid, name, cmdline, container_id, cpu_perc) VALUES (?, ?, ?, ?, ?, ?)`,
			timestamp, p.Pid, p.Name, p.Cmdline, p.ContainerID, cpuPercReal)
		if err != nil {
			log.Printf("Error guardando top5_cpu: %v", err)
		}
	}

	// Guardar top 5 contenedores con MENOR consumo (gráfico extra)
	sort.Slice(containersCopy, func(i, j int) bool {
		return containersCopy[i].Rss < containersCopy[j].Rss
	})

	for i := 0; i < 5 && i < len(containersCopy); i++ {
		p := containersCopy[i]
		memPercReal := float64(p.MemPerc) / 10.0
		_, err = d.db.Exec(`INSERT INTO top5_ram_low (timestamp, pid, name, cmdline, container_id, rss, mem_perc) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			timestamp, p.Pid, p.Name, p.Cmdline, p.ContainerID, p.Rss, memPercReal)
		if err != nil {
			log.Printf("Error guardando top5_ram_low: %v", err)
		}
	}

	// Guardar contador de contenedores eliminados
	_, err = d.db.Exec(`INSERT INTO containers_killed (timestamp, count) VALUES (?, ?)`,
		timestamp, d.containersEliminated)
	if err != nil {
		log.Printf("Error guardando containers_killed: %v", err)
	}
}

func (d *Daemon) setupCronJob() error {
	log.Println("⏰ Configurando cronjob...")

	// Crear cronjob que se ejecute cada 3 minutos
	cronEntry := "*/3 * * * * /home/pablo/proyecto-2/bash/create_containers.sh >/dev/null 2>&1"

	cmd := exec.Command("bash", "-c", fmt.Sprintf("(crontab -l 2>/dev/null; echo '%s') | crontab -", cronEntry))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error configurando cronjob: %v", err)
	}

	log.Println("✅ Cronjob configurado exitosamente")
	return nil
}

func (d *Daemon) removeCronJob() {
	log.Println("🗑️  Removiendo cronjob...")

	cmd := exec.Command("bash", "-c", "crontab -l | grep -v 'create_containers.sh' | crontab -")
	if err := cmd.Run(); err != nil {
		log.Printf("⚠️  Error removiendo cronjob: %v", err)
	} else {
		log.Println("✅ Cronjob removido")
	}
}

func (d *Daemon) loadKernelModules() error {
	log.Println("🔧 Cargando módulos de kernel...")

	modules := []string{
		"/home/pablo/proyecto-2/modulo-kernel/continfo_so1_202200214.ko",
		"/home/pablo/proyecto-2/modulo-kernel/sysinfo_so1_202200214.ko",
	}

	for _, module := range modules {
		cmd := exec.Command("sudo", "insmod", module)
		if err := cmd.Run(); err != nil {
			log.Printf("⚠️  Módulo ya cargado o error: %v", err)
		}
	}

	log.Println("✅ Módulos de kernel listos")
	return nil
}

func (d *Daemon) setupGrafana() error {
	log.Println("🔧 Configurando Grafana...")

	// Verificar si ya existe
	checkCmd := exec.Command("docker", "ps", "-a", "--filter", "name=grafana-so1", "--format", "{{.Names}}")
	output, err := checkCmd.Output()

	if err == nil && strings.TrimSpace(string(output)) != "" {
		log.Println("✅ Grafana ya existe")

		// Verificar si está corriendo
		runningCmd := exec.Command("docker", "ps", "--filter", "name=grafana-so1", "--format", "{{.Names}}")
		runningOutput, _ := runningCmd.Output()

		if strings.TrimSpace(string(runningOutput)) == "" {
			// Iniciar contenedor existente
			startCmd := exec.Command("docker", "start", "grafana-so1")
			startCmd.Run()
			log.Println("✅ Grafana iniciado")
		}
		return nil
	}

	// Crear nuevo contenedor de Grafana
	createCmd := exec.Command("docker", "run", "-d",
		"--name", "grafana-so1",
		"-p", "3001:3000",
		"-e", "GF_SECURITY_ADMIN_PASSWORD=admin",
		"grafana/grafana:latest",
	)

	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("error creando Grafana: %v", err)
	}

	log.Println("✅ Grafana creado en puerto 3001")
	return nil
}

func (d *Daemon) mainLoop() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("📊 === INICIANDO CICLO DE MONITOREO ===")

		// Leer información del sistema
		sysInfo, err := d.readSystemInfo()
		if err != nil {
			log.Printf("❌ Error leyendo sysinfo: %v", err)
			continue
		}

		// Leer información de contenedores
		contInfo, err := d.readContainerInfo()
		if err != nil {
			log.Printf("❌ Error leyendo continfo: %v", err)
			continue
		}

		log.Printf("📈 Sistema: RAM total: %dKB, usada: %dKB, procesos: %d",
			sysInfo.TotalRAM, sysInfo.UsedRAM, sysInfo.TotalProcs)
		log.Printf("🐳 Contenedores detectados: %d", len(contInfo.Containers))

		// Analizar y gestionar contenedores
		if len(contInfo.Containers) > 0 {
			d.analyzeAndManageContainers(contInfo.Containers)
		}

		// Guardar métricas en la base de datos
		d.saveMetrics(sysInfo, contInfo)

		log.Printf("💾 Métricas guardadas. Contenedores eliminados total: %d", d.containersEliminated)
		log.Println("📊 === CICLO COMPLETADO ===")
		log.Println()
	}
}

func main() {
	log.Println("🚀 === DAEMON SO1 INICIANDO ===")
	log.Println("📋 Carné: 202200214")
	log.Println()

	// Crear daemon
	daemon, err := NewDaemon()
	if err != nil {
		log.Fatalf("❌ Error inicializando daemon: %v", err)
	}
	defer daemon.db.Close()

	// Configurar Grafana
	if err := daemon.setupGrafana(); err != nil {
		log.Printf("⚠️  Error configurando Grafana: %v", err)
	}

	// Cargar módulos de kernel
	if err := daemon.loadKernelModules(); err != nil {
		log.Printf("⚠️  Error cargando módulos: %v", err)
	}

	// Configurar cronjob
	if err := daemon.setupCronJob(); err != nil {
		log.Printf("⚠️  Error configurando cronjob: %v", err)
	}

	// Configurar manejo de señales
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar loop principal en goroutine
	go daemon.mainLoop()

	log.Println("✅ Daemon iniciado. Loop principal ejecutándose cada 20 segundos...")
	log.Println("🌐 Grafana disponible en: http://localhost:3001")
	log.Println("📝 Para detener: Ctrl+C")
	log.Println()

	// Esperar señal de terminación
	<-sigChan

	log.Println()
	log.Println("🛑 Señal de terminación recibida...")
	daemon.removeCronJob()
	log.Println("👋 Daemon detenido exitosamente")
}
