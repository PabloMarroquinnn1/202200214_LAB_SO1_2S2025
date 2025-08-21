package main

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	nombre := "Pablo Alejandro Marroquin Cutz"
	carnet := "202200214"
	endPointGeneric := "/api2/202200214/"

	app := fiber.New(fiber.Config{
		AppName: "API2-VM1",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Endpoint raíz
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"mensaje":    fmt.Sprintf("API2 en la VM1 - Estudiante: %s, Carnet: %s", nombre, carnet),
			"api":        "API2",
			"vm":         "VM1",
			"estudiante": nombre,
			"carnet":     carnet,
			"endpoints": []string{
				"/api2/202200214/llamar-api1",
				"/api2/202200214/llamar-api3",
			},
		})
	})

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"api":    "API2",
			"vm":     "VM1",
			"carnet": carnet,
		})
	})

	// Endpoint para llamar a API1 (misma VM - localhost)
	app.Get(fmt.Sprintf("%sllamar-api1", endPointGeneric), func(c *fiber.Ctx) error {
		resp, err := http.Get("http://localhost:8080/")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":       fmt.Sprintf("Error conectando con API1: %v", err),
				"mensaje":     fmt.Sprintf("Error conectando con API1. Responde la API: API2 en la VM1, desarrollada por el estudiante %s con carnet: %s", nombre, carnet),
				"api_origen":  "API2",
				"vm_origen":   "VM1",
				"api_destino": "API1",
				"vm_destino":  "VM1",
			})
		}
		defer resp.Body.Close()

		buffer := make([]byte, 1024)
		n, _ := resp.Body.Read(buffer)
		apiResponse := string(buffer[:n])

		return c.JSON(fiber.Map{
			"mensaje":        fmt.Sprintf("Hola, responde la API: API2 en la VM1, desarrollada por el estudiante %s con carnet: %s. Conexión exitosa con API1", nombre, carnet),
			"api_origen":     "API2",
			"vm_origen":      "VM1",
			"api_destino":    "API1",
			"vm_destino":     "VM1",
			"respuesta_api1": apiResponse,
			"status":         "conexion_exitosa",
		})
	})

	// Endpoint para llamar a API3 (VM2)
	app.Get(fmt.Sprintf("%sllamar-api3", endPointGeneric), func(c *fiber.Ctx) error {
		// IP de VM2 donde está API3
		resp, err := http.Get("http://192.168.122.3:8082/")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":       fmt.Sprintf("Error conectando con API3: %v", err),
				"mensaje":     fmt.Sprintf("Error conectando con API3. Responde la API: API2 en la VM1, desarrollada por el estudiante %s con carnet: %s", nombre, carnet),
				"api_origen":  "API2",
				"vm_origen":   "VM1",
				"api_destino": "API3",
				"vm_destino":  "VM2",
			})
		}
		defer resp.Body.Close()

		buffer := make([]byte, 1024)
		n, _ := resp.Body.Read(buffer)
		apiResponse := string(buffer[:n])

		return c.JSON(fiber.Map{
			"mensaje":        fmt.Sprintf("Hola, responde la API: API2 en la VM1, desarrollada por el estudiante %s con carnet: %s. Conexión exitosa con API3", nombre, carnet),
			"api_origen":     "API2",
			"vm_origen":      "VM1",
			"api_destino":    "API3",
			"vm_destino":     "VM2",
			"respuesta_api3": apiResponse,
			"status":         "conexion_exitosa",
		})
	})

	fmt.Printf("🚀 API2 iniciada en puerto 8081\n")
	fmt.Printf("📋 Estudiante: %s\n", nombre)
	fmt.Printf("🎓 Carnet: %s\n", carnet)
	fmt.Printf("🔗 Endpoints disponibles:\n")
	fmt.Printf("   • GET / - Información de la API\n")
	fmt.Printf("   • GET /health - Health check\n")
	fmt.Printf("   • GET %sllamar-api1 - Llamar a API1\n", endPointGeneric)
	fmt.Printf("   • GET %sllamar-api3 - Llamar a API3\n", endPointGeneric)

	app.Listen("0.0.0.0:8081")
}
