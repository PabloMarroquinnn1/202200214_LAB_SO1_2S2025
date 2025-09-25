#!/usr/bin/env python3
import threading
import time
import os

def cpu_stress():
    print(f"Iniciando stress de CPU en thread {threading.current_thread().name}")
    
    while True:
        result = 0
        for i in range(1000000):
            result += i ** 2
            result = result % 999999
        
        for i in range(100000):
            result += (i * 3.14159) ** 0.5
            result = result % 999999
        
        time.sleep(0.001)

def main():
    print("=== CONTENEDOR PESADO CPU INICIADO ===")
    print(f"PID del proceso: {os.getpid()}")
    
    num_threads = os.cpu_count() or 4
    print(f"Creando {num_threads} threads para stress de CPU...")
    
    threads = []
    for i in range(num_threads):
        thread = threading.Thread(target=cpu_stress, name=f"CPUStress-{i}")
        thread.daemon = True
        thread.start()
        threads.append(thread)
    
    try:
        while True:
            print(f"Contenedor pesado CPU funcionando - Threads activos: {threading.active_count()}")
            time.sleep(30)
    except KeyboardInterrupt:
        print("Deteniendo contenedor pesado CPU...")

if __name__ == "__main__":
    main()
