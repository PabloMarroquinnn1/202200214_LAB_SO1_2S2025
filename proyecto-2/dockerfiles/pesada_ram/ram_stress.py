#!/usr/bin/env python3
import time
import gc
import os
import threading

def create_memory_chunks():
    chunks = []
    chunk_size = 50 * 1024 * 1024  # 50MB por chunk
    
    try:
        while True:
            chunk = bytearray(chunk_size)
            
            for i in range(0, chunk_size, 4096):
                chunk[i] = i % 256
            
            chunks.append(chunk)
            
            total_mb = len(chunks) * 50
            print(f"Memoria consumida: ~{total_mb}MB ({len(chunks)} chunks)")
            
            time.sleep(2)
            
            if len(chunks) > 20 and len(chunks) % 10 == 0:
                print("Liberando algunos chunks para crear variabilidad...")
                released = chunks[:5]
                chunks = chunks[5:]
                del released
                gc.collect()
                time.sleep(1)
                
    except MemoryError:
        print("¡MemoryError! Alcanzado límite de memoria")
        time.sleep(60)
    except Exception as e:
        print(f"Error inesperado: {e}")
        time.sleep(60)

def main():
    print("=== CONTENEDOR PESADO RAM INICIADO ===")
    print(f"PID del proceso: {os.getpid()}")
    
    thread1 = threading.Thread(target=create_memory_chunks, name="MemChunks")
    thread1.daemon = True
    thread1.start()
    
    try:
        while True:
            print(f"Contenedor pesado RAM funcionando")
            time.sleep(45)
    except KeyboardInterrupt:
        print("Deteniendo contenedor pesado RAM...")

if __name__ == "__main__":
    main()
